package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	return cmd.Run()
}

func exitWith(err error) {
	if err == nil {
		return
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	os.Exit(1)
}

func main() {
	headless := flag.Bool("headless", false, "start in headless mode")
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Unknown parameter: %s\n", flag.Arg(0))
		os.Exit(1)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "Failed to locate server.go")
		os.Exit(1)
	}

	scriptDir := filepath.Dir(thisFile)
	if err := os.Chdir(scriptDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Installing dependencies...")
	if err := run("npm", "install"); err != nil {
		fmt.Fprintf(os.Stderr, "npm install failed: %v\n", err)
		fmt.Println("Continuing to start server...")
	}

	fmt.Println("Starting dev-browser server...")
	if err := os.Setenv("HEADLESS", fmt.Sprintf("%t", *headless)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set HEADLESS env: %v\n", err)
		os.Exit(1)
	}

	if err := run("npx", "tsx", "scripts/start-server.ts"); err != nil {
		exitWith(err)
	}
}
