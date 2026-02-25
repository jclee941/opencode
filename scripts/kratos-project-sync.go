//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	reset  = "\033[0m"
)

type project struct {
	name string
	path string
}

type projectEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Root         string `json:"root"`
	CreatedAt    string `json:"createdAt"`
	LastAccessed string `json:"lastAccessed"`
}

type projectsFile struct {
	Projects    []projectEntry `json:"projects"`
	LastUpdated string         `json:"lastUpdated"`
}

type sshConfig struct {
	host string
	user string
	dir  string
}

type workItem struct {
	repo    project
	entry   projectEntry
	status  string
	message string
}

const (
	pathUnitName    = "kratos-sync.path"
	serviceUnitName = "kratos-sync.service"
)

var useColor bool

func init() {
	fi, err := os.Stdout.Stat()
	if err == nil {
		useColor = fi.Mode()&os.ModeCharDevice != 0
	}
}

var pathUnitTemplate = `[Unit]
Description=Watch ~/dev for new project directories

[Path]
PathModified={{DEV_DIR}}
Unit=kratos-sync.service

[Install]
WantedBy=default.target
`

var serviceUnitTemplate = `[Unit]
Description=Kratos project sync for ~/dev repos

[Service]
Type=oneshot
ExecStartPre=/bin/sleep 2
ExecStart={{BINARY_PATH}} sync
Environment=HOME={{HOME_DIR}}
Environment=PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
WorkingDirectory={{WORK_DIR}}
`

func main() {
	subcommand, subArgs := parseSubcommand(os.Args[1:])
	var err error

	switch subcommand {
	case "sync":
		err = runSync(subArgs)
	case "install":
		err = runInstall(subArgs)
	case "uninstall":
		err = runUninstall(subArgs)
	default:
		err = fmt.Errorf("unknown subcommand: %s", subcommand)
	}

	if err != nil {
		printErr(subcommand, err)
		os.Exit(1)
	}
}

func parseSubcommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "sync", nil
	}
	if strings.HasPrefix(args[0], "-") {
		return "sync", args
	}
	switch args[0] {
	case "sync", "install", "uninstall":
		return args[0], args[1:]
	default:
		return args[0], args[1:]
	}
}

func runSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	devDirFlag := fs.String("dev-dir", "~/dev", "Directory containing immediate project subdirectories")
	dryRun := fs.Bool("dry-run", false, "List projects that would be registered without SSH writes")
	kratosHost := fs.String("kratos-host", "192.168.50.112", "Kratos SSH host")
	kratosUser := fs.String("kratos-user", "root", "Kratos SSH user")
	kratosDir := fs.String("kratos-dir", "/root/.kratos", "Kratos data directory on remote host")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	devDir, err := resolvePath(*devDirFlag)
	if err != nil {
		return fmt.Errorf("resolve --dev-dir: %w", err)
	}

	repos, skippedDirs, err := scanGitRepos(devDir)
	if err != nil {
		return fmt.Errorf("scan dev directory: %w", err)
	}

	fmt.Printf("Scanning: %s\n", devDir)
	fmt.Printf("Found %d git repos, %d non-git directories\n\n", len(repos), len(skippedDirs))

	for _, dirName := range skippedDirs {
		fmt.Printf("%s  %s (%s)\n", colorize(yellow, "SKIP"), dirName, filepath.Join(devDir, dirName))
	}
	if len(skippedDirs) > 0 {
		fmt.Println()
	}

	cfg := sshConfig{host: *kratosHost, user: *kratosUser, dir: *kratosDir}
	defer closeSSHMaster(cfg)

	store, err := readProjectsFile(cfg)
	if err != nil {
		return fmt.Errorf("read remote projects.json: %w", err)
	}

	byID := make(map[string]struct{}, len(store.Projects))
	byRoot := make(map[string]struct{}, len(store.Projects))
	for _, p := range store.Projects {
		byID[p.ID] = struct{}{}
		byRoot[normalizePath(p.Root)] = struct{}{}
	}

	now := timestampNow()
	items := make([]workItem, 0, len(repos))
	newEntries := make([]projectEntry, 0)

	for _, repo := range repos {
		absRoot, absErr := filepath.Abs(repo.path)
		if absErr != nil {
			items = append(items, workItem{repo: repo, status: "error", message: absErr.Error()})
			continue
		}

		entry := projectEntry{
			ID:           projectID(absRoot),
			Name:         repo.name,
			Root:         absRoot,
			CreatedAt:    now,
			LastAccessed: now,
		}

		if _, ok := byID[entry.ID]; ok {
			items = append(items, workItem{repo: repo, entry: entry, status: "already-exists", message: "already registered by id"})
			continue
		}
		if _, ok := byRoot[normalizePath(entry.Root)]; ok {
			items = append(items, workItem{repo: repo, entry: entry, status: "already-exists", message: "already registered by root"})
			continue
		}

		items = append(items, workItem{repo: repo, entry: entry, status: "new", message: "pending registration"})
		newEntries = append(newEntries, entry)
	}

	if *dryRun {
		registered := 0
		skipped := len(skippedDirs)
		errors := 0
		for _, item := range items {
			switch item.status {
			case "already-exists":
				skipped++
				fmt.Printf("%s  %s (%s)\n", colorize(yellow, "SKIP"), item.repo.name, item.message)
			case "new":
				fmt.Printf("%s   %s (%s)\n", colorize(yellow, "DRY"), item.repo.name, item.repo.path)
			default:
				errors++
				fmt.Printf("%s %s (%s)\n", colorize(red, "ERROR"), item.repo.name, item.message)
			}
		}
		fmt.Printf("\nSummary: %d registered, %d skipped, %d errors\n", registered, skipped, errors)
		fmt.Printf("Dry-run candidates: %d\n", len(newEntries))
		return nil
	}

	for i := range items {
		if items[i].status != "new" {
			continue
		}
		if writeErr := writeProjectJSON(cfg, items[i].entry); writeErr != nil {
			items[i].status = "error"
			items[i].message = writeErr.Error()
			continue
		}
		items[i].status = "staged"
		items[i].message = "registered"
	}

	staged := make([]projectEntry, 0)
	for _, item := range items {
		if item.status == "staged" {
			staged = append(staged, item.entry)
		}
	}

	if len(staged) > 0 {
		merged := projectsFile{
			Projects:    append(append([]projectEntry{}, store.Projects...), staged...),
			LastUpdated: timestampNow(),
		}
		if writeErr := writeProjectsFile(cfg, merged); writeErr != nil {
			for i := range items {
				if items[i].status == "staged" {
					items[i].status = "error"
					items[i].message = fmt.Sprintf("remote projects.json update failed: %v", writeErr)
				}
			}
		}
	}

	registered := 0
	skipped := len(skippedDirs)
	errors := 0

	for _, item := range items {
		switch item.status {
		case "already-exists":
			skipped++
			fmt.Printf("%s  %s (%s)\n", colorize(yellow, "SKIP"), item.repo.name, item.message)
		case "staged":
			registered++
			fmt.Printf("%s    %s (%s)\n", colorize(green, "OK"), item.repo.name, item.message)
		case "error":
			errors++
			fmt.Printf("%s %s (%s)\n", colorize(red, "ERROR"), item.repo.name, item.message)
		default:
			errors++
			fmt.Printf("%s %s (unexpected status: %s)\n", colorize(red, "ERROR"), item.repo.name, item.status)
		}
	}

	fmt.Printf("\nSummary: %d registered, %d skipped, %d errors\n", registered, skipped, errors)
	return nil
}

