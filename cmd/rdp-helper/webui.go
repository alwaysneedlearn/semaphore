package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	_ "embed"
)

const (
	defaultUIAddr  = "127.0.0.1:17300"
	uiAddrFileName = "ui.addr"
)

//go:embed static/index.html
var uiIndexHTML []byte

func uiAddrPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, uiAddrFileName), nil
}

func listenUIAddr() string {
	if v := strings.TrimSpace(os.Getenv("SEMAPHORE_RDP_HELPER_UI")); v != "" {
		return v
	}
	return defaultUIAddr
}

func cmdUI() error {
	_, err := ensureConfig()
	if err != nil {
		return err
	}
	// Best-effort protocol registration on first UI open.
	if exe, e := os.Executable(); e == nil {
		_ = registerProtocol(exe)
	}

	addr := listenUIAddr()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// Another instance likely running — open existing UI.
		if openExistingUI() {
			fmt.Printf("UI already running, opened browser\n")
			return nil
		}
		return fmt.Errorf("listen %s: %w (is another helper UI running?)", addr, err)
	}
	actual := ln.Addr().String()
	if p, e := uiAddrPath(); e == nil {
		_ = os.WriteFile(p, []byte("http://"+actual+"\n"), 0o644)
	}
	fmt.Printf("RDP Helper UI: http://%s\n", actual)
	logf("ui listen %s", actual)
	_ = openBrowser("http://" + actual)

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/api/status", handleAPIStatus)
	mux.HandleFunc("/api/config", handleAPIConfig)
	mux.HandleFunc("/api/connect", handleAPIConnect)
	mux.HandleFunc("/api/disconnect", handleAPIDisconnect)
	mux.HandleFunc("/api/open", handleAPIOpen)
	mux.HandleFunc("/api/install", handleAPIInstall)
	mux.HandleFunc("/api/logs", handleAPILogs)

	srv := &http.Server{
		Handler:           withLocalOnly(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return srv.Serve(ln)
}

func openExistingUI() bool {
	p, err := uiAddrPath()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	u := strings.TrimSpace(string(b))
	if u == "" {
		return false
	}
	if err := openBrowser(u); err != nil {
		return false
	}
	return true
}

func withLocalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(uiIndexHTML)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

var uiMu sync.Mutex

func handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	c, err := ensureConfig()
	if err != nil {
		writeAPIError(w, 500, err)
		return
	}
	s, _ := loadState()
	connected := map[string]bool{}
	for _, id := range connectedEnvIDs(s) {
		connected[id] = true
	}
	raw, _ := json.MarshalIndent(c, "", "  ")
	writeJSON(w, 200, map[string]any{
		"config":           c,
		"config_json":      string(raw),
		"state":            s,
		"connected_env_ids": connectedEnvIDs(s),
		"connected":        connected,
	})
}

func handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c, err := ensureConfig()
		if err != nil {
			writeAPIError(w, 500, err)
			return
		}
		writeJSON(w, 200, c)
	case http.MethodPut:
		var c Config
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeAPIError(w, 400, err)
			return
		}
		if len(c.Environments) == 0 {
			writeAPIError(w, 400, fmt.Errorf("environments required"))
			return
		}
		if c.ActiveEnv == "" {
			c.ActiveEnv = c.Environments[0].ID
		}
		uiMu.Lock()
		err := saveConfig(c)
		uiMu.Unlock()
		if err != nil {
			writeAPIError(w, 500, err)
			return
		}
		logf("ui saved config envs=%d", len(c.Environments))
		writeJSON(w, 200, map[string]string{"ok": "true"})
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func handleAPIConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		EnvID string `json:"env_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	uiMu.Lock()
	err := cmdConnect(body.EnvID)
	uiMu.Unlock()
	if err != nil {
		writeAPIError(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "true"})
}

func handleAPIDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		EnvID string `json:"env_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	uiMu.Lock()
	err := cmdDisconnect(body.EnvID)
	uiMu.Unlock()
	if err != nil {
		writeAPIError(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "true"})
}

func handleAPIOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		EnvID string `json:"env_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.EnvID != "" {
		c, err := ensureConfig()
		if err == nil {
			c.ActiveEnv = body.EnvID
			_ = saveConfig(c)
		}
	}
	if err := cmdOpen(); err != nil {
		writeAPIError(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "true"})
}

func handleAPIInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if err := cmdInstall(); err != nil {
		writeAPIError(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "true"})
}

func handleAPILogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	tail := 200
	p, err := logPath()
	if err != nil {
		writeAPIError(w, 500, err)
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, 200, map[string]string{"text": ""})
			return
		}
		writeAPIError(w, 500, err)
		return
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}
	writeJSON(w, 200, map[string]string{"text": strings.Join(lines, "\n")})
}
