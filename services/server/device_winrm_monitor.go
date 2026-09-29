package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/masterzen/winrm"
	"github.com/semaphoreui/semaphore/db"
)

const (
	DeviceMonitorMaxDownloadBytes = 20 * 1024 * 1024
	DeviceMonitorFSPageSize       = 10
	DeviceMonitorDownloadChunk    = 512 * 1024
	DeviceMonitorDefaultTimeout   = 60
	DeviceMonitorDownloadTimeout  = 120
)

var (
	// Any single-letter drive root / absolute path. Network (shared) drives are
	// rejected at query time via Win32_LogicalDisk.DriveType != 4.
	deviceMonitorDriveRootRe = regexp.MustCompile(`(?i)^[a-z]:\\?$`)
	deviceMonitorAbsPathRe   = regexp.MustCompile(`(?i)^[a-z]:\\`)
)

// DeviceMonitorMetrics is host utilization from WinRM.
type DeviceMonitorMetrics struct {
	OK           bool                `json:"ok"`
	CPUPercent   float64             `json:"cpu_percent"`
	Memory       DeviceMonitorMemory `json:"memory"`
	Disks        []DeviceMonitorDisk `json:"disks"`
	DurationMS   int                 `json:"duration_ms"`
	ErrorCode    string              `json:"error,omitempty"`
	ErrorMessage string              `json:"message,omitempty"`
	ResolvedUser string              `json:"resolved_user"`
	ResolvedHost string              `json:"resolved_host"`
	ResolvedPort int                 `json:"resolved_port"`
}