func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	sourcePath, err := currentSourcePath()
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}
	repoRoot := filepath.Dir(filepath.Dir(sourcePath))
	devDir := filepath.Join(homeDir, "dev")
	binaryPath := filepath.Join(homeDir, ".local", "bin", "kratos-project-sync")
	userSystemdDir := filepath.Join(homeDir, ".config", "systemd", "user")

	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return fmt.Errorf("create local bin dir: %w", err)
	}

	if err := buildBinary(binaryPath, sourcePath); err != nil {
		return err
	}

	if err := os.MkdirAll(userSystemdDir, 0o755); err != nil {
		return fmt.Errorf("create user systemd dir: %w", err)
	}

	pathUnitContent := strings.ReplaceAll(pathUnitTemplate, "{{DEV_DIR}}", devDir)
	serviceUnitContent := serviceUnitTemplate
	serviceUnitContent = strings.ReplaceAll(serviceUnitContent, "{{BINARY_PATH}}", binaryPath)
	serviceUnitContent = strings.ReplaceAll(serviceUnitContent, "{{HOME_DIR}}", homeDir)
	serviceUnitContent = strings.ReplaceAll(serviceUnitContent, "{{WORK_DIR}}", repoRoot)

	if err := os.WriteFile(filepath.Join(userSystemdDir, pathUnitName), []byte(pathUnitContent), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", pathUnitName, err)
	}
	if err := os.WriteFile(filepath.Join(userSystemdDir, serviceUnitName), []byte(serviceUnitContent), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", serviceUnitName, err)
	}

	if _, err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	if _, err := runSystemctl("enable", "--now", pathUnitName); err != nil {
		return err
	}

	statusOut, err := runSystemctl("status", "--no-pager", pathUnitName)
	if err != nil {
		return err
	}

	fmt.Printf("%s installed and started\n", colorize(green, pathUnitName))
	fmt.Print(statusOut)
	return nil
}

func runUninstall(args []string) error {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	userSystemdDir := filepath.Join(homeDir, ".config", "systemd", "user")
	binaryPath := filepath.Join(homeDir, ".local", "bin", "kratos-project-sync")

	_, _ = runSystemctl("stop", pathUnitName, serviceUnitName)
	_, _ = runSystemctl("disable", pathUnitName, serviceUnitName)

	if err := removeIfExists(filepath.Join(userSystemdDir, pathUnitName)); err != nil {
		return err
	}
	if err := removeIfExists(filepath.Join(userSystemdDir, serviceUnitName)); err != nil {
		return err
	}
	if err := removeIfExists(binaryPath); err != nil {
		return err
	}

	if _, err := runSystemctl("daemon-reload"); err != nil {
		return err
	}

	fmt.Printf("%s removed units and binary\n", colorize(green, "OK"))
	return nil
}

func currentSourcePath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	return filepath.Abs(file)
}

func buildBinary(binaryPath, sourcePath string) error {
	cmd := exec.Command("go", "build", "-o", binaryPath, sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, string(output))
	}
	return nil
}

func resolvePath(raw string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := strings.TrimSpace(raw)
	if path == "~" {
		path = home
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	return filepath.Abs(path)
}

func scanGitRepos(devDir string) ([]project, []string, error) {
	entries, err := os.ReadDir(devDir)
	if err != nil {
		return nil, nil, err
	}

	repos := make([]project, 0)
	skipped := make([]string, 0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		repoPath := filepath.Join(devDir, entry.Name())
		gitPath := filepath.Join(repoPath, ".git")
		info, err := os.Stat(gitPath)
		if err != nil || !info.IsDir() {
			skipped = append(skipped, entry.Name())
			continue
		}

		repos = append(repos, project{name: entry.Name(), path: repoPath})
	}

	return repos, skipped, nil
}

func projectID(absPath string) string {
	normalized := strings.ToLower(absPath)
	sum := sha256.Sum256([]byte(normalized))
	hexSum := fmt.Sprintf("%x", sum)
	return "proj_" + hexSum[:12]
}

func timestampNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func normalizePath(path string) string {
	return strings.ToLower(filepath.Clean(path))
}

func readProjectsFile(cfg sshConfig) (projectsFile, error) {
	filePath := filepath.Join(cfg.dir, "projects.json")
	stdout, stderr, err := runSSH(cfg, "cat "+shellQuote(filePath), nil)
	if err != nil {
		lower := strings.ToLower(stderr + " " + stdout)
		if strings.Contains(lower, "no such file") {
			return projectsFile{Projects: make([]projectEntry, 0)}, nil
		}
		return projectsFile{}, fmt.Errorf("ssh read failed: %v (%s)", err, strings.TrimSpace(stderr))
	}

	content := strings.TrimSpace(stdout)
	if content == "" {
		return projectsFile{Projects: make([]projectEntry, 0)}, nil
	}

	var parsed projectsFile
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return projectsFile{}, fmt.Errorf("decode remote projects.json: %w", err)
	}
	if parsed.Projects == nil {
		parsed.Projects = make([]projectEntry, 0)
	}
	return parsed, nil
}

