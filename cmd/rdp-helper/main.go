// Package main is the Semaphore Native RDP Helper (Windows).
// See docs/plan-rdp-helper.md.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	appName        = "SemaphoreRdpHelper"
	protocolScheme = "semaphore-rdp"
	configFileName = "config.json"
	stateFileName  = "state.json"
	logFileName    = "helper.log"
)

type Hop struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	User string `json:"user"`
}

type Environment struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SemaphoreURL  string `json:"semaphore_url"` // browser URL after connect (often http://127.0.0.1:PORT)
	APIBaseURL    string `json:"api_base_url"`  // optional override for launch-params; else SemaphoreURL or protocol base=
	UILocalPort   int    `json:"ui_local_port"`
	UIRemoteHost  string `json:"ui_remote_host"`
	UIRemotePort  int    `json:"ui_remote_port"`
	ForwardUI     bool   `json:"forward_ui"`
	Hops          []Hop  `json:"hops"`
	LandHost      string `json:"land_host"` // empty = last hop host or empty for direct
	LandUser      string `json:"land_user"`
	LandPort      int    `json:"land_port"`
	SSHIdentity   string `json:"ssh_identity,omitempty"` // optional -i path
}

type Config struct {
	Environments []Environment `json:"environments"`
	ActiveEnv    string        `json:"active_env"`
}

type State struct {
	ConnectedEnv string `json:"connected_env"`
	SSHPid       int    `json:"ssh_pid,omitempty"`
	SocksPort    int    `json:"socks_port,omitempty"`
	LandTarget   string `json:"land_target,omitempty"` // user@host (informational)
	// Legacy ControlMaster fields (ignored; Win32-OpenSSH does not support mux).
	ControlPath string `json:"control_path,omitempty"`
}

type launchParams struct {
	ProjectID        int     `json:"project_id"`
	DeviceID         int     `json:"device_id"`
	Host             string  `json:"host"`
	RDPPort          int     `json:"rdp_port"`
	RDPUser          string  `json:"rdp_user"`
	RDPPassword      *string `json:"rdp_password"`
	PasswordProvided bool    `json:"password_provided"`
}

func main() {
	if len(os.Args) < 2 {
		if err := cmdUI(); err != nil {
			logf("ui error: %v", err)
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	arg := os.Args[1]
	if strings.HasPrefix(strings.ToLower(arg), protocolScheme+":") {
		if err := handleProtocolURL(arg); err != nil {
			logf("protocol error: %v", err)
			notifyUser("RDP Helper", err.Error())
			os.Exit(1)
		}
		return
	}
	cmd := strings.ToLower(arg)
	var err error
	switch cmd {
	case "help", "-h", "--help":
		printHelp()
	case "ui":
		err = cmdUI()
	case "install":
		err = cmdInstall()
	case "status":
		err = cmdStatus()
	case "connect":
		envID := ""
		if len(os.Args) > 2 {
			envID = os.Args[2]
		}
		err = cmdConnect(envID)
	case "disconnect":
		err = cmdDisconnect()
	case "open":
		err = cmdOpen()
	case "envs":
		err = cmdEnvs()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", arg)
		printHelp()
		os.Exit(2)
	}
	if err != nil {
		logf("error: %v", err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`Semaphore RDP Helper

Usage:
  %s                       Open local web panel (http://127.0.0.1:17300)
  %s ui                    Same as no-args
  %s install               Register %s:// (HKCU) and create config dir
  %s envs                  List environments from config
  %s connect [env-id]      SSH tunnel (-N + SOCKS; optional UI -L)
  %s disconnect            Tear down SSH tunnel
  %s open                  Open Semaphore URL in browser
  %s status                Show connection state
  %s %s://connect?token=...  Handle protocol (used by OS)

Config: %%LOCALAPPDATA%%\%s\%s
`, os.Args[0], os.Args[0], os.Args[0], protocolScheme, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], protocolScheme, appName, configFileName)
}

func appDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func configPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

func statePath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

func logPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, logFileName), nil
}