// DeviceMonitorMemory is physical memory usage.
type DeviceMonitorMemory struct {
	TotalBytes  int64   `json:"total_bytes"`
	UsedBytes   int64   `json:"used_bytes"`
	FreeBytes   int64   `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// DeviceMonitorDisk is one fixed drive.
type DeviceMonitorDisk struct {
	Name        string  `json:"name"`
	TotalBytes  int64   `json:"total_bytes"`
	UsedBytes   int64   `json:"used_bytes"`
	FreeBytes   int64   `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// DeviceMonitorFSEntry is one file or directory row.
type DeviceMonitorFSEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"is_dir"`
	SizeBytes  int64  `json:"size_bytes"`
	ModifiedAt string `json:"modified_at"`
	IsHidden   bool   `json:"is_hidden"`
}

// DeviceMonitorFSList is a paginated directory listing.
type DeviceMonitorFSList struct {
	OK           bool                   `json:"ok"`
	Path         string                 `json:"path"`
	Page         int                    `json:"page"`
	PageSize     int                    `json:"page_size"`
	Total        int                    `json:"total"`
	Entries      []DeviceMonitorFSEntry `json:"entries"`
	Roots        []string               `json:"roots,omitempty"`
	DurationMS   int                    `json:"duration_ms"`
	ErrorCode    string                 `json:"error,omitempty"`
	ErrorMessage string                 `json:"message,omitempty"`
	ResolvedUser string                 `json:"resolved_user"`
	ResolvedHost string                 `json:"resolved_host"`
	ResolvedPort int                    `json:"resolved_port"`
}

// DeviceMonitorDownloadMeta is returned before streaming file bytes.
type DeviceMonitorDownloadMeta struct {
	Name      string
	SizeBytes int64
	Path      string
}

type monitorPSEnvelope struct {
	OK      bool            `json:"ok"`
	Error   string          `json:"error"`
	Msg     string          `json:"message"`
	CPU     float64         `json:"cpu_percent"`
	Mem     json.RawMessage `json:"memory"`
	Disks   json.RawMessage `json:"disks"`
	Path    string          `json:"path"`
	Page    int             `json:"page"`
	Total   int             `json:"total"`
	Entries json.RawMessage `json:"entries"`
	Roots   []string        `json:"roots"`
	Name    string          `json:"name"`
	Size    int64           `json:"size"`
	Done    bool            `json:"done"`
	Data    string          `json:"data"`
	Read    int             `json:"read"`
}

// SanitizeDeviceMonitorPath validates and normalizes a Windows drive path (A:\–Z:\).
// Empty path means drive-root listing mode. UNC / shared paths are rejected;
// network mapped drives are blocked in the remote PowerShell (DriveType=4).
func SanitizeDeviceMonitorPath(raw string) (string, error) {
	p := strings.TrimSpace(raw)
	p = strings.ReplaceAll(p, "/", "\\")
	if p == "" || p == "\\" {
		return "", nil
	}
	// Reject UNC and alternate streams / device paths.
	if strings.HasPrefix(p, "\\\\") || strings.Contains(p, "\x00") {
		return "", &db.ValidationError{Message: "path not allowed (UNC/shared paths unsupported)"}
	}
	if !deviceMonitorAbsPathRe.MatchString(p) && !deviceMonitorDriveRootRe.MatchString(p) {
		return "", &db.ValidationError{Message: "path must be a local drive path like E:\\data"}
	}
	// Normalize drive root forms: C: / C:\ → C:\
	if deviceMonitorDriveRootRe.MatchString(p) {
		drive := strings.ToUpper(p[:1])
		return drive + ":\\", nil
	}
	drive := strings.ToUpper(p[:1])
	rest := p[2:]
	if strings.HasPrefix(rest, "\\") {
		rest = rest[1:]
	}
	parts := strings.Split(rest, "\\")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", &db.ValidationError{Message: "path must not contain .."}
		}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		return drive + ":\\", nil
	}
	return drive + ":\\" + strings.Join(clean, "\\"), nil
}

func psSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func runMonitorPowerShell(ctx context.Context, creds DeviceWinRMExecCredentials, script string, timeoutSec int, maxStdout int) DeviceWinRMExecResult {
	res := DeviceWinRMExecResult{
		ResolvedUser: creds.User,
		ResolvedHost: creds.Host,
		ResolvedPort: creds.Port,
	}
	if timeoutSec <= 0 {
		timeoutSec = DeviceMonitorDefaultTimeout
	}
	if timeoutSec > DeviceMonitorDownloadTimeout {
		timeoutSec = DeviceMonitorDownloadTimeout
	}
	if maxStdout <= 0 {
		maxStdout = db.DeviceWinRMExecMaxResponseOut
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	start := time.Now()

	endpoint := winrm.NewEndpoint(
		creds.Host, creds.Port, creds.UseHTTPS, creds.InsecureSkipVerify,
		nil, nil, nil, time.Duration(timeoutSec)*time.Second,
	)
	params := winrm.DefaultParameters
	switch strings.ToLower(strings.TrimSpace(creds.Transport)) {
	case "ntlm", "credssp", "kerberos":
		params = &winrm.Parameters{
			TransportDecorator: func() winrm.Transporter { return &winrm.ClientNTLM{} },
		}
	}
	client, err := winrm.NewClientWithParameters(endpoint, creds.User, creds.Password, params)
	if err != nil {
		res.ErrorCode = "winrm_unreachable"
		res.ErrorMessage = err.Error()
		res.DurationMS = int(time.Since(start).Milliseconds())
		return res
	}
	stdout, stderr, exitCode, err := client.RunPSWithContextWithString(ctx, script, "")
	res.DurationMS = int(time.Since(start).Milliseconds())
	if len(stdout) > maxStdout {
		stdout = stdout[:maxStdout]
		res.OutputTruncated = true
	}
	if len(stderr) > maxStdout {
		stderr = stderr[:maxStdout]
		res.OutputTruncated = true
	}
	res.Stdout = stdout
	res.Stderr = stderr
	code := exitCode
	res.ExitCode = &code
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			res.ErrorCode = "command_timeout"
			res.ErrorMessage = "command timed out"
		} else if isWinRMAuthError(err) {
			res.ErrorCode = "winrm_auth_failed"
			res.ErrorMessage = sanitizeWinRMError(err)
		} else {
			res.ErrorCode = "winrm_unreachable"
			res.ErrorMessage = sanitizeWinRMError(err)
		}
		return res
	}
	res.OK = exitCode == 0
	if !res.OK && res.ErrorMessage == "" {
		res.ErrorMessage = fmt.Sprintf("exit code %d", exitCode)
	}
	return res
}

func parseMonitorJSON(stdout string) (monitorPSEnvelope, error) {
	raw := extractMonitorJSONObject(stdout)
	if raw == "" {
		return monitorPSEnvelope{}, fmt.Errorf("empty response")
	}
	var env monitorPSEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return monitorPSEnvelope{}, err
	}
	return env, nil
}

// extractMonitorJSONObject returns the first complete JSON object in s.
// PowerShell ConvertTo-Json sometimes wraps a single object as [{...}] or adds
// CLIXML noise; slicing from the last '{' can leave a trailing ']' and break Unmarshal.
func extractMonitorJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// GetDeviceMonitorMetrics queries CPU / memory / disk utilization.
func GetDeviceMonitorMetrics(ctx context.Context, creds DeviceWinRMExecCredentials) DeviceMonitorMetrics {
	out := DeviceMonitorMetrics{
		ResolvedUser: creds.User,
		ResolvedHost: creds.Host,
		ResolvedPort: creds.Port,
	}
	script := `
$ErrorActionPreference = 'Stop'
try {
  $cpu = 0.0
  $cpus = Get-CimInstance Win32_Processor -ErrorAction SilentlyContinue
  if ($cpus) { $cpu = [double](($cpus | Measure-Object -Property LoadPercentage -Average).Average) }
  $os = Get-CimInstance Win32_OperatingSystem
  $memTotal = [int64]$os.TotalVisibleMemorySize * 1024
  $memFree = [int64]$os.FreePhysicalMemory * 1024
  $memUsed = $memTotal - $memFree
  $memPct = 0.0
  if ($memTotal -gt 0) { $memPct = [math]::Round(($memUsed * 100.0) / $memTotal, 1) }
  $disks = @(Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | ForEach-Object {
    $total = [int64]($_.Size)
    $free = [int64]($_.FreeSpace)
    $used = $total - $free
    $pct = 0.0
    if ($total -gt 0) { $pct = [math]::Round(($used * 100.0) / $total, 1) }
    [pscustomobject]@{
      name = $_.DeviceID
      total_bytes = $total
      used_bytes = $used
      free_bytes = $free
      used_percent = $pct
    }
  })
  $payload = [ordered]@{
    ok = $true
    cpu_percent = [math]::Round($cpu, 1)
    memory = @{
      total_bytes = $memTotal
      used_bytes = $memUsed
      free_bytes = $memFree
      used_percent = $memPct
    }
    disks = @($disks)
  }
  # Single string to stdout (avoid pipeline wrapping as [{...}])
  $json = ($payload | ConvertTo-Json -Compress -Depth 6)
  [Console]::Out.Write($json)
} catch {
  $json = (@{ ok = $false; error = 'query_failed'; message = $_.Exception.Message } | ConvertTo-Json -Compress)
  [Console]::Out.Write($json)
}
`
	execRes := runMonitorPowerShell(ctx, creds, script, DeviceMonitorDefaultTimeout, db.DeviceWinRMExecMaxResponseOut)
	out.DurationMS = execRes.DurationMS
	if execRes.ErrorCode != "" {
		out.ErrorCode = execRes.ErrorCode
		out.ErrorMessage = execRes.ErrorMessage
		return out
	}
	env, err := parseMonitorJSON(execRes.Stdout)
	if err != nil {
		out.ErrorCode = "invalid_response"
		out.ErrorMessage = err.Error()
		if execRes.Stderr != "" {
			out.ErrorMessage = execRes.Stderr
		}
		return out
	}
	if !env.OK {
		out.ErrorCode = firstNonEmpty(env.Error, "query_failed")
		out.ErrorMessage = env.Msg
		return out
	}
	out.OK = true
	out.CPUPercent = env.CPU
	_ = json.Unmarshal(env.Mem, &out.Memory)
	_ = json.Unmarshal(env.Disks, &out.Disks)
	if out.Disks == nil {
		out.Disks = []DeviceMonitorDisk{}
	}
	return out
}

// ListDeviceMonitorFS lists a directory (or local drive roots) with server-side pagination.
// Roots exclude network/shared drives (Win32 DriveType=4). Page uses Skip/Take on the host.
func ListDeviceMonitorFS(ctx context.Context, creds DeviceWinRMExecCredentials, rawPath string, page int) DeviceMonitorFSList {
	out := DeviceMonitorFSList{
		PageSize:     DeviceMonitorFSPageSize,
		Entries:      []DeviceMonitorFSEntry{},
		ResolvedUser: creds.User,
		ResolvedHost: creds.Host,
		ResolvedPort: creds.Port,
	}
	if page < 1 {
		page = 1
	}
	out.Page = page

	clean, err := SanitizeDeviceMonitorPath(rawPath)
	if err != nil {
		out.ErrorCode = "invalid_path"
		out.ErrorMessage = err.Error()
		return out
	}
	out.Path = clean

	pathLit := psSingleQuote(clean)
	pageLit := strconv.Itoa(page)
	pageSizeLit := strconv.Itoa(DeviceMonitorFSPageSize)

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$path = %s
$page = %s
$pageSize = %s
function Test-SemLocalDrive([string]$p) {
  if ([string]::IsNullOrWhiteSpace($p) -or $p.Length -lt 2) { return $false }
  $id = $p.Substring(0,2).ToUpperInvariant()
  if ($id -notmatch '^[A-Z]:$') { return $false }
  $ld = Get-CimInstance Win32_LogicalDisk -Filter ("DeviceID='" + $id + "'") -ErrorAction SilentlyContinue
  if (-not $ld) { return $false }
  # 4 = Network Drive (mapped share) — not allowed
  return ($ld.DriveType -ne 4)
}
try {
  if ([string]::IsNullOrWhiteSpace($path)) {
    # All present drives except network/shared (DriveType=4)
    $roots = @(Get-CimInstance Win32_LogicalDisk -ErrorAction SilentlyContinue |
      Where-Object { $_.DriveType -ne 4 -and $_.DeviceID } |
      Sort-Object DeviceID |
      ForEach-Object {
        $root = $_.DeviceID + '\'
        if (Test-Path -LiteralPath $root) { $root }
      })
    $entries = @($roots | ForEach-Object {
      $di = Get-Item -LiteralPath $_ -Force
      [pscustomobject]@{
        name = $_.TrimEnd('\')
        path = $_
        is_dir = $true
        size_bytes = 0
        modified_at = $di.LastWriteTime.ToString('o')
        is_hidden = [bool]($di.Attributes -band [IO.FileAttributes]::Hidden)
      }
    })
    $total = $entries.Count
    $slice = @($entries | Select-Object -Skip (($page - 1) * $pageSize) -First $pageSize)
    (@{
      ok = $true
      path = ''
      page = $page
      total = $total
      roots = $roots
      entries = $slice
    } | ConvertTo-Json -Compress -Depth 6)
    return
  }
  if (-not (Test-SemLocalDrive $path)) {
    (@{ ok = $false; error = 'network_drive'; message = 'network/shared drives are not allowed'; path = $path } | ConvertTo-Json -Compress)
    return
  }
  if (-not (Test-Path -LiteralPath $path)) {
    (@{ ok = $false; error = 'not_found'; message = 'path not found'; path = $path } | ConvertTo-Json -Compress)
    return
  }
  $item = Get-Item -LiteralPath $path -Force
  if (-not $item.PSIsContainer) {
    (@{ ok = $false; error = 'not_directory'; message = 'path is not a directory'; path = $path } | ConvertTo-Json -Compress)
    return
  }
  # Server-side pagination: count + Skip/Take on the remote host (not full pull to Semaphore)
  $all = @(Get-ChildItem -LiteralPath $path -Force | Sort-Object { -not $_.PSIsContainer }, Name)
  $total = $all.Count
  $slice = @($all | Select-Object -Skip (($page - 1) * $pageSize) -First $pageSize | ForEach-Object {
    $full = $_.FullName
    if ($_.PSIsContainer -and -not $full.EndsWith('\')) { $full = $full + '\' }
    [pscustomobject]@{
      name = $_.Name
      path = $full
      is_dir = [bool]$_.PSIsContainer
      size_bytes = $(if ($_.PSIsContainer) { 0 } else { [int64]$_.Length })
      modified_at = $_.LastWriteTime.ToString('o')
      is_hidden = [bool]($_.Attributes -band [IO.FileAttributes]::Hidden)
    }
  })
  (@{
    ok = $true
    path = $path
    page = $page
    total = $total
    entries = $slice
  } | ConvertTo-Json -Compress -Depth 6)
} catch {
  (@{ ok = $false; error = 'list_failed'; message = $_.Exception.Message; path = $path } | ConvertTo-Json -Compress)
}
`, pathLit, pageLit, pageSizeLit)

	execRes := runMonitorPowerShell(ctx, creds, script, DeviceMonitorDefaultTimeout, db.DeviceWinRMExecMaxResponseOut)
	out.DurationMS = execRes.DurationMS
	if execRes.ErrorCode != "" {
		out.ErrorCode = execRes.ErrorCode
		out.ErrorMessage = execRes.ErrorMessage
		return out
	}
	env, err := parseMonitorJSON(execRes.Stdout)
	if err != nil {
		out.ErrorCode = "invalid_response"
		out.ErrorMessage = err.Error()
		return out
	}
	if !env.OK {
		out.ErrorCode = firstNonEmpty(env.Error, "list_failed")
		out.ErrorMessage = env.Msg
		return out
	}
	out.OK = true
	out.Total = env.Total
	out.Roots = env.Roots
	if env.Path != "" {
		out.Path = env.Path
	}
	_ = json.Unmarshal(env.Entries, &out.Entries)
	if out.Entries == nil {
		out.Entries = []DeviceMonitorFSEntry{}
	}
	return out
}

