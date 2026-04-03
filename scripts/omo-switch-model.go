//go:build ignore

// omo-switch-model.go - OMO 설정에서 Claude Opus와 Kimi 모델 간 전환 스크립트
//
// 사용법:
//   sw                # 현재 모델과 반대로 전환 (toggle)
//   sw opus           # Claude Opus로 전환
//   sw kimi           # Kimi로 전환
//   sw --list         # 현재 상태 확인
//
// 동작:
//   - config/base.jsonc의 "model" 필드를 업데이트
//   - config/omo.jsonc의 sisyphus agent model을 업데이트
//   - gen-opencode-config.go를 실행하여 설정 재생성

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	modelOpus = "anthropic/claude-opus-4-6"
	modelKimi = "kimi-for-coding/k2p5"
)

var presets = map[string]string{
	"opus":   modelOpus,
	"claude": modelOpus,
	"kimi":   modelKimi,
}

// base.jsonc의 "model" 필드 매칭 (줄 단위)
var baseModelRE = regexp.MustCompile(`(?m)(^\s*"model"\s*:\s*")[^"]+(",?\s*$)`)

// omo.jsonc의 sisyphus agent model 매칭 (블록 단위, 멀티라인)
var sisyphusModelRE = regexp.MustCompile(`(?s)("sisyphus"\s*:\s*\{.*?"model"\s*:\s*")[^"]+(".*?\})`)

func main() {
	root := resolveRoot()
	args := os.Args[1:]

	// 인자 없으면 toggle 모드
	if len(args) == 0 {
		toggleModel(root)
		return
	}

	if args[0] == "--list" || args[0] == "-l" {
		showStatus(root)
		return
	}

	preset := normalizePreset(args[0])
	if preset == "" {
		fmt.Fprintf(os.Stderr, "Unknown preset: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "Usage: sw [opus|kimi|--list]\n")
		fmt.Fprintf(os.Stderr, "\nPresets:\n")
		fmt.Fprintf(os.Stderr, "  opus, claude  -> %s\n", modelOpus)
		fmt.Fprintf(os.Stderr, "  kimi          -> %s\n", modelKimi)
		fmt.Fprintf(os.Stderr, "  (no args)     -> toggle between opus/kimi\n")
		os.Exit(1)
	}

	target := presets[preset]
	switchToModel(root, target)
}

// toggleModel: 현재 모델을 감지해서 반대로 전환
func toggleModel(root string) {
	// 현재 모델 확인
	baseData, err := os.ReadFile(filepath.Join(root, "config", "base.jsonc"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read base.jsonc: %v\n", err)
		os.Exit(1)
	}
	currentModel := extractModel(baseModelRE, string(baseData))

	// 반대 모델로 전환
	var target string
	if strings.Contains(currentModel, "kimi") {
		target = modelOpus
		fmt.Printf("Current: Kimi -> Switching to Opus\n\n")
	} else if strings.Contains(currentModel, "claude") || strings.Contains(currentModel, "opus") {
		target = modelKimi
		fmt.Printf("Current: Opus -> Switching to Kimi\n\n")
	} else {
		// 알 수 없는 모델이면 기본값으로 Kimi
		target = modelKimi
		fmt.Printf("Current: %s (unknown) -> Switching to Kimi\n\n", currentModel)
	}

	switchToModel(root, target)
}

// switchToModel: 지정된 모델로 전환
func switchToModel(root string, target string) {
	changed := false

	// 1. base.jsonc 업데이트
	baseChanged, err := updateBaseModel(filepath.Join(root, "config", "base.jsonc"), target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update base.jsonc: %v\n", err)
		os.Exit(1)
	}
	if baseChanged {
		changed = true
		fmt.Printf("✓ Updated config/base.jsonc: %s\n", target)
	}

	// 2. omo.jsonc 업데이트
	omoChanged, err := updateOmoModel(filepath.Join(root, "config", "omo.jsonc"), target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update omo.jsonc: %v\n", err)
		os.Exit(1)
	}
	if omoChanged {
		changed = true
		fmt.Printf("✓ Updated config/omo.jsonc (sisyphus agent): %s\n", target)
	}

	if !changed {
		fmt.Printf("Already using %s - no changes needed.\n", target)
		return
	}

	// 3. 설정 재생성
	fmt.Println("\nRegenerating opencode.jsonc...")
	cmd := exec.Command("go", "run", filepath.Join(root, "scripts", "gen-opencode-config.go"))
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen-opencode-config.go failed: %v\n", err)
		os.Exit(1)
	}

	// 4. OMO 설정 재생성
	fmt.Println("\nRegenerating oh-my-opencode.jsonc...")
	cmd2 := exec.Command("go", "run", filepath.Join(root, "scripts", "gen-omo-config.go"))
	cmd2.Dir = root
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	if err := cmd2.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen-omo-config.go failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Successfully switched to %s\n", target)
}

func resolveRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	// Verify we're in the right repo
	if _, err := os.Stat(filepath.Join(wd, "config", "omo.jsonc")); err != nil {
		fmt.Fprintf(os.Stderr, "config/omo.jsonc not found in %s\n", wd)
		fmt.Fprintf(os.Stderr, "Run from repository root.\n")
		os.Exit(1)
	}
	return wd
}

func normalizePreset(arg string) string {
	normalized := strings.ToLower(strings.TrimSpace(arg))
	if _, ok := presets[normalized]; ok {
		return normalized
	}
	return ""
}

func updateBaseModel(path string, target string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	content := string(data)
	updated := baseModelRE.ReplaceAllString(content, `${1}`+target+`${2}`)
	if updated == content {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(updated), 0o644)
}

func updateOmoModel(path string, target string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	content := string(data)
	updated := sisyphusModelRE.ReplaceAllString(content, `${1}`+target+`${2}`)
	if updated == content {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(updated), 0o644)
}

func showStatus(root string) {
	fmt.Println("=== OMO Model Status ===")
	fmt.Println()

	// base.jsonc 상태
	baseData, err := os.ReadFile(filepath.Join(root, "config", "base.jsonc"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read base.jsonc: %v\n", err)
		os.Exit(1)
	}
	baseModel := extractModel(baseModelRE, string(baseData))
	fmt.Printf("config/base.jsonc model:     %s\n", baseModel)

	// omo.jsonc 상태
	omoData, err := os.ReadFile(filepath.Join(root, "config", "omo.jsonc"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read omo.jsonc: %v\n", err)
		os.Exit(1)
	}
	sisyphusModel := extractModel(sisyphusModelRE, string(omoData))
	fmt.Printf("config/omo.jsonc sisyphus:   %s\n", sisyphusModel)

	fmt.Println("")
	fmt.Println("Usage: sw [opus|kimi|--list]")
	fmt.Println("       sw              # toggle between opus/kimi")
}

func extractModel(re *regexp.Regexp, content string) string {
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "?"
	}
	line := matches[0]
	start := strings.Index(line, `"model"`)
	if start < 0 {
		return "?"
	}
	idx := strings.Index(line[start:], `: "`)
	if idx < 0 {
		return "?"
	}
	rest := line[start+idx+3:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return "?"
	}
	return rest[:end]
}
