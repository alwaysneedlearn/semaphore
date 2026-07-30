// Package main is the Semaphore Native RDP Helper (Windows).
// See docs/plan-rdp-helper.md.
package main

import (
	"bufio"
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
	"sort"
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
}

// EnvSession is one project's connection (direct marker and/or SSH SOCKS tunnel).
type EnvSession struct {
	SSHPid     int    `json:"ssh_pid,omitempty"`
	SocksPort  int    `json:"socks_port,omitempty"`
	LandTarget string `json:"land_target,omitempty"`
	Direct     bool   `json:"direct,omitempty"` // connected without SSH
}

type State struct {
	// Sessions maps environment id → independent connection (projects do not share one tunnel).
	Sessions map[string]EnvSession `json:"sessions,omitempty"`

	// Legacy single-session fields (migrated on load).
	ConnectedEnv string `json:"connected_env,omitempty"`
	SSHPid       int    `json:"ssh_pid,omitempty"`
	SocksPort    int    `json:"socks_port,omitempty"`
	LandTarget   string `json:"land_target,omitempty"`
	ControlPath  string `json:"control_path,omitempty"`
}

type launchParams struct {
	ProjectID           int     `json:"project_id"`
	DeviceID            int     `json:"device_id"`
	LogID               int     `json:"log_id"`
	Host                string  `json:"host"`
	RDPPort             int     `json:"rdp_port"`
	RDPUser             string  `json:"rdp_user"`
	RDPPassword         *string `json:"rdp_password"`
	PasswordProvided    bool    `json:"password_provided"`
	LifecycleToken      string  `json:"lifecycle_token"`
	LifecycleExpiresIn  int     `json:"lifecycle_expires_in"`
}

func main() {
	if len(os.Args) < 2 {
		// Double-click / desktop shortcut: keep console open with a prompt.
		if err := cmdShell(); err != nil {
			logf("shell error: %v", err)
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	arg := os.Args[1]
	if strings.HasPrefix(strings.ToLower(arg), protocolScheme+":") {
		// Protocol launches must not leave a visible console for the whole mstsc session
		// (helper waits so SOCKS local-forward / cmdkey cleanup stay alive).
		hideConsoleWindow()
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
	case "shell":
		err = cmdShell()
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
		envID := ""
		if len(os.Args) > 2 {
			envID = os.Args[2]
		}
		err = cmdDisconnect(envID)
	case "open":
		envID := ""
		if len(os.Args) > 2 {
			envID = os.Args[2]
		}
		err = cmdOpen(envID)
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

func exeName() string {
	return filepath.Base(os.Args[0])
}

func printHelp() {
	n := exeName()
	fmt.Printf(`Semaphore RDP Helper

Usage:
  %s                       Interactive shell (for desktop shortcut / double-click)
  %s shell                 Same as no-args
  %s install               Register %s:// (HKCU) and create config in exe folder
  %s envs                  List environments from config
  %s connect <env-id>      SSH tunnel for one project (-N + SOCKS; optional UI -L)
  %s disconnect <env-id>   Tear down that project's tunnel
  %s open [env-id]         Open Semaphore URL in browser
  %s status                Show connection state
  %s help                  Show this help

Protocol (OS): %s://connect?token=...&base=...
               %s://stop?device_id=...&base=...

Config / state / log: same folder as this exe (%s, %s, %s)
`, n, n, n, protocolScheme, n, n, n, n, n, n, protocolScheme, protocolScheme, configFileName, stateFileName, logFileName)
}

func printShellHelp() {
	fmt.Print(`Commands:
  install
  envs
  connect <env-id>
  disconnect <env-id>
  open [env-id]
  status
  help
  exit
`)
}

// cmdShell keeps the console open (desktop shortcut / double-click friendly).
func cmdShell() error {
	if _, err := ensureConfig(); err != nil {
		return err
	}
	if exe, err := os.Executable(); err == nil {
		_ = registerProtocol(exe)
	}
	dir, _ := appDir()
	fmt.Println("Semaphore RDP Helper — interactive shell")
	if dir != "" {
		fmt.Println("Dir:", dir)
	}
	fmt.Println("Type help for commands, exit to quit.")
	fmt.Println()

	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("rdp> ")
		if !sc.Scan() {
			fmt.Println()
			return sc.Err()
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])
		arg1 := ""
		if len(parts) > 1 {
			arg1 = parts[1]
		}
		var err error
		switch cmd {
		case "exit", "quit", "q":
			return nil
		case "help", "?", "h":
			printShellHelp()
		case "install":
			err = cmdInstall()
		case "envs":
			err = cmdEnvs()
		case "status":
			err = cmdStatus()
		case "connect":
			err = cmdConnect(arg1)
		case "disconnect":
			err = cmdDisconnect(arg1)
		case "open":
			err = cmdOpen(arg1)
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s (type help)\n", cmd)
		}
		if err != nil {
			logf("shell cmd %s: %v", cmd, err)
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}
}

// appDir is the directory containing the helper executable (portable layout).
// Uses os.Executable so desktop shortcuts / protocol launches still work even when
// the process working directory is not the exe folder.
func appDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe), nil
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
			ForwardUI:    false,
			Hops:         nil,
			LandUser:     "",
		}},
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
			return State{Sessions: map[string]EnvSession{}}, nil
		}
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return State{}, err
	}
	migrateLegacyState(&s)
	return s, nil
}