func logf(format string, args ...any) {
	p, err := logPath()
	if err != nil {
		return
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	line := time.Now().Format(time.RFC3339) + " " + fmt.Sprintf(format, args...) + "\n"
	_, _ = f.WriteString(line)
}

func loadConfig() (Config, error) {
	p, err := configPath()
	if err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func saveConfig(c Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func defaultConfig() Config {
	return Config{
		Environments: []Environment{{
			ID:           "my-project",
			Name:         "我的项目",
			SemaphoreURL: "http://127.0.0.1:3000",
			UILocalPort:  3000,
			UIRemoteHost: "",
			UIRemotePort: 3000,
			ForwardUI:    true,
			Hops:         []Hop{{Host: "", Port: 22, User: ""}},
			LandUser:     "",
		}},
		ActiveEnv: "my-project",
	}
}

func ensureConfig() (Config, error) {
	p, err := configPath()
	if err != nil {
		return Config{}, err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		c := defaultConfig()
		if err := saveConfig(c); err != nil {
			return Config{}, err
		}
		return c, nil
	}
	return loadConfig()
}

func loadState() (State, error) {
	p, err := statePath()
	if err != nil {
		return State{}, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return State{}, err
	}
	return s, nil
}

func saveState(s State) error {
	p, err := statePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func findEnv(c Config, id string) (Environment, bool) {
	if id == "" {
		id = c.ActiveEnv
	}
	for _, e := range c.Environments {
		if e.ID == id {
			return e, true
		}
	}
	if len(c.Environments) == 1 {
		return c.Environments[0], true
	}
	return Environment{}, false
}

func cmdInstall() error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}
	_ = c
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := registerProtocol(exe); err != nil {
		return fmt.Errorf("register protocol: %w", err)
	}
	p, _ := configPath()
	fmt.Printf("Registered %s://\nConfig: %s\nEdit environments in config.json, then: connect / open\n", protocolScheme, p)
	logf("install ok exe=%s", exe)
	return nil
}

func cmdEnvs() error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}
	s, _ := loadState()
	for _, e := range c.Environments {
		mark := " "
		if e.ID == c.ActiveEnv {
			mark = "*"
		}
		conn := ""
		if s.ConnectedEnv == e.ID {
			conn = " [connected]"
		}
		fmt.Printf("%s %s (%s) hops=%d%s\n", mark, e.ID, e.Name, len(e.Hops), conn)
	}
	return nil
}

func cmdStatus() error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}
	s, _ := loadState()
	fmt.Printf("active_env=%s connected=%s ssh_pid=%d socks_port=%d\n", c.ActiveEnv, s.ConnectedEnv, s.SSHPid, s.SocksPort)
	if envNeedsSSH(mustFindEnv(c, s.ConnectedEnv, c.ActiveEnv)) || s.SocksPort > 0 || s.SSHPid > 0 {
		if sessionAlive(s) {
			fmt.Println("ssh_session: alive")
		} else {
			fmt.Println("ssh_session: dead (click Connect in the helper panel)")
		}
	}
	return nil
}

func mustFindEnv(c Config, ids ...string) Environment {
	for _, id := range ids {
		if e, ok := findEnv(c, id); ok {
			return e
		}
	}
	return Environment{}
}

func envNeedsSSH(env Environment) bool {
	return len(env.Hops) > 0 || strings.TrimSpace(env.LandHost) != ""
}

func sessionAlive(s State) bool {
	if s.SocksPort <= 0 {
		return false
	}
	if s.SSHPid > 0 && !processExists(s.SSHPid) {
		return false
	}
	return tcpReachable("127.0.0.1", s.SocksPort, 500*time.Millisecond)
}

func waitSOCKSReady(cmd *exec.Cmd, socksPort int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return fmt.Errorf("ssh exited before tunnel was ready (exit=%s)", cmd.ProcessState.String())
		}
		if cmd.Process != nil && !processExists(cmd.Process.Pid) {
			_ = cmd.Wait()
			if cmd.ProcessState != nil {
				return fmt.Errorf("ssh exited before tunnel was ready (exit=%s)", cmd.ProcessState.String())
			}
			return fmt.Errorf("ssh exited before tunnel was ready")
		}
		if tcpReachable("127.0.0.1", socksPort, 200*time.Millisecond) {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for SOCKS on 127.0.0.1:%d (check password / network)", socksPort)
}

