//go:build ignore

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultServerPassword = "opencode-tmux-local"
	serverHost            = "127.0.0.1"
	windowDataRootDir     = ".local/share/opencode-windows"
	windowStateRootDir    = ".local/state/opencode-windows"
	stateTrackingDirName  = "state"
)

type serverState struct {
	PID       int    `json:"pid"`
	Port      int    `json:"port"`
	Hostname  string `json:"hostname"`
	WindowID  string `json:"window_id"`
	DataDir   string `json:"data_dir"`
	StateDir  string `json:"state_dir"`
	LogFile   string `json:"log_file"`
	StartedAt string `json:"started_at"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	subcommand := os.Args[1]
	args := os.Args[2:]

	switch subcommand {
	case "start":
		err = runStart(args)
	case "stop":
		err = runStop(args)
	case "attach":
		err = runAttach(args)
	case "status":
		err = runStatus(args)
	case "cleanup":
		err = runCleanup(args)
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "error: unknown subcommand %q\n", subcommand)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/opencode-tmux-server.go start [--window <id>] [--cwd <dir>]\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/opencode-tmux-server.go stop [--window <id>] [--all]\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/opencode-tmux-server.go attach [--window <id>]\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/opencode-tmux-server.go status\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/opencode-tmux-server.go cleanup [--purge]\n")
}

func runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	windowFlag := fs.String("window", "", "target window id")
	cwdFlag := fs.String("cwd", "", "working directory for opencode server")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("start: unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	windowID, windowFSName, err := resolveWindow(*windowFlag)
	if err != nil {
		return err
	}

	if st, err := readState(windowFSName); err == nil {
		if isProcessAlive(st.PID) {
			fmt.Printf("Server already running for window %s on http://%s:%d\n", st.WindowID, st.Hostname, st.Port)
			return nil
		}
		_ = os.Remove(stateFilePath(windowFSName))
	}

	wd := *cwdFlag
	if wd == "" {
		wd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve current directory: %w", err)
		}
	}

	opencodeBin, err := findOpencodeBinary()
	if err != nil {
		return err
	}

	port, err := findFreePort()
	if err != nil {
		return err
	}

	password := os.Getenv("OPENCODE_SERVER_PASSWORD")
	if password == "" {
		password = defaultServerPassword
	}

	dirData := dataDir(windowFSName)
	dirState := stateDirForWindow(windowFSName)
	if err := os.MkdirAll(dirData, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(dirState, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	logDir := filepath.Join(dirData, "opencode")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	logFilePath := filepath.Join(logDir, "server.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	cmd := exec.Command(opencodeBin, "serve", "--port", strconv.Itoa(port), "--hostname", serverHost)
	cmd.Dir = wd
	cmd.Env = buildServerEnv(dirData, dirState, password)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start opencode server: %w", err)
	}

	pid := cmd.Process.Pid
	time.Sleep(2 * time.Second)
	if !isProcessAlive(pid) {
		return fmt.Errorf("opencode server exited early (pid %d), check log: %s", pid, logFilePath)
	}

	st := &serverState{
		PID:       pid,
		Port:      port,
		Hostname:  serverHost,
		WindowID:  windowID,
		DataDir:   dirData,
		StateDir:  dirState,
		LogFile:   logFilePath,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeState(st); err != nil {
		return err
	}

	fmt.Printf("Server started for window %s on http://%s:%d\n", windowID, serverHost, port)
	return nil
}

func runStop(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	windowFlag := fs.String("window", "", "target window id")
	allFlag := fs.Bool("all", false, "stop all window servers")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("stop: unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	if *allFlag {
		files, err := listStateFiles()
		if err != nil {
			return err
		}
		if len(files) == 0 {
			fmt.Println("No servers found.")
			return nil
		}
		for _, file := range files {
			windowFSName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
			stopped, windowID, err := stopWindow(windowFSName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: stop %s: %v\n", windowFSName, err)
				continue
			}
			if stopped {
				fmt.Printf("Server stopped for window %s\n", windowID)
			}
		}
		return nil
	}

	windowID, windowFSName, err := resolveWindow(*windowFlag)
	if err != nil {
		return err
	}

	stopped, stoppedWindowID, err := stopWindow(windowFSName)
	if err != nil {
		return err
	}
	if !stopped {
		return fmt.Errorf("no server state found for window %s", windowID)
	}
	fmt.Printf("Server stopped for window %s\n", stoppedWindowID)
	return nil
}

func runAttach(args []string) error {
	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	windowFlag := fs.String("window", "", "target window id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("attach: unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	windowID, windowFSName, err := resolveWindow(*windowFlag)
	if err != nil {
		return err
	}

	st, err := readState(windowFSName)
	if err != nil {
		return fmt.Errorf("no server for window %s; run start first", windowID)
	}
	if !isProcessAlive(st.PID) {
		return fmt.Errorf("server for window %s is not running; run start first", st.WindowID)
	}

	opencodeBin, err := findOpencodeBinary()
	if err != nil {
		return err
	}

	password := os.Getenv("OPENCODE_SERVER_PASSWORD")
	if password == "" {
		password = defaultServerPassword
	}

	url := fmt.Sprintf("http://%s:%d", st.Hostname, st.Port)
	env := withEnvOverride(os.Environ(), "OPENCODE_SERVER_PASSWORD", password)
	argv := []string{opencodeBin, "attach", url}
	return syscall.Exec(opencodeBin, argv, env)
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("status: unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	files, err := listStateFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("No servers found.")
		return nil
	}

	fmt.Printf("%-10s %-7s %-7s %-8s %s\n", "WINDOW", "PORT", "PID", "STATUS", "STARTED")

	runningCount := 0
	deadCount := 0

	for _, file := range files {
		st, err := readStateFile(file)
		if err != nil {
			window := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
			fmt.Printf("%-10s %-7s %-7s %-8s %s\n", window, "-", "-", "dead", "unknown")
			deadCount++
			continue
		}

		status := "dead"
		if isProcessAlive(st.PID) {
			status = "running"
			runningCount++
		} else {
			deadCount++
		}

		window := windowIDToFSNameFromID(st.WindowID)
		if window == "" {
			window = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		}
		fmt.Printf("%-10s %-7d %-7d %-8s %s\n", window, st.Port, st.PID, status, startedAgo(st.StartedAt))
	}

	fmt.Printf("Total: running=%d dead=%d\n", runningCount, deadCount)
	return nil
}

func runCleanup(args []string) error {
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	purgeFlag := fs.Bool("purge", false, "remove per-window data dir for dead servers")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("cleanup: unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	files, err := listStateFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("Nothing to clean.")
		return nil
	}

	cleaned := 0
	for _, file := range files {
		windowFSName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		st, err := readStateFile(file)
		if err != nil {
			if remErr := os.Remove(file); remErr == nil {
				fmt.Printf("Removed invalid state file: %s\n", file)
				cleaned++
			}
			continue
		}

		if isProcessAlive(st.PID) {
			continue
		}

		if err := os.Remove(file); err != nil {
			fmt.Fprintf(os.Stderr, "error: remove state %s: %v\n", file, err)
			continue
		}
		fmt.Printf("Removed dead state for window %s\n", st.WindowID)
		cleaned++

		if *purgeFlag {
			if err := os.RemoveAll(dataDir(windowFSName)); err != nil {
				fmt.Fprintf(os.Stderr, "error: purge data dir %s: %v\n", dataDir(windowFSName), err)
			} else {
				fmt.Printf("Purged data dir: %s\n", dataDir(windowFSName))
			}
		}
	}

	fmt.Printf("Cleanup complete. Removed %d stale state file(s).\n", cleaned)
	return nil
}

func getTmuxWindowID() (sessionName, windowIndex string, err error) {
	if strings.TrimSpace(os.Getenv("TMUX")) == "" {
		return "", "", errors.New("not inside tmux (TMUX environment variable is empty)")
	}

	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}:#{window_index}")
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("tmux display-message failed: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected tmux window format: %q", raw)
	}
	return parts[0], parts[1], nil
}

func windowIDToFSName(session, window string) string {
	return strings.ReplaceAll(session+":"+window, ":", "-")
}

func windowIDToFSNameFromID(windowID string) string {
	if strings.TrimSpace(windowID) == "" {
		return ""
	}
	return strings.ReplaceAll(windowID, ":", "-")
}

func resolveWindow(windowFlag string) (windowID string, windowFSName string, err error) {
	if strings.TrimSpace(windowFlag) != "" {
		windowID = strings.TrimSpace(windowFlag)
		windowFSName = windowIDToFSNameFromID(windowID)
		return windowID, windowFSName, nil
	}

	session, window, err := getTmuxWindowID()
	if err != nil {
		return "", "", err
	}
	windowID = session + ":" + window
	windowFSName = windowIDToFSName(session, window)
	return windowID, windowFSName, nil
}

func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", serverHost+":0")
	if err != nil {
		return 0, fmt.Errorf("find free port: %w", err)
	}
	defer func() {
		_ = ln.Close()
	}()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("failed to resolve free port address")
	}
	return addr.Port, nil
}

func findOpencodeBinary() (string, error) {
	if p, err := exec.LookPath("opencode"); err == nil {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	fallback := filepath.Join(home, ".opencode", "bin", "opencode")
	if st, err := os.Stat(fallback); err == nil && !st.IsDir() {
		return fallback, nil
	}

	return "", errors.New("opencode binary not found in PATH or ~/.opencode/bin/opencode")
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func stopWindow(windowFSName string) (stopped bool, windowID string, err error) {
	st, err := readState(windowFSName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	windowID = st.WindowID
	if strings.TrimSpace(windowID) == "" {
		windowID = windowFSName
	}

	if isProcessAlive(st.PID) {
		if err := syscall.Kill(st.PID, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
			return false, windowID, fmt.Errorf("send SIGTERM to pid %d: %w", st.PID, err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if !isProcessAlive(st.PID) {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if isProcessAlive(st.PID) {
			if err := syscall.Kill(st.PID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				return false, windowID, fmt.Errorf("send SIGKILL to pid %d: %w", st.PID, err)
			}
		}
	}

	if err := os.Remove(stateFilePath(windowFSName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, windowID, fmt.Errorf("remove state file: %w", err)
	}
	return true, windowID, nil
}

func readState(windowFSName string) (*serverState, error) {
	return readStateFile(stateFilePath(windowFSName))
}

func readStateFile(path string) (*serverState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var st serverState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("decode state file %s: %w", path, err)
	}
	return &st, nil
}

func writeState(st *serverState) error {
	windowFSName := windowIDToFSNameFromID(st.WindowID)
	if windowFSName == "" {
		return errors.New("empty window id")
	}

	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		return fmt.Errorf("create state tracking dir: %w", err)
	}

	path := stateFilePath(windowFSName)
	payload, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state file: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write state file %s: %w", path, err)
	}
	return nil
}

func stateDir() string {
	home := userHomeOrFallback()
	return filepath.Join(home, windowStateRootDir, stateTrackingDirName)
}

func dataDir(windowFSName string) string {
	home := userHomeOrFallback()
	return filepath.Join(home, windowDataRootDir, windowFSName)
}

func stateDirForWindow(windowFSName string) string {
	home := userHomeOrFallback()
	return filepath.Join(home, windowStateRootDir, windowFSName)
}

func stateFilePath(windowFSName string) string {
	return filepath.Join(stateDir(), windowFSName+".json")
}

func userHomeOrFallback() string {
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return home
	}
	if envHome := strings.TrimSpace(os.Getenv("HOME")); envHome != "" {
		return envHome
	}
	return "."
}

func buildServerEnv(windowDataDir string, windowStateDir string, password string) []string {
	env := os.Environ()
	env = withEnvOverride(env, "XDG_DATA_HOME", windowDataDir)
	env = withEnvOverride(env, "XDG_STATE_HOME", windowStateDir)
	env = withEnvOverride(env, "OPENCODE_SERVER_PASSWORD", password)
	return env
}

func withEnvOverride(env []string, key string, value string) []string {
	prefix := key + "="
	for i := range env {
		if strings.HasPrefix(env[i], prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func listStateFiles() ([]string, error) {
	dir := stateDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list state dir %s: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func startedAgo(ts string) string {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(ts))
	if err != nil {
		return "unknown"
	}

	d := time.Since(t)
	if d < 0 {
		d = 0
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