func migrateLegacyState(s *State) {
	if s.Sessions == nil {
		s.Sessions = map[string]EnvSession{}
	}
	if len(s.Sessions) == 0 && strings.TrimSpace(s.ConnectedEnv) != "" {
		s.Sessions[s.ConnectedEnv] = EnvSession{
			SSHPid:     s.SSHPid,
			SocksPort:  s.SocksPort,
			LandTarget: s.LandTarget,
			Direct:     s.SSHPid == 0 && s.SocksPort == 0,
		}
	}
	// Drop legacy top-level fields from future writes.
	s.ConnectedEnv = ""
	s.SSHPid = 0
	s.SocksPort = 0
	s.LandTarget = ""
	s.ControlPath = ""
}

func saveState(s State) error {
	migrateLegacyState(&s)
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

func getSession(s State, envID string) (EnvSession, bool) {
	if s.Sessions == nil || envID == "" {
		return EnvSession{}, false
	}
	sess, ok := s.Sessions[envID]
	return sess, ok
}

func putSession(envID string, sess EnvSession) error {
	s, _ := loadState()
	if s.Sessions == nil {
		s.Sessions = map[string]EnvSession{}
	}
	s.Sessions[envID] = sess
	return saveState(s)
}

func clearSession(envID string) error {
	s, _ := loadState()
	if s.Sessions == nil {
		return saveState(State{Sessions: map[string]EnvSession{}})
	}
	if sess, ok := s.Sessions[envID]; ok {
		if sess.SSHPid > 0 {
			killProcess(sess.SSHPid)
		}
		delete(s.Sessions, envID)
	}
	return saveState(s)
}

func connectedEnvIDs(s State) []string {
	var ids []string
	for id, sess := range s.Sessions {
		if sessionAlive(sess) || sess.Direct {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func findEnv(c Config, id string) (Environment, bool) {
	id = strings.TrimSpace(id)
	if id != "" {
		for _, e := range c.Environments {
			if e.ID == id {
				return e, true
			}
		}
		return Environment{}, false
	}
	// No id: only OK when exactly one project is configured.
	if len(c.Environments) == 1 {
		return c.Environments[0], true
	}
	return Environment{}, false
}

func resolveEnv(c Config, id, cmd string) (Environment, error) {
	env, ok := findEnv(c, id)
	if ok {
		return env, nil
	}
	if strings.TrimSpace(id) == "" && len(c.Environments) > 1 {
		return Environment{}, fmt.Errorf("%s: pass <env-id> (multiple projects in config.json)", cmd)
	}
	if strings.TrimSpace(id) != "" {
		return Environment{}, fmt.Errorf("environment %q not found (edit config.json)", id)
	}
	return Environment{}, fmt.Errorf("no environments in config.json")
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
	fmt.Printf("Registered %s://\nConfig: %s\n(Edit config.json next to the exe, then: connect / open)\n", protocolScheme, p)
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
		conn := ""
		if sess, ok := getSession(s, e.ID); ok && (sessionAlive(sess) || sess.Direct) {
			conn = " [connected]"
		}
		fmt.Printf("  %s (%s) hops=%d%s\n", e.ID, e.Name, len(e.Hops), conn)
	}
	return nil
}

func cmdStatus() error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}
	s, _ := loadState()
	ids := connectedEnvIDs(s)
	if len(ids) == 0 {
		fmt.Println("connected: (none)")
		_ = c
		return nil
	}
	for _, id := range ids {
		sess, _ := getSession(s, id)
		if sess.Direct {
			fmt.Printf("connected %s: direct\n", id)
			continue
		}
		alive := sessionAlive(sess)
		fmt.Printf("connected %s: ssh_pid=%d socks_port=%d alive=%v land=%s\n", id, sess.SSHPid, sess.SocksPort, alive, sess.LandTarget)
	}
	return nil
}

func envNeedsSSH(env Environment) bool {
	return len(env.Hops) > 0 || strings.TrimSpace(env.LandHost) != ""
}

func sessionAlive(sess EnvSession) bool {
	if sess.Direct {
		return true
	}
	if sess.SocksPort <= 0 {
		return false
	}
	if sess.SSHPid > 0 && !processExists(sess.SSHPid) {
		return false
	}
	return tcpReachable("127.0.0.1", sess.SocksPort, 500*time.Millisecond)
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
// Other projects' sessions are left untouched.
func ensureConnected(env Environment) (State, EnvSession, error) {
	s, _ := loadState()
	if !envNeedsSSH(env) {
		sess := EnvSession{Direct: true}
		if err := putSession(env.ID, sess); err != nil {
			return s, sess, err
		}
		s, _ = loadState()
		return s, sess, nil
	}
	if sess, ok := getSession(s, env.ID); ok && sessionAlive(sess) {
		return s, sess, nil
	}
	logf("auto-connect env=%s (use this console for SSH password if prompted)", env.ID)
	if err := cmdConnect(env.ID); err != nil {
		return s, EnvSession{}, fmt.Errorf("%w — or open the helper panel and click 连接 first", err)
	}
	s, _ = loadState()
	sess, _ := getSession(s, env.ID)
	return s, sess, nil
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
	env, err := resolveEnv(c, envID, "connect")
	if err != nil {
		return err
	}

	user, host, port, err := landSpec(env)
	if err != nil {
		return err
	}

	// Only replace THIS project's session — leave other projects alone.
	_ = clearSession(env.ID)

	// No SSH needed
	if host == "" {
		if err := putSession(env.ID, EnvSession{Direct: true}); err != nil {
			return err
		}
		fmt.Printf("connected env=%s (direct, no SSH)\n", env.ID)
		logf("connect direct env=%s", env.ID)
		return nil
	}

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

	sess := EnvSession{
		SSHPid:     cmd.Process.Pid,
		SocksPort:  socksPort,
		LandTarget: target,
	}
	if err := putSession(env.ID, sess); err != nil {
		killProcess(cmd.Process.Pid)
		return err
	}
	fmt.Printf("connected env=%s land=%s socks=127.0.0.1:%d pid=%d\n", env.ID, target, socksPort, cmd.Process.Pid)
	logf("connect ok env=%s land=%s socks=%d pid=%d", env.ID, target, socksPort, cmd.Process.Pid)
	return nil
}

func cmdDisconnect(envID string) error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}
	env, err := resolveEnv(c, envID, "disconnect")
	if err != nil {
		return err
	}
	envID = env.ID
	s, _ := loadState()
	if _, ok := getSession(s, envID); !ok {
		fmt.Printf("not connected: %s\n", envID)
		return nil
	}
	if err := clearSession(envID); err != nil {
		return err
	}
	fmt.Printf("disconnected %s\n", envID)
	logf("disconnect ok env=%s", envID)
	return nil
}