// ensureConnected makes sure env is ready for protocol launch (auto-connect if needed).
func ensureConnected(env Environment) (State, error) {
	s, _ := loadState()
	if !envNeedsSSH(env) {
		if s.ConnectedEnv != env.ID {
			s = State{ConnectedEnv: env.ID}
			if err := saveState(s); err != nil {
				return s, err
			}
		}
		return s, nil
	}
	if s.ConnectedEnv == env.ID && sessionAlive(s) {
		return s, nil
	}
	logf("auto-connect env=%s (use this console for SSH password if prompted)", env.ID)
	if err := cmdConnect(env.ID); err != nil {
		return s, fmt.Errorf("%w — or open the helper panel and click 连接 first", err)
	}
	return loadState()
}

func sanitizeID(s string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return r.Replace(s)
}

func landSpec(env Environment) (user, host string, port int, err error) {
	user = strings.TrimSpace(env.LandUser)
	host = strings.TrimSpace(env.LandHost)
	port = env.LandPort
	if host == "" && len(env.Hops) > 0 {
		// Land = last hop. Always take that hop's port — do not keep a stale
		// default land_port=22 from the UI when land_host is empty.
		last := env.Hops[len(env.Hops)-1]
		host = strings.TrimSpace(last.Host)
		if user == "" {
			user = strings.TrimSpace(last.User)
		}
		port = last.Port
	}
	if port == 0 {
		port = 22
	}
	if host == "" && len(env.Hops) == 0 {
		// direct / no SSH land — OK for connect that only opens local URL
		return user, "", 0, nil
	}
	if host == "" {
		return "", "", 0, fmt.Errorf("environment %q: set land_host or hops", env.ID)
	}
	return user, host, port, nil
}

func proxyJumpArg(env Environment) string {
	if len(env.Hops) == 0 {
		return ""
	}
	// If land is last hop, ProxyJump is hops[0..n-2]; if land separate, all hops are jumps.
	hops := env.Hops
	landHost := strings.TrimSpace(env.LandHost)
	if landHost == "" && len(hops) > 0 {
		hops = hops[:len(hops)-1]
	}
	if len(hops) == 0 {
		return ""
	}
	parts := make([]string, 0, len(hops))
	for _, h := range hops {
		u := h.User
		if u == "" {
			u = env.LandUser
		}
		p := h.Port
		if p == 0 {
			p = 22
		}
		// user@host:port — OpenSSH ProxyJump accepts user@host:port
		parts = append(parts, fmt.Sprintf("%s@%s:%d", u, h.Host, p))
	}
	return strings.Join(parts, ",")
}