func writeProjectsFile(cfg sshConfig, data projectsFile) error {
	if err := ensureRemoteDir(cfg, cfg.dir); err != nil {
		return err
	}

	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	filePath := filepath.Join(cfg.dir, "projects.json")
	_, stderr, err := runSSH(cfg, "cat > "+shellQuote(filePath), body)
	if err != nil {
		return fmt.Errorf("ssh write %s failed: %v (%s)", filePath, err, strings.TrimSpace(stderr))
	}
	return nil
}

func writeProjectJSON(cfg sshConfig, entry projectEntry) error {
	projectDir := filepath.Join(cfg.dir, "projects", entry.ID)
	if err := ensureRemoteDir(cfg, projectDir); err != nil {
		return err
	}

	body, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	filePath := filepath.Join(projectDir, "project.json")
	_, stderr, err := runSSH(cfg, "cat > "+shellQuote(filePath), body)
	if err != nil {
		return fmt.Errorf("ssh write %s failed: %v (%s)", filePath, err, strings.TrimSpace(stderr))
	}
	return nil
}

func ensureRemoteDir(cfg sshConfig, dir string) error {
	_, stderr, err := runSSH(cfg, "mkdir -p "+shellQuote(dir), nil)
	if err != nil {
		return fmt.Errorf("ssh mkdir %s failed: %v (%s)", dir, err, strings.TrimSpace(stderr))
	}
	return nil
}

func runSSH(cfg sshConfig, remoteCmd string, stdin []byte) (string, string, error) {
	target := fmt.Sprintf("%s@%s", cfg.user, cfg.host)
	args := append(sshBaseArgs(cfg), target, remoteCmd)
	cmd := exec.Command("ssh", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	stdoutBytes, errOut := cmd.Output()
	if errOut != nil {
		stderr := ""
		if exitErr, ok := errOut.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		return string(stdoutBytes), stderr, errOut
	}

	return string(stdoutBytes), "", nil
}

func sshBaseArgs(cfg sshConfig) []string {
	controlPath := filepath.Join(os.TempDir(), fmt.Sprintf("kratos-ssh-%s@%s", cfg.user, cfg.host))
	return []string{
		"-o", "ConnectTimeout=5",
		"-o", "BatchMode=yes",
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=" + controlPath,
		"-o", "ControlPersist=30s",
	}
}

func closeSSHMaster(cfg sshConfig) {
	target := fmt.Sprintf("%s@%s", cfg.user, cfg.host)
	args := append(sshBaseArgs(cfg), "-O", "exit", target)
	cmd := exec.Command("ssh", args...)
	_ = cmd.Run()
}

func shellQuote(raw string) string {
	return "'" + strings.ReplaceAll(raw, "'", "'\"'\"'") + "'"
}

func printErr(action string, err error) {
	fmt.Printf("%s %s: %v\n", colorize(red, "ERROR"), action, err)
}

func colorize(color, text string) string {
	if !useColor {
		return text
	}
	return color + text + reset
}

func runSystemctl(args ...string) (string, error) {
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("systemctl %v failed: %w\n%s", args, err, string(out))
	}
	return string(out), nil
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("remove %s: %w", path, err)
}
