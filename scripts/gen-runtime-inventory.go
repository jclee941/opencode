//go:build ignore

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

const runtimeInventoryHeader = `<!-- AUTO-GENERATED - do not edit directly.
Source of truth: config/base.jsonc, package.json, config/lsp.jsonc, scripts/omo-conflicts.json, and explicit local surface discovery in scripts/gen-runtime-inventory.go
Regenerate: npm run gen:runtime-inventory
-->
`

var runtimeTrailingCommaRE = regexp.MustCompile(`,\s*([}\]])`)

type packageManifest struct {
	Dependencies map[string]string `json:"dependencies"`
}

type omoConflictManifest struct {
	Plugins []string `json:"plugins"`
}

type mcpServer struct {
	Name    string
	Type    string
	Enabled bool
	Target  string
}

type capabilitySurface struct {
	Path    string
	Kind    string
	Status  string
	Backing string
	Schema  string
	Notes   string
}

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
	return []byte(runtimeTrailingCommaRE.ReplaceAllString(out.String(), "$1"))
}

func readJSONCObject(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	clean := stripJSONCComments(data)
	var out map[string]interface{}
	if err := json.Unmarshal(clean, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func readPackageManifest(path string) (packageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return packageManifest{}, err
	}
	var out packageManifest
	if err := json.Unmarshal(data, &out); err != nil {
		return packageManifest{}, err
	}
	if out.Dependencies == nil {
		out.Dependencies = map[string]string{}
	}
	return out, nil
}

func pluginName(plugin string) string {
	idx := strings.LastIndex(plugin, "@")
	if idx <= 0 {
		return plugin
	}

	if strings.HasPrefix(plugin, "@") {
		slash := strings.Index(plugin, "/")
		if slash < 0 || idx <= slash {
			return plugin
		}
	}

	suffix := plugin[idx+1:]
	if suffix == "latest" || suffix == "beta" || strings.HasPrefix(suffix, "^") {
		return plugin[:idx]
	}

	return plugin
}

func extractPlugins(base map[string]interface{}) []string {
	raw, ok := base["plugin"].([]interface{})
	if !ok {
		return nil
	}
	plugins := make([]string, 0, len(raw))
	for _, item := range raw {
		value, ok := item.(string)
		if ok {
			plugins = append(plugins, value)
		}
	}
	return plugins
}

func extractMCPServers(base map[string]interface{}) []mcpServer {
	raw, ok := base["mcp"].(map[string]interface{})
	if !ok {
		return nil
	}
	servers := make([]mcpServer, 0, len(raw))
	for name, value := range raw {
		obj, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		target := ""
		if url, ok := obj["url"].(string); ok {
			target = url
		} else if command, ok := obj["command"].([]interface{}); ok {
			parts := make([]string, 0, len(command))
			for _, item := range command {
				if s, ok := item.(string); ok {
					parts = append(parts, s)
				}
			}
			target = strings.Join(parts, " ")
		}
		servers = append(servers, mcpServer{
			Name:    name,
			Type:    stringValue(obj["type"]),
			Enabled: boolValue(obj["enabled"]),
			Target:  target,
		})
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})
	return servers
}

func extractSchemaMap(lsp map[string]interface{}) map[string]string {
	result := map[string]string{}
	lspMap, ok := lsp["lsp"].(map[string]interface{})
	if !ok {
		return result
	}
	jsonServer, ok := lspMap["json"].(map[string]interface{})
	if !ok {
		return result
	}
	init, ok := jsonServer["initialization"].(map[string]interface{})
	if !ok {
		return result
	}
	jsonInit, ok := init["json"].(map[string]interface{})
	if !ok {
		return result
	}
	entries, ok := jsonInit["schemas"].([]interface{})
	if !ok {
		return result
	}
	for _, entry := range entries {
		obj, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		url := stringValue(obj["url"])
		matches, ok := obj["fileMatch"].([]interface{})
		if !ok {
			continue
		}
		for _, match := range matches {
			if s, ok := match.(string); ok {
				result[s] = url
			}
		}
	}
	return result
}

func readOMOConflicts(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest omoConflictManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	conflicts := append([]string(nil), manifest.Plugins...)
	if len(conflicts) == 0 {
		return nil, fmt.Errorf("no OMO conflicts found in %s", path)
	}
	sort.Strings(conflicts)
	return conflicts, nil
}

