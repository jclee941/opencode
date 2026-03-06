// gen-opencode-config.go merges config/*.jsonc → opencode.jsonc
//
// Usage:
//
//	go run scripts/gen-opencode-config.go          # generate
//	go run scripts/gen-opencode-config.go --check   # verify up-to-date (exit 1 if stale)
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const header = `// AUTO-GENERATED — do not edit directly.
// Source of truth: config/base.jsonc, config/providers.jsonc, config/lsp.jsonc
// Regenerate: go run scripts/gen-opencode-config.go
`

// stripJSONCComments removes // and /* */ comments from JSONC content.
// Uses a state-based parser to avoid stripping // inside string literals (URLs etc).
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

// deepMerge merges src into dst recursively. src values override dst.
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = sv
			continue
		}
		dstMap, dOK := dv.(map[string]interface{})
		srcMap, sOK := sv.(map[string]interface{})
		if dOK && sOK {
			dst[k] = deepMerge(dstMap, srcMap)
		} else {
			dst[k] = sv
		}
	}
	return dst
}

// omoConflicts lists plugins that silently fail when loaded alongside
// oh-my-opencode due to OpenCode's ESM plugin resolution bug.
// See: compressed session b2 — toggle test results.
var omoConflicts = map[string]bool{
	"opencode-morph-fast-apply": true,
	"opencode-smart-title":      true,
	"opencode-convodump":        true,
	"octto":                     true,
	"opencode-supermemory":      true,
	"opencode-scheduler":        true,
	"micode":                    true,
	"opencode-notify":           true,
	"opencode-websearch-cited":  true,
	"opencode-wakatime":         true,
	"@plannotator/opencode":     true,
	"@openspoon/subtask2":       true,
	"opencode-worktree":                true,
	"opencode-zellij-namer":            true,
	"opencode-devcontainers":           true,
	"opencode-daytona":                 true,
	"opencode-helicone-session":        true,
	"opencode-openai-codex-auth":       true,
	"opencode-gemini-auth":             true,

}

// resolvePluginConflicts removes plugins that conflict with oh-my-opencode.
// If OMO is not in the list, all plugins are kept as-is.
func resolvePluginConflicts(merged map[string]interface{}) {
	raw, ok := merged["plugin"]
	if !ok {
		return
	}
	plugins, ok := raw.([]interface{})
	if !ok {
		return
	}

	// Check if OMO is present.
	hasOMO := false
	for _, p := range plugins {
		if s, ok := p.(string); ok && s == "oh-my-opencode" {
			hasOMO = true
			break
		}
	}
	if !hasOMO {
		return
	}

	// Filter out conflicting plugins.
	var kept []interface{}
	var removed []string
	for _, p := range plugins {
		s, ok := p.(string)
		if !ok {
			kept = append(kept, p)
			continue
		}
		// Strip version suffix (e.g., "micode@latest" → "micode")
		name := s
		if idx := strings.Index(s, "@"); idx >= 0 {
			name = s[:idx]
		}
		if omoConflicts[name] {
			removed = append(removed, s)
			continue
		}
		kept = append(kept, p)
	}
	if len(removed) > 0 {
		merged["plugin"] = kept
		fmt.Fprintf(os.Stderr, "OMO conflict resolver: removed %v (ESM resolution conflict)\n", removed)
	}
}

func main() {
	check := len(os.Args) > 1 && os.Args[1] == "--check"

	exe, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	root := exe
	outPath := filepath.Join(root, "opencode.jsonc")

	configFiles := []string{
		filepath.Join(root, "config", "base.jsonc"),
		filepath.Join(root, "config", "providers.jsonc"),
		filepath.Join(root, "config", "lsp.jsonc"),
	}

	merged := make(map[string]interface{})

	for _, path := range configFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
			os.Exit(1)
		}
		clean := stripJSONCComments(data)

		var obj map[string]interface{}
		if err := json.Unmarshal(clean, &obj); err != nil {
			fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", path, err)
			os.Exit(1)
		}
		merged = deepMerge(merged, obj)
	}

	// Resolve OMO plugin conflicts before output.
	resolvePluginConflicts(merged)

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling: %v\n", err)
		os.Exit(1)
	}

	content := header + string(out) + "\n"

	if check {
		existing, err := os.ReadFile(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", outPath, err)
			os.Exit(1)
		}
		if strings.TrimSpace(string(existing)) != strings.TrimSpace(content) {
			fmt.Fprintf(os.Stderr, "opencode.jsonc is stale. Run: npm run gen:config\n")
			os.Exit(1)
		}
		fmt.Println("opencode.jsonc is up-to-date.")
		return
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s from %d config files.\n", outPath, len(configFiles))
}
