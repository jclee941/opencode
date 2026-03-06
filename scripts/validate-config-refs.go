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

var trailingCommaRE = regexp.MustCompile(`,\s*([}\]])`)

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
	return []byte(trailingCommaRE.ReplaceAllString(out.String(), "$1"))
}

type modelRef struct {
	Path  string
	Model string
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	files := []string{
		"config/base.jsonc",
		"config/providers.jsonc",
		"config/lsp.jsonc",
		"opencode.jsonc",
		"oh-my-opencode.jsonc",
		"antigravity.jsonc",
		"dcp.jsonc",
		"opencode-mem.jsonc",
		"subtask2.jsonc",
		"smart-title.jsonc",
		"snippet/config.jsonc",
	}

	violations := make([]string, 0)
	parsed := make(map[string]map[string]interface{})

	for _, rel := range files {
		full := filepath.Join(root, rel)
		data, readErr := os.ReadFile(full)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				violations = append(violations, fmt.Sprintf("MISSING_FILE  %s", rel))
				continue
			}
			violations = append(violations, fmt.Sprintf("PARSE_ERROR   %s: %v", rel, readErr))
			continue
		}

		clean := stripJSONCComments(data)
		var obj map[string]interface{}
		if unmarshalErr := json.Unmarshal(clean, &obj); unmarshalErr != nil {
			violations = append(violations, fmt.Sprintf("PARSE_ERROR   %s: %v", rel, unmarshalErr))
			continue
		}
		parsed[rel] = obj
	}

	providerGoogleModels := extractGoogleModelKeys(parsed["config/providers.jsonc"])
	modelRefs := collectModelRefs(parsed["oh-my-opencode.jsonc"], parsed["config/base.jsonc"])

	uniqueModels := make(map[string]struct{})
	for _, ref := range modelRefs {
		if ref.Model == "" {
			continue
		}
		uniqueModels[ref.Model] = struct{}{}
		if !isValidModelReference(ref.Model, providerGoogleModels) {
			if strings.HasPrefix(ref.Model, "google/") {
				violations = append(violations, fmt.Sprintf("MODEL_REF     %s: %q not found in providers", ref.Path, ref.Model))
			} else {
				violations = append(violations, fmt.Sprintf("MODEL_REF     %s: %q has unsupported provider prefix", ref.Path, ref.Model))
			}
		}
	}

	dcpModelLimits := extractDCPModelLimitKeys(parsed["dcp.jsonc"])
	models := make([]string, 0, len(uniqueModels))
	for m := range uniqueModels {
		models = append(models, m)
	}
	sort.Strings(models)
	for _, m := range models {
		if _, ok := dcpModelLimits[m]; !ok {
			violations = append(violations, fmt.Sprintf("DCP_MISSING   model %q has no entry in dcp.jsonc modelLimits", m))
		}
	}

	deps := extractPackageDependencies(root)
	for _, plugin := range collectBasePlugins(parsed["config/base.jsonc"]) {
		normalized := normalizePluginDependencyName(plugin)
		if normalized == "" || normalized == "oh-my-opencode" {
			continue
		}
		if _, ok := deps[normalized]; !ok {
			violations = append(violations, fmt.Sprintf("PLUGIN_DEP    plugin %q has no matching package.json dependency", normalized))
		}
	}

	if len(violations) > 0 {
		for _, v := range violations {
			fmt.Println(v)
		}
		fmt.Printf("\nFound %d config reference violations.\n", len(violations))
		os.Exit(1)
	}

	fmt.Println("Config references check passed.")
}

func extractGoogleModelKeys(providers map[string]interface{}) map[string]struct{} {
	out := make(map[string]struct{})
	providerMap := getMap(providers, "provider")
	googleMap := getMap(providerMap, "google")
	modelsMap := getMap(googleMap, "models")
	for k := range modelsMap {
		out[k] = struct{}{}
	}
	return out
}