func discoverCapabilities(root string, active map[string]bool, schemaMap map[string]string, conflictSet map[string]bool) []capabilitySurface {
	hasOMO := active["oh-my-opencode"]
	surfaces := []capabilitySurface{}

	appendSurface := func(path, kind, backing, notes string) {
		abs := filepath.Join(root, path)
		if _, err := os.Stat(abs); err != nil {
			return
		}
		status := "supporting"
		switch kind {
		case "plugin-config":
			if active[backing] {
				status = "active"
			} else if hasOMO && conflictSet[backing] {
				status = "present, blocked with OMO"
			} else {
				status = "present, inactive"
			}
		case "skill":
			status = "available"
		}
		surfaces = append(surfaces, capabilitySurface{
			Path:    path,
			Kind:    kind,
			Status:  status,
			Backing: backing,
			Schema:  schemaMap[path],
			Notes:   notes,
		})
	}

	appendSurface("oh-my-opencode.jsonc", "plugin-config", "oh-my-opencode", "Primary orchestration, routing, fallback, browser, and search settings.")
	appendSurface("dcp.jsonc", "plugin-config", "@tarquinen/opencode-dcp", "Dynamic context pruning policy and protected tool settings.")
	appendSurface("pilot/config.yaml", "plugin-config", "opencode-pilot", "GitHub work polling and session defaults.")
	appendSurface("smart-title.jsonc", "plugin-config", "opencode-smart-title", "Config file exists locally even though the plugin is not active.")
	appendSurface("subtask2.jsonc", "plugin-config", "@openspoon/subtask2", "Config file exists locally even though the plugin is not active.")
	appendSurface("snippet/config.jsonc", "supporting-config", "", "Snippet output and skill rendering settings.")

	skillPaths, _ := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	sort.Strings(skillPaths)
	for _, abs := range skillPaths {
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			continue
		}
		name := filepath.Base(filepath.Dir(abs))
		notes := "Policy or capability skill surface."
		if name == "dev-browser" {
			notes = "Standalone executable browser automation skill package."
		}
		appendSurface(rel, "skill", name, notes)
	}

	sort.Slice(surfaces, func(i, j int) bool {
		return surfaces[i].Path < surfaces[j].Path
	})
	return surfaces
}

func stringValue(value interface{}) string {
	s, _ := value.(string)
	return s
}

func boolValue(value interface{}) bool {
	b, _ := value.(bool)
	return b
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	if value == "" {
		return "-"
	}
	return value
}

func configSurfaceForPlugin(name string) string {
	switch name {
	case "oh-my-opencode":
		return "oh-my-opencode.jsonc"
	case "@tarquinen/opencode-dcp":
		return "dcp.jsonc"
	case "opencode-pilot":
		return "pilot/config.yaml"
	case "opencode-smart-title":
		return "smart-title.jsonc"
	case "@openspoon/subtask2":
		return "subtask2.jsonc"
	default:
		return ""
	}
}

func pluginNotes(name string) string {
	switch name {
	case "oh-my-opencode":
		return "Must remain last in config/base.jsonc; activates OMO conflict filtering."
	case "@tarquinen/opencode-dcp":
		return "Active dynamic context pruning plugin with dedicated root config."
	case "opencode-pilot":
		return "Active queue/polling plugin with pilot/config.yaml."
	case "opencode-pty":
		return "Interactive PTY support for long-running shell sessions."
	case "@nick-vi/opencode-type-inject":
		return "Type/tool augmentation plugin loaded from npm dependency entry."
	default:
		return ""
	}
}

