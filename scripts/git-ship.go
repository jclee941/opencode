//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var blockedSecrets = []*regexp.Regexp{
	regexp.MustCompile(`(^|/)\.env(\..+)?$`),
	regexp.MustCompile(`(^|/)credentials\.json$`),
	regexp.MustCompile(`(^|/)id_(rsa|ed25519)(\.pub)?$`),
	regexp.MustCompile(`\.(pem|p12|key)$`),
}

func main() {
	fs := flag.NewFlagSet("git-ship", flag.ExitOnError)
	msg := fs.String("message", "", "Commit message to use")
	push := fs.Bool("push", false, "Push after creating the commit")
	dryRun := fs.Bool("dry-run", false, "Print actions without running git mutations")
	pathsFlag := fs.String("paths", "", "Comma-separated paths to stage instead of all changes")
	fs.Parse(os.Args[1:])

	root, err := gitOutput("rev-parse", "--show-toplevel")
	if err != nil {
		fail(fmt.Errorf("resolve repo root: %w", err))
	}
	if err := os.Chdir(root); err != nil {
		fail(fmt.Errorf("enter repo root: %w", err))
	}

	paths, err := statusPaths()
	if err != nil {
		fail(err)
	}
	if len(paths) == 0 {
		fail(errors.New("no working tree changes to ship"))
	}
	if err := checkBlockedPaths(paths); err != nil {
		fail(err)
	}

	stagePaths := splitPaths(*pathsFlag)
	if *dryRun {
		printPlan(*msg, *push, stagePaths, paths)
		return
	}
	if strings.TrimSpace(*msg) == "" {
		fail(errors.New("--message is required unless --dry-run is used"))
	}

	run("npm", "run", "prepush:check")
	if len(stagePaths) == 0 {
		run("git", "add", "-A")
	} else {
		run(append([]string{"git", "add", "--"}, stagePaths...)...)
	}

	if err := ensureStagedChanges(); err != nil {
		fail(err)
	}

	run("git", "commit", "-m", *msg)
	if *push {
		branch, err := gitOutput("branch", "--show-current")
		if err != nil {
			fail(fmt.Errorf("read current branch: %w", err))
		}
		if branch == "" {
			fail(errors.New("cannot push from detached HEAD"))
		}
		if _, err := gitOutput("rev-parse", "--abbrev-ref", "@{upstream}"); err == nil {
			run("git", "push")
		} else {
			run("git", "push", "-u", "origin", branch)
		}
	}
}

func printPlan(msg string, push bool, stagePaths []string, changed []string) {
	fmt.Printf("repo: %s\n", mustGitOutput("rev-parse", "--show-toplevel"))
	fmt.Printf("message: %s\n", msg)
	fmt.Printf("push: %t\n", push)
	if len(stagePaths) == 0 {
		fmt.Println("stage: git add -A")
	} else {
		fmt.Printf("stage: git add -- %s\n", strings.Join(stagePaths, " "))
	}
	fmt.Println("checks: npm run prepush:check")
	fmt.Println("changed paths:")
	for _, path := range changed {
		fmt.Printf("- %s\n", path)
	}
	if push {
		fmt.Println("push mode: enabled")
	}
}

func ensureStagedChanges() error {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()
	if err == nil {
		return errors.New("no staged changes after git add")
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return nil
	}
	return err
}

func checkBlockedPaths(paths []string) error {
	for _, path := range paths {
		for _, pattern := range blockedSecrets {
			if pattern.MatchString(path) {
				return fmt.Errorf("refusing to automate commit with sensitive-looking path: %s", path)
			}
		}
	}
	return nil
}

func statusPaths() ([]string, error) {
	out, err := gitOutput("status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return nil, fmt.Errorf("read git status: %w", err)
	}
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	paths := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if idx := strings.LastIndex(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths, nil
}

func splitPaths(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		paths = append(paths, trimmed)
	}
	return paths
}

func run(args ...string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fail(fmt.Errorf("run %s: %w", strings.Join(args, " "), err))
	}
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func mustGitOutput(args ...string) string {
	out, err := gitOutput(args...)
	if err != nil {
		return ""
	}
	return out
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
