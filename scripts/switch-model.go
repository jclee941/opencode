package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const omoPath = "/home/jclee/.config/opencode/config/omo.jsonc"

var opusAgents = []string{"sisyphus", "prometheus", "metis", "atlas"}

func main() {
	if len(os.Args) < 2 {
		// Toggle mode - detect current state and switch
		content, err := os.ReadFile(omoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading omo.jsonc: %v\n", err)
			os.Exit(1)
		}
		contentStr := string(content)

		// Check if currently using kimi by looking at sisyphus model specifically
		isKimi := detectKimiMode(contentStr)

		if isKimi {
			switchToClaude(&contentStr)
			fmt.Println("Switched to Claude mode")
		} else {
			switchToKimi(&contentStr)
			fmt.Println("Switched to Kimi mode")
		}

		if err := os.WriteFile(omoPath, []byte(contentStr), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing omo.jsonc: %v\n", err)
			os.Exit(1)
		}

		regenerateConfigs()
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "kimi":
		content, err := os.ReadFile(omoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading omo.jsonc: %v\n", err)
			os.Exit(1)
		}
		contentStr := string(content)
		switchToKimi(&contentStr)
		if err := os.WriteFile(omoPath, []byte(contentStr), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing omo.jsonc: %v\n", err)
			os.Exit(1)
		}
		regenerateConfigs()
		fmt.Println("Switched to Kimi mode")

	case "claude":
		content, err := os.ReadFile(omoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading omo.jsonc: %v\n", err)
			os.Exit(1)
		}
		contentStr := string(content)
		switchToClaude(&contentStr)
		if err := os.WriteFile(omoPath, []byte(contentStr), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing omo.jsonc: %v\n", err)
			os.Exit(1)
		}
		regenerateConfigs()
		fmt.Println("Switched to Claude mode")

	case "--list", "-l", "list":
		content, err := os.ReadFile(omoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading omo.jsonc: %v\n", err)
			os.Exit(1)
		}
		contentStr := string(content)
		showCurrentStatus(contentStr)

	default:
		fmt.Fprintf(os.Stderr, "Usage: sw [kimi|claude|--list]\n")
		fmt.Fprintf(os.Stderr, "       sw (toggle between modes)\n")
		os.Exit(1)
		fmt.Fprintf(os.Stderr, "       sw (toggle between modes)\n")
		os.Exit(1)
	}
}

// detectKimiMode checks if sisyphus is currently using kimi model
func detectKimiMode(content string) bool {
	// Find sisyphus agent block and check its model
	re := regexp.MustCompile(`"sisyphus"\s*:\s*\{[^}]*"model"\s*:\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		model := matches[1]
		return strings.Contains(model, "kimi")
	}

	// Fallback: check if $CLAUDE_OPUS is set to kimi
	re = regexp.MustCompile(`"\$CLAUDE_OPUS"\s*:\s*"([^"]+)"`)
	matches = re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.Contains(matches[1], "kimi")
	}

	return false
}

func switchToKimi(content *string) {
	// Update model variables using simple string replacement
	*content = strings.ReplaceAll(*content, `"$CLAUDE_OPUS": "anthropic/claude-opus-4-6"`, `"$CLAUDE_OPUS": "kimi-for-coding/k2p5"`)
	*content = strings.ReplaceAll(*content, `"$CLAUDE_OPUS": "kimi-for-coding/k2p5"`, `"$CLAUDE_OPUS": "kimi-for-coding/k2p5"`) // idempotent
	*content = strings.ReplaceAll(*content, `"$CLAUDE_SONNET": "anthropic/claude-sonnet-4-6"`, `"$CLAUDE_SONNET": "kimi-for-coding/k2p5"`)
	*content = strings.ReplaceAll(*content, `"$CLAUDE_SONNET": "kimi-for-coding/k2p5"`, `"$CLAUDE_SONNET": "kimi-for-coding/k2p5"`) // idempotent

	// Update descriptions and models for Opus agents
	for _, agent := range opusAgents {
		// Update description
		reDesc := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*\{[^}]*"description"\s*:\s*"[^"]*"`, agent))
		*content = reDesc.ReplaceAllStringFunc(*content, func(match string) string {
			return regexp.MustCompile(`"description"\s*:\s*"[^"]*"`).ReplaceAllString(match, `"description": "Kimi K2.5 (rate limit bypass)"`)
		})

		// Update model
		reModel := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*\{[^}]*"model"\s*:\s*"[^"]+"`, agent))
		*content = reModel.ReplaceAllStringFunc(*content, func(match string) string {
			return regexp.MustCompile(`"model"\s*:\s*"[^"]+"`).ReplaceAllString(match, `"model": "kimi-for-coding/k2p5"`)
		})

		// Also update fallback models
		reFallback := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*\{[^}]*"fallback_models"\s*:\s*\[[^\]]*\]`, agent))
		*content = reFallback.ReplaceAllStringFunc(*content, func(match string) string {
			return regexp.MustCompile(`anthropic/claude-[^"\s]+`).ReplaceAllString(match, "kimi-for-coding/k2p5")
		})
	}
}