// DownloadDeviceMonitorFile streams a file ≤20MB to w via chunked WinRM reads.
// onMeta is called once with file metadata before any bytes are written (may be nil).
func DownloadDeviceMonitorFile(
	ctx context.Context,
	creds DeviceWinRMExecCredentials,
	rawPath string,
	onMeta func(DeviceMonitorDownloadMeta) error,
	w io.Writer,
) (DeviceMonitorDownloadMeta, error) {
	meta := DeviceMonitorDownloadMeta{}
	clean, err := SanitizeDeviceMonitorPath(rawPath)
	if err != nil {
		return meta, err
	}
	if clean == "" || deviceMonitorDriveRootRe.MatchString(clean) {
		return meta, &db.ValidationError{Message: "path must be a file on a local drive"}
	}

	var offset int64
	metaReady := false
	for {
		chunk, done, fileSize, fileName, err := readMonitorFileChunk(ctx, creds, clean, offset, DeviceMonitorDownloadChunk)
		if err != nil {
			return meta, err
		}
		if !metaReady {
			meta.Name = fileName
			meta.SizeBytes = fileSize
			meta.Path = clean
			metaReady = true
			if fileSize > DeviceMonitorMaxDownloadBytes {
				return meta, &db.ValidationError{Message: fmt.Sprintf("file exceeds %d bytes", DeviceMonitorMaxDownloadBytes)}
			}
			if onMeta != nil {
				if err := onMeta(meta); err != nil {
					return meta, err
				}
			}
			if fileSize == 0 {
				return meta, nil
			}
		}
		if len(chunk) > 0 {
			if _, err := w.Write(chunk); err != nil {
				return meta, err
			}
			offset += int64(len(chunk))
		}
		if done || offset >= meta.SizeBytes {
			break
		}
		if len(chunk) == 0 {
			return meta, fmt.Errorf("empty chunk at offset %d", offset)
		}
	}
	return meta, nil
}

