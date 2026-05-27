package tdengine

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds TDengine REST connection settings.
type Config struct {
	Enabled  bool   `json:"enabled"`
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	// AutoSyncOnBulk writes a full project snapshot to TDengine after each
	// playbook bulk status callback. Default false — use manual publish or enable explicitly.
	AutoSyncOnBulk bool `json:"auto_sync_on_bulk,omitempty"`
}

// StatusRow is one device snapshot row for the status table.
type StatusRow struct {
	ProjectID        int
	DeviceID         int
	Hostname         string
	IP               string
	Status           string // online | offline
	DeviceStatusRaw  string
	WinRMStatus      string
	APIStatus        string
}

// Client executes SQL via TDengine REST API.
type Client struct {
	cfg    Config
	http   *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c.cfg.Enabled && strings.TrimSpace(c.cfg.URL) != ""
}

func (c *Client) ExecSQL(sql string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(c.cfg.URL), "/")
	req, err := http.NewRequest(http.MethodPost, base+"/rest/sql", strings.NewReader(sql))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")
	if u := strings.TrimSpace(c.cfg.User); u != "" {
		auth := u + ":" + c.cfg.Password
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(body), fmt.Errorf("tdengine http %d: %s", resp.StatusCode, truncate(string(body), 500))
	}
	return string(body), nil
}

func (c *Client) TestConnection() error {
	if !c.Enabled() {
		return fmt.Errorf("tdengine is disabled or url is empty")
	}
	_, err := c.ExecSQL("show databases")
	return err
}

// PublishStatusSnapshot inserts current rows (full snapshot as new points).
func (c *Client) PublishStatusSnapshot(table string, rows []StatusRow) error {
	if !c.Enabled() {
		return nil
	}
	if len(rows) == 0 {
		return nil
	}
	dbName := strings.TrimSpace(c.cfg.Database)
	if dbName == "" {
		dbName = "semaphore_devices"
	}
	table = strings.TrimSpace(table)
	if table == "" {
		table = "status"
	}
	fqTable := fmt.Sprintf("%s.%s", escapeIdent(dbName), escapeIdent(table))

	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(fqTable)
	b.WriteString(" (ts, project_id, device_id, hostname, ip, status, device_status_raw, winrm_status, api_status) VALUES ")
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000")
	for i, r := range rows {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			"('%s',%d,%d,'%s','%s','%s','%s','%s','%s')",
			now, r.ProjectID, r.DeviceID,
			escapeSQL(r.Hostname), escapeSQL(r.IP),
			escapeSQL(r.Status), escapeSQL(r.DeviceStatusRaw),
			escapeSQL(r.WinRMStatus), escapeSQL(r.APIStatus),
		))
	}
	_, err := c.ExecSQL(b.String())
	return err
}

func escapeIdent(s string) string {
	s = strings.ReplaceAll(s, "`", "")
	return "`" + s + "`"
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ParseConfigJSON loads config from option storage JSON.
func ParseConfigJSON(raw string) (Config, error) {
	var cfg Config
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(raw), &cfg)
	return cfg, err
}

func (cfg Config) RedactedJSON() ([]byte, error) {
	out := cfg
	if out.Password != "" {
		out.Password = "********"
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