func collectModelRefs(ohMy map[string]interface{}, base map[string]interface{}) []modelRef {
	refs := make([]modelRef, 0)

	categories := getMap(ohMy, "categories")
	categoryNames := make([]string, 0, len(categories))
	for k := range categories {
		categoryNames = append(categoryNames, k)
	}
	sort.Strings(categoryNames)
	for _, name := range categoryNames {
		category := getMap(categories, name)
		if model, ok := getString(category, "model"); ok {
			refs = append(refs, modelRef{Path: "categories." + name + ".model", Model: model})
		}
		fallbacks := getStringSlice(category, "fallback_models")
		for i, model := range fallbacks {
			refs = append(refs, modelRef{Path: fmt.Sprintf("categories.%s.fallback_models[%d]", name, i), Model: model})
		}
	}

	agents := getMap(ohMy, "agents")
	agentNames := make([]string, 0, len(agents))
	for k := range agents {
		agentNames = append(agentNames, k)
	}
	sort.Strings(agentNames)
	for _, name := range agentNames {
		agent := getMap(agents, name)
		if model, ok := getString(agent, "model"); ok {
			refs = append(refs, modelRef{Path: "agents." + name + ".model", Model: model})
		}
		fallbacks := getStringSlice(agent, "fallback_models")
		for i, model := range fallbacks {
			refs = append(refs, modelRef{Path: fmt.Sprintf("agents.%s.fallback_models[%d]", name, i), Model: model})
		}
		compaction := getMap(agent, "compaction")
		if model, ok := getString(compaction, "model"); ok {
			refs = append(refs, modelRef{Path: "agents." + name + ".compaction.model", Model: model})
		}
	}

	backgroundTask := getMap(ohMy, "background_task")
	modelConcurrency := getMap(backgroundTask, "modelConcurrency")
	concurrencyModels := make([]string, 0, len(modelConcurrency))
	for k := range modelConcurrency {
		concurrencyModels = append(concurrencyModels, k)
	}
	sort.Strings(concurrencyModels)
	for _, model := range concurrencyModels {
		refs = append(refs, modelRef{Path: "background_task.modelConcurrency." + model, Model: model})
	}

	if model, ok := getString(base, "small_model"); ok {
		refs = append(refs, modelRef{Path: "small_model", Model: model})
	}

	return refs
}

func isValidModelReference(model string, googleModels map[string]struct{}) bool {
	if strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "minimax-coding-plan/") || strings.HasPrefix(model, "opencode-go/") {
		return true
	}
	if strings.HasPrefix(model, "google/") {
		stripped := strings.TrimPrefix(model, "google/")
		_, ok := googleModels[stripped]
		return ok
	}
	return false
}

func extractDCPModelLimitKeys(dcp map[string]interface{}) map[string]struct{} {
	out := make(map[string]struct{})
	tools := getMap(dcp, "tools")
	settings := getMap(tools, "settings")
	modelLimits := getMap(settings, "modelLimits")
	for k := range modelLimits {
		out[k] = struct{}{}
	}
	return out
}

func extractPackageDependencies(root string) map[string]struct{} {
	out := make(map[string]struct{})
	path := filepath.Join(root, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return out
	}

	deps := getMap(obj, "dependencies")
	for k := range deps {
		out[k] = struct{}{}
	}
	return out
}

func collectBasePlugins(base map[string]interface{}) []string {
	pluginsAny, ok := base["plugin"]
	if !ok {
		return nil
	}
	items, ok := pluginsAny.([]interface{})
	if !ok {
		return nil
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if ok {
			out = append(out, s)
		}
	}
	return out
}

func normalizePluginDependencyName(plugin string) string {
	name := plugin
	idx := strings.LastIndex(name, "@")
	if idx <= 0 {
		return name
	}

	if strings.HasPrefix(name, "@") {
		slash := strings.Index(name, "/")
		if slash < 0 || idx <= slash {
			return name
		}
	}

	suffix := name[idx+1:]
	if suffix == "latest" || suffix == "beta" || strings.HasPrefix(suffix, "^") {
		return name[:idx]
	}
	return name
}

func getMap(obj map[string]interface{}, key string) map[string]interface{} {
	if obj == nil {
		return map[string]interface{}{}
	}
	v, ok := obj[key]
	if !ok {
		return map[string]interface{}{}
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return m
}

func getString(obj map[string]interface{}, key string) (string, bool) {
	if obj == nil {
		return "", false
	}
	v, ok := obj[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func getStringSlice(obj map[string]interface{}, key string) []string {
	if obj == nil {
		return nil
	}
	v, ok := obj[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))

	for _, item := range arr {
		s, ok := item.(string)
		if ok {
			out = append(out, s)
		}
	}
	return out
}