func cmdOpen(envID string) error {
	c, err := ensureConfig()
	if err != nil {
		return err
	}
	env, err := resolveEnv(c, envID, "open")
	if err != nil {
		return err
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
	action := strings.ToLower(strings.TrimSpace(u.Host))
	if action == "" {
		action = strings.ToLower(strings.Trim(strings.TrimSpace(u.Path), "/"))
	}
	switch action {
	case "stop":
		return handleProtocolStop(u)
	case "connect", "":
		return handleProtocolConnect(u)
	default:
		return fmt.Errorf("unknown protocol action %q (use connect or stop)", action)
	}
}

func handleProtocolStop(u *url.URL) error {
	q := u.Query()
	deviceID, _ := strconv.Atoi(strings.TrimSpace(q.Get("device_id")))
	if deviceID <= 0 {
		return fmt.Errorf("missing or invalid device_id")
	}
	sess, ok := getActiveRDPSession(deviceID)
	if !ok {
		logf("stop: no active session device=%d", deviceID)
		return fmt.Errorf("no active remote desktop session for device %d on this PC", deviceID)
	}
	logf("stop: killing mstsc device=%d pid=%d log_id=%d", deviceID, sess.PID, sess.LogID)
	killProcess(sess.PID)
	postRDPLaunchEvent(sess.APIBase, sess.LifecycleToken, "mstsc_exited")
	clearActiveRDPSession(deviceID, sess.PID)
	return nil
}

func handleProtocolConnect(u *url.URL) error {
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
	env, ok := pickEnvForLaunch(c, s, base)
	if !ok {
		return fmt.Errorf("no environment configured; open the helper panel, save a project, then retry")
	}

	s, sess, err := ensureConnected(env)
	if err != nil {
		return fmt.Errorf("auto-connect %s failed: %w", env.ID, err)
	}
	_ = s

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
	logf("launch-params env=%s host=%s port=%d user=%s password_provided=%v", env.ID, params.Host, params.RDPPort, params.RDPUser, params.PasswordProvided)

	targetHost := params.Host
	targetPort := params.RDPPort
	if targetPort == 0 {
		targetPort = 3389
	}

	useTunnel := false
	localPort := 0
	sshOK := !sess.Direct && sessionAlive(sess)
	if sshOK {
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
		stop, err := startLocalForwardViaSOCKS("127.0.0.1", sess.SocksPort, localPort, targetHost, targetPort)
		if err != nil {
			return fmt.Errorf("socks forward failed: %w", err)
		}
		defer stop()
		mstscHost = "127.0.0.1"
		mstscPort = localPort
		logf("socks forward env=%s 127.0.0.1:%d -> %s:%d via socks :%d", env.ID, localPort, targetHost, targetPort, sess.SocksPort)
	}

	pass := ""
	if params.PasswordProvided && params.RDPPassword != nil {
		pass = *params.RDPPassword
	}
	lifecycleAPI := apiBase
	lifecycleTok := strings.TrimSpace(params.LifecycleToken)
	deviceID := params.DeviceID
	logID := params.LogID
	onStarted := func(pid int) {
		putActiveRDPSession(ActiveRDPSession{
			DeviceID:       deviceID,
			LogID:          logID,
			PID:            pid,
			APIBase:        lifecycleAPI,
			LifecycleToken: lifecycleTok,
			Host:           targetHost,
			StartedAt:      time.Now(),
		})
		postRDPLaunchEvent(lifecycleAPI, lifecycleTok, "mstsc_started")
	}
	onExited := func(pid int) {
		clearActiveRDPSession(deviceID, pid)
		postRDPLaunchEvent(lifecycleAPI, lifecycleTok, "mstsc_exited")
	}
	if err := launchMSTSC(mstscHost, mstscPort, params.RDPUser, pass, onStarted, onExited); err != nil {
		return err
	}
	return nil
}

// pickEnvForLaunch chooses which project handles an RDP protocol launch.
// Prefer URL match (protocol base=); else the sole configured project.
func pickEnvForLaunch(c Config, s State, base string) (Environment, bool) {
	_ = s
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	norm := func(u string) string {
		return strings.TrimRight(strings.TrimSpace(u), "/")
	}
	if base != "" {
		for _, e := range c.Environments {
			if norm(e.APIBaseURL) == base || norm(e.SemaphoreURL) == base {
				return e, true
			}
		}
	}
	if len(c.Environments) == 1 {
		return c.Environments[0], true
	}
	if len(c.Environments) > 1 && base != "" {
		// Multiple projects but base did not match any semaphore_url.
		return Environment{}, false
	}
	if len(c.Environments) > 0 {
		return c.Environments[0], true
	}
	return Environment{}, false
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

func postRDPLaunchEvent(apiBase, lifecycleToken, event string) {
	lifecycleToken = strings.TrimSpace(lifecycleToken)
	if apiBase == "" || lifecycleToken == "" {
		return
	}
	u := strings.TrimRight(apiBase, "/") + "/api/rdp/launch-events"
	payload, err := json.Marshal(map[string]string{
		"token": lifecycleToken,
		"event": event,
	})
	if err != nil {
		logf("launch-event marshal failed event=%s err=%v", event, err)
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(string(payload)))
	if err != nil {
		logf("launch-event request failed event=%s err=%v", event, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logf("launch-event post failed event=%s err=%v", event, err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		logf("launch-event HTTP %d event=%s body=%s", resp.StatusCode, event, truncate(string(body), 200))
		return
	}
	logf("launch-event ok event=%s", event)
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

func launchMSTSC(host string, port int, user, password string, onStarted, onExited func(pid int)) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("mstsc only supported on Windows (got %s)", runtime.GOOS)
	}
	host = strings.TrimSpace(host)
	user = strings.TrimSpace(user)
	if host == "" {
		return fmt.Errorf("empty RDP host")
	}
	if port <= 0 {
		port = 3389
	}
	fullAddress := fmt.Sprintf("%s:%d", host, port)
	termsrv := "TERMSRV/" + fullAddress

	if password != "" && user != "" {
		_ = runHidden("cmdkey", "/delete:"+termsrv)
		if out, err := combinedOutputHidden("cmdkey", "/generic:"+termsrv, "/user:"+user, "/pass:"+password); err != nil {
			logf("cmdkey failed: %v %s", err, string(out))
			// continue; mstsc will prompt
		} else {
			defer func() {
				_ = runHidden("cmdkey", "/delete:"+termsrv)
			}()
		}
	}

	// Never pass /u: — mstsc treats unknown switches as fatal and shows the usage dialog.
	// Prefer a short-lived .rdp so username can be prefilled without invalid CLI flags.
	rdpBody := buildRDPFile(fullAddress, port, user)
	dir, err := appDir()
	if err != nil {
		return err
	}
	rdpPath := filepath.Join(dir, fmt.Sprintf("launch-%d.rdp", time.Now().UnixNano()))
	if err := os.WriteFile(rdpPath, []byte(rdpBody), 0o600); err != nil {
		return fmt.Errorf("write rdp file: %w", err)
	}
	defer removeFileWithRetry(rdpPath, 8, 400*time.Millisecond)

	cmd := exec.Command("mstsc", rdpPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mstsc: %w", err)
	}
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	logf("mstsc started target=%s user=%s rdp=%s pid=%d", fullAddress, user, rdpPath, pid)
	if onStarted != nil {
		onStarted(pid)
	}

	// mstsc keeps the .rdp open briefly while parsing; delete early once it has loaded
	// so leftover launch-*.rdp files do not accumulate when Remove-at-exit fails.
	go func() {
		time.Sleep(2 * time.Second)
		removeFileWithRetry(rdpPath, 10, 500*time.Millisecond)
	}()

	_ = cmd.Wait()
	if onExited != nil {
		onExited(pid)
	}
	return nil
}

// removeFileWithRetry deletes path, retrying while mstsc (or AV) still holds a lock.
func removeFileWithRetry(path string, attempts int, delay time.Duration) {
	if path == "" || attempts <= 0 {
		return
	}
	for i := 0; i < attempts; i++ {
		err := os.Remove(path)
		if err == nil || os.IsNotExist(err) {
			if i > 0 {
				logf("rdp file removed after retry path=%s attempt=%d", path, i+1)
			}
			return
		}
		if i+1 < attempts {
			time.Sleep(delay)
		} else {
			logf("rdp file remove failed path=%s err=%v", path, err)
		}
	}
}

func buildRDPFile(fullAddress string, port int, user string) string {
	var b strings.Builder
	b.WriteString("full address:s:")
	b.WriteString(fullAddress)
	b.WriteString("\r\n")
	b.WriteString("server port:i:")
	b.WriteString(strconv.Itoa(port))
	b.WriteString("\r\n")
	if user != "" {
		b.WriteString("username:s:")
		b.WriteString(user)
		b.WriteString("\r\n")
	}
	b.WriteString("prompt for credentials:i:1\r\n")
	b.WriteString("authentication level:i:2\r\n")
	// Local Resources → Clipboard: checked by default (mstsc .rdp).
	b.WriteString("redirectclipboard:i:1\r\n")
	return b.String()
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
