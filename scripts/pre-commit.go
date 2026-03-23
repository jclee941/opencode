// pre-commit.go - Git pre-commit hook for OpenCode config repository
//
// This hook runs before each commit to prevent common mistakes:
// 1. Blocks direct edits to auto-generated opencode.jsonc
// 2. Runs config generation check to ensure opencode.jsonc is up-to-date
// 3. Validates naming conventions
// 4. Validates config cross-references
//
// Installation:
//
//	go run scripts/install-git-hooks.go
//
// Or manually:
//
//	cp scripts/pre-commit.go .git/hooks/pre-commit
//	chmod +x .git/hooks/pre-commit

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	// Check for direct edits to opencode.jsonc
	if err := checkOpenCodeDirectEdit(root); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Run config generation check
	if err := runConfigCheck(root); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Run naming validation
	if err := runNamingCheck(root); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Run config refs validation
	if err := runConfigRefsCheck(root); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ All pre-commit checks passed.")
}

func checkOpenCodeDirectEdit(root string) error {
	// Check if opencode.jsonc is staged for commit
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error checking staged files: %v", err)
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
	hasOpenCode := false
	hasConfigSource := false

	for _, file := range stagedFiles {
		if file == "opencode.jsonc" {
			hasOpenCode = true
		}
		if strings.HasPrefix(file, "config/") && strings.HasSuffix(file, ".jsonc") {
			hasConfigSource = true
		}
	}

	if !hasOpenCode {
		return nil // opencode.jsonc not staged, no issue
	}

	// Check if this is a legitimate regeneration (config sources also changed)
	if hasConfigSource {
		// Config sources are being modified - likely legitimate
		// But still verify the generated file is correct
		fmt.Println("⚠️  opencode.jsonc is staged with config/ changes - verifying...")
		return nil
	}

	// Direct edit to opencode.jsonc without source changes - BLOCK
	return fmt.Errorf(`❌ BLOCKED: Direct edit to opencode.jsonc detected!

opencode.jsonc is AUTO-GENERATED from config/*.jsonc source files:
  - config/base.jsonc
  - config/providers.jsonc
  - config/lsp.jsonc

Your changes will be LOST when the config is regenerated.

To make changes:
  1. Edit the source files in config/ instead
  2. Run: npm run gen:config
  3. Stage both config/ changes and the regenerated opencode.jsonc

To recover changes from this edit:
  go run scripts/config-recover.go

To see what changes you made:
  go run scripts/config-recover.go --diff

To discard changes and regenerate:
  go run scripts/config-recover.go --reset

To force commit anyway (NOT recommended):
  git commit --no-verify

See: AGENTS.md for configuration workflow documentation.`)
}

func runConfigCheck(root string) error {
	fmt.Println("→ Checking if opencode.jsonc is up-to-date...")

	cmd := exec.Command("go", "run", "scripts/gen-opencode-config.go", "--check")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(`❌ Config check failed: opencode.jsonc is stale or invalid

%s

To regenerate:
  npm run gen:config

Then stage the changes and commit again.`, string(output))
	}

	fmt.Printf("  %s", output)
	return nil
}

func runNamingCheck(root string) error {
	fmt.Println("→ Running naming convention check...")

	cmd := exec.Command("node", "scripts/validate-monorepo-naming.mjs")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(`❌ Naming validation failed:

%s`, string(output))
	}

	fmt.Printf("  %s", output)
	return nil
}

func runConfigRefsCheck(root string) error {
	fmt.Println("→ Running config cross-reference check...")

	cmd := exec.Command("go", "run", "scripts/validate-config-refs.go")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(`❌ Config reference validation failed:

%s`, string(output))
	}

	fmt.Printf("  %s", output)
	return nil
}
