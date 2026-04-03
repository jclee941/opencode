// pre-push.go - Git pre-push hook for OpenCode config repository
//
// This hook runs before each push to ensure code quality:
// 1. Verifies opencode.jsonc is up-to-date with config/*.jsonc
// 2. Validates monorepo naming conventions
// 3. Validates config cross-references
//
// Installation:
//
//	go run scripts/install-git-hooks.go

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("→ Running pre-push checks...")
	fmt.Println("")

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

	fmt.Println("")
	fmt.Println("✓ All pre-push checks passed.")
}

func runConfigCheck(root string) error {
	fmt.Println("  → Checking if opencode.jsonc is up-to-date...")

	cmd := exec.Command("go", "run", "scripts/gen-opencode-config.go", "--check")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(`❌ Config check failed: opencode.jsonc is stale or invalid

%s

To fix:
  npm run gen:config
  git add opencode.jsonc
  git commit -m "chore: regenerate config"

Then push again.`, string(output))
	}

	fmt.Printf("    %s", output)
	return nil
}

func runNamingCheck(root string) error {
	fmt.Println("  → Running naming convention check...")

	cmd := exec.Command("node", "scripts/validate-monorepo-naming.mjs")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(`❌ Naming validation failed:

%s

Fix naming issues and try again.`, string(output))
	}

	fmt.Printf("    %s", output)
	return nil
}

func runConfigRefsCheck(root string) error {
	fmt.Println("  → Running config cross-reference check...")

	cmd := exec.Command("go", "run", "scripts/validate-config-refs.go")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(`❌ Config reference validation failed:

%s

Fix config reference issues and try again.`, string(output))
	}

	fmt.Printf("    %s", output)
	return nil
}