func cmdConnect(envID string) error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}
	env, ok := findEnv(c, envID)
	if !ok {
		return fmt.Errorf("environment not found (edit config.json)")
	}
	c.ActiveEnv = env.ID
	_ = saveConfig(c)

	user, host, port, err := landSpec(env)
	if err != nil {
		return err
	}

	// No SSH needed
	if host == "" {
		_ = cmdDisconnectQuiet()
		s := State{ConnectedEnv: env.ID}
		if err := saveState(s); err != nil {
			return err
		}
		fmt.Printf("connected env=%s (direct, no SSH)\n", env.ID)
		logf("connect direct env=%s", env.ID)
		return nil
	}

	_ = cmdDisconnectQuiet()

	target := host
	if user != "" {
		target = user + "@" + host
	}

	socksPort, err := pickLocalPort()
	if err != nil {
		return err
	}

	// Win32-OpenSSH does NOT support ControlMaster (getsockname / Bad file descriptor).
	// Keep one long-lived `ssh -N` with DynamicForward (SOCKS); RDP uses that SOCKS.
	args := []string{
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-D", fmt.Sprintf("127.0.0.1:%d", socksPort),
		"-p", strconv.Itoa(port),
	}
	if id := strings.TrimSpace(env.SSHIdentity); id != "" {
		args = append(args, "-i", id)
	}
	if pj := proxyJumpArg(env); pj != "" {
		args = append(args, "-J", pj)
	}
	if env.ForwardUI {
		lport := env.UILocalPort
		if lport == 0 {
			lport = 3000
		}
		rhost := env.UIRemoteHost
		if rhost == "" {
			rhost = "127.0.0.1"
		}
		rport := env.UIRemotePort
		if rport == 0 {
			rport = 3000
		}
		args = append(args, "-L", fmt.Sprintf("%d:%s:%d", lport, rhost, rport))
	}
	args = append(args, target)

	fmt.Println("Connecting via SSH (enter password in this console if prompted)...")
	fmt.Printf("ssh target %s -p %d", target, port)
	if pj := proxyJumpArg(env); pj != "" {
		fmt.Printf(" -J %s", pj)
	}
	fmt.Println()
	logf("ssh connect start env=%s land=%s port=%d jump=%s socks=%d", env.ID, target, port, proxyJumpArg(env), socksPort)
	cmd, err := startSSHTunnel(args)
	if err != nil {
		return fmt.Errorf("ssh start failed: %w", err)
	}
	// Allow time for multi-hop password prompts.
	if err := waitSOCKSReady(cmd, socksPort, 3*time.Minute); err != nil {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
		return fmt.Errorf("ssh connect failed: %w", err)
	}
	go func() { _ = cmd.Wait() }()

	s := State{
		ConnectedEnv: env.ID,
		SSHPid:       cmd.Process.Pid,
		SocksPort:    socksPort,
		LandTarget:   target,
	}
	if err := saveState(s); err != nil {
		killProcess(cmd.Process.Pid)
		return err
	}
	fmt.Printf("connected env=%s land=%s socks=127.0.0.1:%d pid=%d\n", env.ID, target, socksPort, cmd.Process.Pid)
	logf("connect ok env=%s land=%s socks=%d pid=%d", env.ID, target, socksPort, cmd.Process.Pid)
	return nil
}

func cmdDisconnectQuiet() error {
	s, _ := loadState()
	if s.SSHPid > 0 {
		killProcess(s.SSHPid)
	}
	_ = saveState(State{})
	return nil
}

func cmdDisconnect() error {
	s, err := loadState()
	if err != nil {
		return err
	}
	if s.SSHPid == 0 && s.SocksPort == 0 && s.ConnectedEnv == "" {
		fmt.Println("not connected")
		return nil
	}
	if s.SSHPid > 0 {
		killProcess(s.SSHPid)
	}
	_ = saveState(State{})
	fmt.Println("disconnected")
	logf("disconnect ok")
	return nil
}

func cmdOpen() error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}
	env, ok := findEnv(c, c.ActiveEnv)
	if !ok {
		return fmt.Errorf("no active environment")
	}
	u := strings.TrimSpace(env.SemaphoreURL)
	if u == "" {
		return fmt.Errorf("semaphore_url empty for env %s", env.ID)
	}
	return openBrowser(u)
}

