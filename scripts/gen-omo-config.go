//go:build ignore

// gen-omo-config.go resolves $models variables in config/omo.jsonc
// and writes oh-my-opencode.jsonc.
//
// Usage:
//
//	go run scripts/gen-omo-config.go          # generate
//	go run scripts/gen-omo-config.go --check   # verify up-to-date (exit 1 if stale)
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const header = `// AUTO-GENERATED — do not edit directly.
// Source: config/omo.jsonc
// Regenerate: go run scripts/gen-omo-config.go
`

// stripJSONCComments removes // and /* */ comments from JSONC content.
// Uses a state-based parser to avoid stripping // inside string literals.
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
	re := regexp.MustCompile(`,\s*([}\]])`)
	return []byte(re.ReplaceAllString(out.String(), "$1"))
}

func main() {
	check := len(os.Args) > 1 && os.Args[1] == "--check"

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	srcPath := filepath.Join(root, "config", "omo.jsonc")
	outPath := filepath.Join(root, "oh-my-opencode.jsonc")

	data, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", srcPath, err)
		os.Exit(1)
	}

	clean := stripJSONCComments(data)

	var obj map[string]interface{}
	if err := json.Unmarshal(clean, &obj); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", srcPath, err)
		os.Exit(1)
	}

	// Extract and remove $models.
	modelsRaw, ok := obj["$models"]
	if !ok {
		fmt.Fprintf(os.Stderr, "no $models section in %s\n", srcPath)
		os.Exit(1)
	}
	modelsMap, ok := modelsRaw.(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "$models is not an object\n")
		os.Exit(1)
	}
	delete(obj, "$models")

	// Build ordered replacement list (longest variable names first to avoid
	// partial matches like $CLAUDE matching before $CLAUDE_OPUS).
	type replacement struct {
		varName  string
		modelStr string
	}
	var replacements []replacement
	for k, v := range modelsMap {
		s, ok := v.(string)
		if !ok {
			fmt.Fprintf(os.Stderr, "$models.%s is not a string\n", k)
			os.Exit(1)
		}
		replacements = append(replacements, replacement{k, s})
	}
	sort.Slice(replacements, func(i, j int) bool {
		return len(replacements[i].varName) > len(replacements[j].varName)
	})

	// Marshal to JSON.
	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling: %v\n", err)
		os.Exit(1)
	}

	// Resolve $VARIABLE references → actual model strings.
	content := string(out)
	for _, r := range replacements {
		content = strings.ReplaceAll(content, r.varName, r.modelStr)
	}

	final := header + content + "\n"

	if check {
		existing, err := os.ReadFile(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", outPath, err)
			os.Exit(1)
		}
		if strings.TrimSpace(string(existing)) != strings.TrimSpace(final) {
			fmt.Fprintf(os.Stderr, "oh-my-opencode.jsonc is stale. Run: npm run gen:omo\n")
			os.Exit(1)
		}
		fmt.Println("oh-my-opencode.jsonc is up-to-date.")
		return
	}

	if err := os.WriteFile(outPath, []byte(final), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s (resolved %d model variables).\n", outPath, len(replacements))
}