func switchToClaude(content *string) {
	// Update model variables using simple string replacement
	*content = strings.ReplaceAll(*content, `"$CLAUDE_OPUS": "kimi-for-coding/k2p5"`, `"$CLAUDE_OPUS": "anthropic/claude-opus-4-6"`)
	*content = strings.ReplaceAll(*content, `"$CLAUDE_OPUS": "anthropic/claude-opus-4-6"`, `"$CLAUDE_OPUS": "anthropic/claude-opus-4-6"`) // idempotent
	*content = strings.ReplaceAll(*content, `"$CLAUDE_SONNET": "kimi-for-coding/k2p5"`, `"$CLAUDE_SONNET": "anthropic/claude-sonnet-4-6"`)
	*content = strings.ReplaceAll(*content, `"$CLAUDE_SONNET": "anthropic/claude-sonnet-4-6"`, `"$CLAUDE_SONNET": "anthropic/claude-sonnet-4-6"`) // idempotent

	// Update descriptions and models for Opus agents
	for _, agent := range opusAgents {
		// Update description
		reDesc := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*\{[^}]*"description"\s*:\s*"[^"]*"`, agent))
		*content = reDesc.ReplaceAllStringFunc(*content, func(match string) string {
			return regexp.MustCompile(`"description"\s*:\s*"[^"]*"`).ReplaceAllString(match, `"description": "Claude Opus (communicator)"`)
		})

		// Update model
		reModel := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*\{[^}]*"model"\s*:\s*"[^"]+"`, agent))
		*content = reModel.ReplaceAllStringFunc(*content, func(match string) string {
			return regexp.MustCompile(`"model"\s*:\s*"[^"]+"`).ReplaceAllString(match, `"model": "anthropic/claude-opus-4-6"`)
		})

		// Also update fallback models
		reFallback := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*\{[^}]*"fallback_models"\s*:\s*\[[^\]]*\]`, agent))
		*content = reFallback.ReplaceAllStringFunc(*content, func(match string) string {
			return regexp.MustCompile(`kimi-for-coding/k2p5`).ReplaceAllString(match, "anthropic/claude-opus-4-6")
		})
	}
}

func showCurrentStatus(content string) {
	fmt.Println("Current agent configurations:")
	fmt.Println()

	// Show $CLAUDE_OPUS value
	re := regexp.MustCompile(`"\$CLAUDE_OPUS"\s*:\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		fmt.Printf("$CLAUDE_OPUS: %s\n", matches[1])
	}

	re = regexp.MustCompile(`"\$CLAUDE_SONNET"\s*:\s*"([^"]+)"`)
	matches = re.FindStringSubmatch(content)
	if len(matches) > 1 {
		fmt.Printf("$CLAUDE_SONNET: %s\n", matches[1])
	}
	fmt.Println()

	// Show agent models
	for _, agent := range opusAgents {
		re := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*\{[^}]*"model"\s*:\s*"([^"]+)"`, agent))
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			model := matches[1]
			reDesc := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*\{[^}]*"description"\s*:\s*"([^"]*)"`, agent))
			descMatches := reDesc.FindStringSubmatch(content)
			desc := ""
			if len(descMatches) > 1 {
				desc = descMatches[1]
			}
			fmt.Printf("%s: %s (%s)\n", agent, model, desc)
		}
	}
}

func regenerateConfigs() {
	// Change to opencode directory
	os.Chdir("/home/jclee/.config/opencode")

	// Run config generators
	if _, err := os.Stat("scripts/gen-omo-config.go"); err == nil {
		exec.Command("go", "run", "scripts/gen-omo-config.go").Run()
	}
	if _, err := os.Stat("scripts/gen-opencode-config.go"); err == nil {
		exec.Command("go", "run", "scripts/gen-opencode-config.go").Run()
	}
}