func handleProtocolURL(raw string) error {
	logf("protocol url received")
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	q := u.Query()
	token := strings.TrimSpace(q.Get("token"))
	if token == "" {
		return fmt.Errorf("missing token")
	}
	base := strings.TrimSpace(q.Get("base"))

	c, err := ensureConfig()
	if err != nil {
		return err
	}
	s, _ := loadState()
	env, ok := findEnv(c, s.ConnectedEnv)
	if !ok {
		env, ok = findEnv(c, c.ActiveEnv)
	}
	if !ok {
		return fmt.Errorf("no environment configured; open the helper panel, save a project, then retry")
	}

	s, err = ensureConnected(env)
	if err != nil {
		return fmt.Errorf("auto-connect %s failed: %w", env.ID, err)
	}

	apiBase := strings.TrimRight(base, "/")
	if apiBase == "" {
		apiBase = strings.TrimRight(strings.TrimSpace(env.APIBaseURL), "/")
	}
	if apiBase == "" {
		apiBase = strings.TrimRight(strings.TrimSpace(env.SemaphoreURL), "/")
	}
	if apiBase == "" {
		return fmt.Errorf("no API base URL (protocol base= or env semaphore_url)")
	}

	params, err := fetchLaunchParams(apiBase, token)
	if err != nil {
		return err
	}
	logf("launch-params host=%s port=%d user=%s password_provided=%v", params.Host, params.RDPPort, params.RDPUser, params.PasswordProvided)

	targetHost := params.Host
	targetPort := params.RDPPort
	if targetPort == 0 {
		targetPort = 3389
	}

	useTunnel := false
	localPort := 0
	sessionOK := sessionAlive(s)
	if sessionOK {
		if !tcpReachable(targetHost, targetPort, 800*time.Millisecond) {
			useTunnel = true
		}
	} else if !tcpReachable(targetHost, targetPort, 800*time.Millisecond) {
		return fmt.Errorf("host %s:%d unreachable and no SSH session; connect project %s in the helper panel", targetHost, targetPort, env.ID)
	}

	mstscHost := targetHost
	mstscPort := targetPort
	if useTunnel {
		lp, err := pickLocalPort()
		if err != nil {
			return err
		}
		localPort = lp
		stop, err := startLocalForwardViaSOCKS("127.0.0.1", s.SocksPort, localPort, targetHost, targetPort)
		if err != nil {
			return fmt.Errorf("socks forward failed: %w", err)
		}
		defer stop()
		mstscHost = "127.0.0.1"
		mstscPort = localPort
		logf("socks forward 127.0.0.1:%d -> %s:%d via socks :%d", localPort, targetHost, targetPort, s.SocksPort)
	}

	pass := ""
	if params.PasswordProvided && params.RDPPassword != nil {
		pass = *params.RDPPassword
	}
	if err := launchMSTSC(mstscHost, mstscPort, params.RDPUser, pass); err != nil {
		return err
	}
	return nil
}

func fetchLaunchParams(apiBase, token string) (launchParams, error) {
	u := strings.TrimRight(apiBase, "/") + "/api/rdp/launch-params?token=" + url.QueryEscape(token)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return launchParams{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return launchParams{}, fmt.Errorf("launch-params HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var p launchParams
	if err := json.Unmarshal(body, &p); err != nil {
		return launchParams{}, err
	}
	return p, nil
}

func tcpReachable(host string, port int, timeout time.Duration) bool {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func pickLocalPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func launchMSTSC(host string, port int, user, password string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("mstsc only supported on Windows (got %s)", runtime.GOOS)
	}
	target := fmt.Sprintf("%s:%d", host, port)
	termsrv := fmt.Sprintf("TERMSRV/%s", target)

	if password != "" && user != "" {
		_ = exec.Command("cmdkey", "/delete:"+termsrv).Run()
		cmd := exec.Command("cmdkey", "/generic:"+termsrv, "/user:"+user, "/pass:"+password)
		if out, err := cmd.CombinedOutput(); err != nil {
			logf("cmdkey failed: %v %s", err, string(out))
			// continue; mstsc will prompt
		} else {
			defer func() {
				_ = exec.Command("cmdkey", "/delete:"+termsrv).Run()
			}()
		}
	}

	args := []string{"/v:" + target}
	if user != "" && password == "" {
		// username only — mstsc prompts for password
		args = append(args, "/u:"+user)
	}
	cmd := exec.Command("mstsc", args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mstsc: %w", err)
	}
	logf("mstsc started target=%s user=%s", target, user)
	_ = cmd.Wait()
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}

func notifyUser(title, msg string) {
	if runtime.GOOS != "windows" {
		fmt.Fprintf(os.Stderr, "%s: %s\n", title, msg)
		return
	}
	ps := fmt.Sprintf(
		`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('%s','%s')`,
		escapePS(msg), escapePS(title),
	)
	_ = exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps).Run()
}

func escapePS(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
