// config-recover.go - Recover changes made directly to opencode.jsonc
//
// Usage:
//
//	go run scripts/config-recover.go           # Check for uncommitted changes in opencode.jsonc
//	go run scripts/config-recover.go --apply   # Attempt to migrate changes to source files
//	go run scripts/config-recover.go --reset   # Discard opencode.jsonc changes and regenerate
//
// This tool helps recover from accidental direct edits to the auto-generated
// opencode.jsonc file by either:
// 1. Showing what changes were made (diff mode)
// 2. Attempting to apply changes to the appropriate config/*.jsonc source
// 3. Resetting and regenerating from source

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const usage = `config-recover: Recover from direct edits to opencode.jsonc

opencode.jsonc is AUTO-GENERATED from config/*.jsonc source files.
Direct edits will be LOST on regeneration.

Commands:
  (no args)    Show diff between current opencode.jsonc and regenerated version
  --apply      Attempt to migrate changes to config/*.jsonc source files
  --reset      Discard changes and regenerate from config/*.jsonc
  --help       Show this help message

Examples:
  go run scripts/config-recover.go
  go run scripts/config-recover.go --apply
  go run scripts/config-recover.go --reset
`

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Print(usage)
		os.Exit(0)
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	opencodePath := filepath.Join(root, "opencode.jsonc")

	// Check if opencode.jsonc has uncommitted changes
	if !hasUncommittedChanges(opencodePath) {
		fmt.Println("✓ opencode.jsonc has no uncommitted changes.")
		fmt.Println("  Your config is in sync with source files.")
		os.Exit(0)
	}

	mode := "diff"
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--apply":
			mode = "apply"
		case "--reset":
			mode = "reset"
		default:
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n\n%s", os.Args[1], usage)
			os.Exit(1)
		}
	}

	switch mode {
	case "diff":
		showDiff(root, opencodePath)
	case "apply":
		applyChanges(root, opencodePath)
	case "reset":
		resetChanges(root, opencodePath)
	}
}

func hasUncommittedChanges(path string) bool {
	cmd := exec.Command("git", "diff", "HEAD", "--", path)
	cmd.Dir = filepath.Dir(path)
	output, err := cmd.Output()
	if err != nil {
		// If git fails, assume no changes (file might not be tracked)
		return false
	}
	return len(output) > 0
}

func showDiff(root, opencodePath string) {
	// Generate fresh config
	cmd := exec.Command("go", "run", "scripts/gen-opencode-config.go")
	cmd.Dir = root
	freshJSON, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating fresh config: %v\n", err)
		os.Exit(1)
	}

	// Write to temp file
	tmpFile := filepath.Join(root, ".opencode.jsonc.fresh")
	if err := os.WriteFile(tmpFile, freshJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing temp file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile)

	// Show diff
	fmt.Println("=== Changes in opencode.jsonc (will be LOST on regeneration) ===")
	fmt.Println("")

	diffCmd := exec.Command("git", "diff", "--no-index", "--", tmpFile, opencodePath)
	diffCmd.Dir = root
	diffOutput, _ := diffCmd.CombinedOutput()

	// Remove the temp file indicators from diff output
	lines := strings.Split(string(diffOutput), "\n")
	for _, line := range lines {
		if strings.Contains(line, ".opencode.jsonc.fresh") {
			line = strings.Replace(line, ".opencode.jsonc.fresh", "opencode.jsonc (fresh)", -1)
		}
		fmt.Println(line)
	}

	fmt.Println("")
	fmt.Println("⚠️  These changes will be LOST when config is regenerated.")
	fmt.Println("")
	fmt.Println("To apply changes to source files:")
	fmt.Println("  go run scripts/config-recover.go --apply")
	fmt.Println("")
	fmt.Println("To discard changes and regenerate:")
	fmt.Println("  go run scripts/config-recover.go --reset")
}

func applyChanges(root, opencodePath string) {
	fmt.Println("🔄 Attempting to migrate changes to config/*.jsonc...")
	fmt.Println("")
	fmt.Println("⚠️  This is a best-effort operation. Please review changes carefully.")
	fmt.Println("")

	// Read current (modified) opencode.jsonc
	currentData, err := os.ReadFile(opencodePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading opencode.jsonc: %v\n", err)
		os.Exit(1)
	}

	// Read source files
	sourceFiles := map[string]string{
		"base":      filepath.Join(root, "config", "base.jsonc"),
		"providers": filepath.Join(root, "config", "providers.jsonc"),
		"lsp":       filepath.Join(root, "config", "lsp.jsonc"),
	}

	// Parse current config
	var currentConfig map[string]interface{}
	if err := json.Unmarshal(stripJSONCComments(currentData), &currentConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing opencode.jsonc: %v\n", err)
		os.Exit(1)
	}

	// For now, just show what changes need manual migration
	fmt.Println("=== Manual Migration Required ===")
	fmt.Println("")
	fmt.Println("Auto-migration is not yet implemented. Please manually transfer your changes:")
	fmt.Println("")

	for name, path := range sourceFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [!] Cannot read %s: %v\n", path, err)
			continue
		}

		var sourceConfig map[string]interface{}
		if err := json.Unmarshal(stripJSONCComments(data), &sourceConfig); err != nil {
			fmt.Fprintf(os.Stderr, "  [!] Cannot parse %s: %v\n", path, err)
			continue
		}

		fmt.Printf("📄 %s (%s):\n", name, path)
		findDifferences(sourceConfig, currentConfig, "  ")
		fmt.Println("")
	}

	fmt.Println("After editing source files, regenerate with:")
	fmt.Println("  npm run gen:config")
}

func findDifferences(source, current map[string]interface{}, indent string) {
	// Simple diff - show keys that differ
	for key, currentVal := range current {
		if sourceVal, exists := source[key]; exists {
			// Key exists in both - check if values differ
			currentJSON, _ := json.Marshal(currentVal)
			sourceJSON, _ := json.Marshal(sourceVal)
			if string(currentJSON) != string(sourceJSON) {
				fmt.Printf("%s[~] %s: value differs\n", indent, key)
			}
		} else {
			// Key only in current (user added it)
			fmt.Printf("%s[+] %s: added in opencode.jsonc\n", indent, key)
		}
	}

	for key := range source {
		if _, exists := current[key]; !exists {
			// Key only in source (user removed it)
			fmt.Printf("%s[-] %s: removed in opencode.jsonc\n", indent, key)
		}
	}
}

func resetChanges(root, opencodePath string) {
	fmt.Println("🗑️  Discarding changes to opencode.jsonc...")

	// Git checkout to discard changes
	cmd := exec.Command("git", "checkout", "--", "opencode.jsonc")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error discarding changes: %v\n", err)
		os.Exit(1)
	}

	// Regenerate fresh
	genCmd := exec.Command("go", "run", "scripts/gen-opencode-config.go")
	genCmd.Dir = root
	genCmd.Stdout = os.Stdout
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error regenerating config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println("✓ opencode.jsonc has been reset and regenerated from config/*.jsonc")
}

// stripJSONCComments removes // and /* */ comments from JSONC content
func stripJSONCComments(data []byte) []byte {
	s := string(data)
	var out strings.Builder
	out.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '"' {
			out.WriteByte(s[i])
			i++
			for i < len(s) {
				out.WriteByte(s[i])
				if s[i] == '\\' {
					i++
					if i < len(s) {
						out.WriteByte(s[i])
					}
				} else if s[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			}
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return []byte(out.String())
}
