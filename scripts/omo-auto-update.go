//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// omo-auto-update.go — Pulls merged Dependabot updates to the local dev machine.
//
// Designed to run as a cron job. Safe to run frequently:
//   - Skips if working tree is dirty (uncommitted changes)
//   - Skips if already up-to-date
//   - Only runs bun install when package.json actually changed
//   - Only regenerates config when config sources changed
//   - Updates opencode CLI via bun
//
// Usage:
//   go run scripts/omo-auto-update.go           # normal mode
//   go run scripts/omo-auto-update.go --dry-run  # show what would happen

var repoDir = filepath.Join(os.Getenv("HOME"), ".config", "opencode")

func main() {
	log.SetFlags(0)
	dryRun := len(os.Args) > 1 && os.Args[1] == "--dry-run"

	ts := time.Now().Format(time.RFC3339)

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		log.Fatalf("%s SKIP: %s is not a git repository", ts, repoDir)
	}

	status, err := gitCmd("status", "--porcelain")
	if err != nil {
		log.Fatalf("%s ERROR: git status failed: %v", ts, err)
	}
	if strings.TrimSpace(status) != "" {
		log.Printf("%s SKIP: working tree has uncommitted changes", ts)
		os.Exit(0)
	}

	if dryRun {
		log.Printf("%s DRY-RUN: would fetch origin master", ts)
	} else {
		if _, err := gitCmd("fetch", "origin", "master"); err != nil {
			log.Fatalf("%s ERROR: git fetch failed: %v", ts, err)
		}
	}

	localHead, err := gitCmd("rev-parse", "HEAD")
	if err != nil {
		log.Fatalf("%s ERROR: git rev-parse HEAD failed: %v", ts, err)
	}
	remoteHead, err := gitCmd("rev-parse", "origin/master")
	if err != nil {
		log.Fatalf("%s ERROR: git rev-parse origin/master failed: %v", ts, err)
	}

	localHead = strings.TrimSpace(localHead)
	remoteHead = strings.TrimSpace(remoteHead)

	if localHead == remoteHead {
		log.Printf("%s OK: already up-to-date (%s)", ts, localHead[:8])
		os.Exit(0)
	}

	diffFiles, err := gitCmd("diff", "--name-only", localHead, remoteHead)
	if err != nil {
		log.Fatalf("%s ERROR: git diff --name-only failed: %v", ts, err)
	}

	changedFiles := strings.Split(strings.TrimSpace(diffFiles), "\n")
	pkgChanged := contains(changedFiles, "package.json")
	configChanged := anyPrefix(changedFiles, "config/")
	bunChanged := contains(changedFiles, "bun.lock")

	log.Printf("%s UPDATE: %s -> %s (%d files changed)", ts, localHead[:8], remoteHead[:8], len(changedFiles))
	for _, f := range changedFiles {
		log.Printf("  %s", f)
	}

	if dryRun {
		log.Printf("%s DRY-RUN: would pull, bun_install=%v, gen_config=%v", ts, pkgChanged || bunChanged, configChanged)
		os.Exit(0)
	}

	pullOut, err := gitCmd("pull", "--ff-only", "origin", "master")
	if err != nil {
		log.Fatalf("%s ERROR: git pull --ff-only failed: %v\n%s", ts, err, pullOut)
	}
	log.Printf("%s PULLED: %s", ts, strings.TrimSpace(pullOut))

	if pkgChanged || bunChanged {
		log.Printf("%s INSTALLING: bun install (package.json changed)", ts)
		out, err := bunCmd("install")
		if err != nil {
			log.Printf("%s ERROR: bun install failed: %v\n%s", ts, err, out)
		} else {
			log.Printf("%s INSTALLED: bun dependencies updated", ts)
		}
	}

	if configChanged {
		log.Printf("%s REGENERATING: config changed, running gen-opencode-config", ts)
		out, err := runCmd("go", "run", "scripts/gen-opencode-config.go")
		if err != nil {
			log.Printf("%s ERROR: config generation failed: %v\n%s", ts, err, out)
		} else {
			log.Printf("%s REGENERATED: opencode.jsonc updated", ts)
		}
	}

	// Update opencode CLI via bun
	log.Printf("%s UPDATING: opencode CLI via bun", ts)
	out, err := bunCmd("install", "-g", "opencode-ai@latest")
	if err != nil {
		log.Printf("%s ERROR: opencode update failed: %v\n%s", ts, err, out)
	} else {
		log.Printf("%s UPDATED: opencode CLI updated", ts)
	}

	log.Printf("%s DONE: update complete", ts)
}

func gitCmd(args ...string) (string, error) {
	return runCmd("git", args...)
}

func bunCmd(args ...string) (string, error) {
	return runCmd("bun", args...)
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = repoDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	combined := stdout.String()
	if stderr.Len() > 0 {
		combined += stderr.String()
	}

	if err != nil {
		return combined, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(combined))
	}
	return combined, nil
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func anyPrefix(list []string, prefix string) bool {
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
