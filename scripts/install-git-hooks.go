//go:build ignore
// +build ignore

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

func main() {
	fs := flag.NewFlagSet("install-git-hooks", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Print actions without copying files")
	fs.Parse(os.Args[1:])

	root, err := gitPath("rev-parse", "--show-toplevel")
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve repo root: %v\n", err)
		os.Exit(1)
	}

	hooksDir, err := gitPath("rev-parse", "--git-path", "hooks")
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve hooks dir: %v\n", err)
		os.Exit(1)
	}

	sourceDir := filepath.Join(root, ".githooks")
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", sourceDir, err)
		os.Exit(1)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no hook files found in .githooks")
		os.Exit(1)
	}

	if *dryRun {
		for _, name := range files {
			fmt.Printf("DRY  %s -> %s\n", filepath.Join(sourceDir, name), filepath.Join(hooksDir, name))
		}
		return
	}

	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create hooks dir: %v\n", err)
		os.Exit(1)
	}

	for _, name := range files {
		src := filepath.Join(sourceDir, name)
		dst := filepath.Join(hooksDir, name)
		srcAbs, err := filepath.Abs(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve %s: %v\n", src, err)
			os.Exit(1)
		}
		dstAbs, err := filepath.Abs(dst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve %s: %v\n", dst, err)
			os.Exit(1)
		}
		if srcAbs == dstAbs {
			if err := os.Chmod(dstAbs, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "chmod %s: %v\n", dstAbs, err)
				os.Exit(1)
			}
			fmt.Printf("ACTIVE  %s\n", dst)
			continue
		}
		if err := copyFile(src, dst); err != nil {
			fmt.Fprintf(os.Stderr, "install %s: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("INSTALLED  %s\n", dst)
	}
}

func gitPath(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return filepath.Clean(string(trimSpace(out))), nil
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o755
	}
	return os.Chmod(dst, mode)
}

func trimSpace(data []byte) []byte {
	start := 0
	for start < len(data) && (data[start] == ' ' || data[start] == '\n' || data[start] == '\r' || data[start] == '\t') {
		start++
	}
	end := len(data)
	for end > start && (data[end-1] == ' ' || data[end-1] == '\n' || data[end-1] == '\r' || data[end-1] == '\t') {
		end--
	}
	return data[start:end]
}