func readMonitorFileChunk(ctx context.Context, creds DeviceWinRMExecCredentials, filePath string, offset int64, length int) (data []byte, done bool, size int64, name string, err error) {
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$path = %s
$offset = [int64]%d
$length = [int]%d
$max = [int64]%d
try {
  $driveId = $path.Substring(0,2).ToUpperInvariant()
  $ld = Get-CimInstance Win32_LogicalDisk -Filter ("DeviceID='" + $driveId + "'") -ErrorAction SilentlyContinue
  if (-not $ld -or $ld.DriveType -eq 4) {
    (@{ ok = $false; error = 'network_drive'; message = 'network/shared drives are not allowed' } | ConvertTo-Json -Compress)
    return
  }
  if (-not (Test-Path -LiteralPath $path)) {
    (@{ ok = $false; error = 'not_found'; message = 'file not found' } | ConvertTo-Json -Compress)
    return
  }
  $item = Get-Item -LiteralPath $path -Force
  if ($item.PSIsContainer) {
    (@{ ok = $false; error = 'is_directory'; message = 'path is a directory' } | ConvertTo-Json -Compress)
    return
  }
  $size = [int64]$item.Length
  if ($size -gt $max) {
    (@{ ok = $false; error = 'too_large'; message = ('file exceeds ' + $max + ' bytes'); size = $size; name = $item.Name } | ConvertTo-Json -Compress)
    return
  }
  if ($offset -ge $size) {
    (@{ ok = $true; done = $true; size = $size; name = $item.Name; read = 0; data = '' } | ConvertTo-Json -Compress)
    return
  }
  $fs = [System.IO.File]::Open($item.FullName, [System.IO.FileMode]::Open, [System.IO.FileAccess]::Read, [System.IO.FileShare]::ReadWrite)
  try {
    $null = $fs.Seek($offset, [System.IO.SeekOrigin]::Begin)
    $toRead = [Math]::Min([int64]$length, ($size - $offset))
    $buf = New-Object byte[] $toRead
    $n = $fs.Read($buf, 0, $toRead)
    if ($n -le 0) {
      (@{ ok = $true; done = $true; size = $size; name = $item.Name; read = 0; data = '' } | ConvertTo-Json -Compress)
      return
    }
    if ($n -lt $buf.Length) { $buf = $buf[0..($n-1)] }
    $b64 = [Convert]::ToBase64String($buf)
    $done = (($offset + $n) -ge $size)
    (@{ ok = $true; done = $done; size = $size; name = $item.Name; read = $n; data = $b64 } | ConvertTo-Json -Compress)
  } finally {
    $fs.Dispose()
  }
} catch {
  (@{ ok = $false; error = 'read_failed'; message = $_.Exception.Message } | ConvertTo-Json -Compress)
}
`, psSingleQuote(filePath), offset, length, DeviceMonitorMaxDownloadBytes)

	// Base64 of 512KiB ≈ 700KiB; allow 1.5MiB stdout.
	execRes := runMonitorPowerShell(ctx, creds, script, DeviceMonitorDownloadTimeout, 2*1024*1024)
	if execRes.ErrorCode != "" {
		return nil, false, 0, "", fmt.Errorf("%s: %s", execRes.ErrorCode, execRes.ErrorMessage)
	}
	env, err := parseMonitorJSON(execRes.Stdout)
	if err != nil {
		return nil, false, 0, "", err
	}
	if !env.OK {
		msg := firstNonEmpty(env.Msg, env.Error, "read_failed")
		if env.Error == "too_large" {
			return nil, false, env.Size, env.Name, &db.ValidationError{Message: msg}
		}
		if env.Error == "not_found" {
			return nil, false, 0, "", &db.ValidationError{Message: msg}
		}
		return nil, false, env.Size, env.Name, fmt.Errorf("%s", msg)
	}
	if env.Data == "" {
		return nil, env.Done, env.Size, env.Name, nil
	}
	raw, err := base64.StdEncoding.DecodeString(env.Data)
	if err != nil {
		return nil, false, env.Size, env.Name, fmt.Errorf("invalid base64 chunk: %w", err)
	}
	return raw, env.Done, env.Size, env.Name, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ParentDeviceMonitorPath returns the parent directory for breadcrumbs, or "" at roots.
func ParentDeviceMonitorPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	clean, err := SanitizeDeviceMonitorPath(p)
	if err != nil || clean == "" {
		return ""
	}
	if deviceMonitorDriveRootRe.MatchString(clean) {
		return ""
	}
	parent := path.Dir(strings.ReplaceAll(clean, "\\", "/"))
	parent = strings.ReplaceAll(parent, "/", "\\")
	if deviceMonitorDriveRootRe.MatchString(parent) || len(parent) == 2 {
		drive := strings.ToUpper(clean[:1])
		return drive + ":\\"
	}
	out, err := SanitizeDeviceMonitorPath(parent)
	if err != nil {
		return ""
	}
	return out
}

// SafeDownloadFilename strips path components for Content-Disposition.
func SafeDownloadFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	name = strings.Trim(name, ". ")
	if name == "" || name == "." || name == "/" {
		return "download.bin"
	}
	if !utf8.ValidString(name) {
		return "download.bin"
	}
	return name
}