func renderInventory(root string) (string, error) {
	base, err := readJSONCObject(filepath.Join(root, "config", "base.jsonc"))
	if err != nil {
		return "", fmt.Errorf("read config/base.jsonc: %w", err)
	}
	lsp, err := readJSONCObject(filepath.Join(root, "config", "lsp.jsonc"))
	if err != nil {
		return "", fmt.Errorf("read config/lsp.jsonc: %w", err)
	}
	pkg, err := readPackageManifest(filepath.Join(root, "package.json"))
	if err != nil {
		return "", fmt.Errorf("read package.json: %w", err)
	}
	conflicts, err := readOMOConflicts(filepath.Join(root, "scripts", "omo-conflicts.json"))
	if err != nil {
		return "", err
	}

	plugins := extractPlugins(base)
	servers := extractMCPServers(base)
	schemaMap := extractSchemaMap(lsp)
	active := map[string]bool{}
	for _, plugin := range plugins {
		active[pluginName(plugin)] = true
	}
	conflictSet := map[string]bool{}
	for _, name := range conflicts {
		conflictSet[name] = true
	}
	surfaces := discoverCapabilities(root, active, schemaMap, conflictSet)

	var out strings.Builder
	out.WriteString(runtimeInventoryHeader)
	out.WriteString("# Runtime Inventory\n\n")
	out.WriteString("This report stays intentionally narrow: it shows load-time plugin and MCP visibility from `config/base.jsonc`, the OMO conflict watchlist from `scripts/omo-conflicts.json`, and explicit local capability surfaces that commonly change decision-making in this repo. It does not attempt to expand every nested setting inside files like `oh-my-opencode.jsonc`.\n\n")
	out.WriteString("## Scope\n\n")
	out.WriteString("- Show active plugins requested in `config/base.jsonc`.\n")
	out.WriteString("- Show configured MCP servers from `config/base.jsonc`.\n")
	out.WriteString("- Show explicit local capability surfaces that affect operator decisions in this repo.\n")
	out.WriteString("- Show the OMO conflict watchlist derived from `scripts/omo-conflicts.json`.\n\n")
	out.WriteString("## Inputs/constraints\n\n")
	out.WriteString("- Source-of-truth plugin and MCP declarations come from `config/base.jsonc`.\n")
	out.WriteString("- Package install versions come from `package.json`.\n")
	out.WriteString("- Schema coverage comes from `config/lsp.jsonc`.\n")
	out.WriteString("- OMO conflict truth comes from `scripts/omo-conflicts.json`.\n")
	out.WriteString("- This report is static visibility only; it does not perform live MCP health checks.\n\n")
	out.WriteString("## Decision/rules\n\n")
	out.WriteString("- Keep the report deterministic and generated.\n")
	out.WriteString("- Do not duplicate OMO conflict data outside the generator source.\n")
	out.WriteString("- Prefer explicit file-path visibility over interpreting every nested runtime setting.\n\n")
	out.WriteString("## Verification\n\n")
	out.WriteString("- Regenerate with `npm run gen:runtime-inventory`.\n")
	out.WriteString("- Check freshness with `npm run gen:runtime-inventory:check`.\n")
	out.WriteString("- `npm run prepush:check` also enforces this report stays current.\n\n")
	out.WriteString("## Rollback/safety\n\n")
	out.WriteString("- This artifact is additive and does not change runtime plugin or MCP behavior.\n")
	out.WriteString("- Remove the generator, generated doc, and npm script wiring together to revert this feature.\n\n")
	out.WriteString(fmt.Sprintf("- Active plugins: %d\n", len(plugins)))
	out.WriteString(fmt.Sprintf("- Configured MCP servers: %d\n", len(servers)))
	out.WriteString(fmt.Sprintf("- Capability surfaces tracked: %d\n", len(surfaces)))
	out.WriteString(fmt.Sprintf("- OMO conflict watchlist entries: %d\n\n", len(conflicts)))

	out.WriteString("## Active Plugins\n\n")
	out.WriteString("| Order | Plugin | Requested entry | Dependency version | Config surface | Notes |\n")
	out.WriteString("|---|---|---|---|---|---|\n")
	for i, raw := range plugins {
		name := pluginName(raw)
		out.WriteString(fmt.Sprintf("| %d | `%s` | `%s` | `%s` | `%s` | %s |\n",
			i+1,
			escapeTable(name),
			escapeTable(raw),
			escapeTable(pkg.Dependencies[name]),
			escapeTable(configSurfaceForPlugin(name)),
			escapeTable(pluginNotes(name)),
		))
	}
	out.WriteString("\n")

	out.WriteString("## Configured MCP Servers\n\n")
	out.WriteString("| Name | Type | Enabled | Target |\n")
	out.WriteString("|---|---|---|---|\n")
	for _, server := range servers {
		enabled := "false"
		if server.Enabled {
			enabled = "true"
		}
		out.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` |\n",
			escapeTable(server.Name),
			escapeTable(server.Type),
			escapeTable(enabled),
			escapeTable(server.Target),
		))
	}
	out.WriteString("\n")

	out.WriteString("## Capability Surfaces\n\n")
	out.WriteString("| Path | Kind | Status | Backing | Schema | Notes |\n")
	out.WriteString("|---|---|---|---|---|---|\n")
	for _, surface := range surfaces {
		out.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | %s |\n",
			escapeTable(surface.Path),
			escapeTable(surface.Kind),
			escapeTable(surface.Status),
			escapeTable(surface.Backing),
			escapeTable(surface.Schema),
			escapeTable(surface.Notes),
		))
	}
	out.WriteString("\n")

	out.WriteString("## OMO Conflict Watchlist\n\n")
	out.WriteString("These entries are auto-removed when `oh-my-opencode` is present, based on the shared conflict list in `scripts/omo-conflicts.json` and the resolver in `scripts/gen-opencode-config.go`.\n\n")
	for _, conflict := range conflicts {
		out.WriteString(fmt.Sprintf("- `%s`\n", conflict))
	}
	out.WriteString("\n")

	return out.String(), nil
}

func main() {
	check := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--check":
			check = true
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\nUsage: go run scripts/gen-runtime-inventory.go [--check]\n", arg)
			os.Exit(1)
		}
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	content, err := renderInventory(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	outPath := filepath.Join(root, "docs", "runtime-inventory.md")
	if check {
		existing, err := os.ReadFile(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", outPath, err)
			os.Exit(1)
		}
		if strings.TrimSpace(string(existing)) != strings.TrimSpace(content) {
			fmt.Fprintf(os.Stderr, "docs/runtime-inventory.md is stale. Run: npm run gen:runtime-inventory\n")
			os.Exit(1)
		}
		fmt.Println("docs/runtime-inventory.md is up-to-date.")
		return
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s from config/base.jsonc, config/lsp.jsonc, package.json, and scripts/omo-conflicts.json\n", outPath)
}
