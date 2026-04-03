// validate-requirements-verification.go validates rule and doc files against
// the requirements-verification policy structure contract, cross-reference
// integrity, and checklist quality rules.
//
// Usage:
//
//	go run scripts/validate-requirements-verification.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// structureSections are the 5 canonical sections from requirements-verification.md
// § Structure contract. Each non-trivial requirement doc must contain these
// (or explicitly state N/A).
var structureSections = []struct {
	Name         string
	Alternatives []string
}{
	{
		Name:         "scope",
		Alternatives: []string{"scope boundaries", "scope and applicability"},
	},
	{
		Name:         "inputs/constraints",
		Alternatives: []string{"constraint", "prerequisite", "assumption", "dependenc", "requirement", "inputs"},
	},
	{
		Name:         "decision/rules",
		Alternatives: []string{"rules", "core rule", "execution rule", "policy", "decision", "enforcement"},
	},
	{
		Name:         "verification",
		Alternatives: []string{"verification", "post-implementation", "completion check", "evidence"},
	},
	{
		Name:         "rollback/safety",
		Alternatives: []string{"safety", "rollback", "recovery", "revert", "risk", "anti-pattern", "failure", "blocked"},
	},
}

// skipStructureCheck lists files exempt from the structure contract check.
// Architecture docs, meta docs, and the policy itself are excluded.
var skipStructureCheck = map[string]bool{
	"README.md":                     true,
	"AGENTS.md":                     true,
	"requirements-verification.md":  true,
	"hard-autonomy-no-questions.md": true,
}

// minLinesForStructureCheck skips trivially small files.
const minLinesForStructureCheck = 30

// imperativeVerbs is a comprehensive list of imperative verbs acceptable
// as first words in numbered checklist items.
var imperativeVerbs = map[string]bool{}

func init() {
	verbs := []string{
		// Core instruction verbs from requirements-verification.md
		"run", "verify", "record", "update", "check", "create", "delete",
		"add", "remove", "set", "configure", "deploy", "install", "build",
		"test", "validate", "ensure", "apply", "search", "extract", "flag",
		"report", "use", "keep", "avoid", "do", "skip", "include", "mark",
		"fire", "launch", "send", "read", "write", "call", "return", "pass",
		"fail", "stop", "start", "continue", "follow", "load", "import",
		"export", "merge", "split", "move", "rename", "copy", "filter",
		"sort", "limit", "prefer", "match", "prevent", "block", "allow",
		"enable", "disable", "specify", "provide", "define", "implement",
		"handle", "process", "convert", "assign", "override", "navigate",
		"fetch", "execute", "persist", "document", "announce", "collect",
		"cancel", "revert", "isolate", "fix", "inspect", "make",
		// Extended imperative verbs found across rule files
		"get", "find", "select", "confirm", "query", "discover", "prioritize",
		"normalize", "establish", "identify", "group", "consult", "submit",
		"classify", "resolve", "propose", "correlate", "treat", "chain",
		"notify", "trigger", "tighten", "choose", "switch", "retry",
		"translate", "complete", "commit", "preserve", "inventory", "list",
		"infer", "scope", "detect", "scan", "lint", "generate", "regenerate",
		"accept", "reject", "abort", "emit", "log", "print", "warn",
		"parse", "format", "trim", "strip", "replace", "insert", "append",
		"prepend", "push", "pop", "wait", "signal", "subscribe", "publish",
		"open", "close", "connect", "disconnect", "bind", "unbind",
		"register", "unregister", "initialize", "finalize", "setup",
		"cleanup", "reset", "clear", "flush", "drain",
		"monitor", "observe", "watch", "track", "trace", "profile",
		"measure", "compare", "diff", "patch", "restore",
		"backup", "snapshot", "clone", "fork", "branch", "tag", "release",
		"promote", "delegate", "forward", "redirect",
		"replay", "reproduce", "simulate", "mock",
		"intercept", "proxy", "cache", "invalidate", "refresh",
		"rotate", "iterate", "traverse", "walk", "visit",
		"map", "reduce", "aggregate", "summarize", "annotate", "label",
		"categorize", "partition", "segment",
		"encode", "decode", "encrypt", "decrypt", "sign", "hash",
		"compress", "decompress", "serialize", "deserialize",
		"migrate", "upgrade", "downgrade", "pin", "unpin", "lock", "unlock",
		"grant", "revoke", "authorize", "authenticate",
		"throttle", "debounce",
		"assert", "expect", "require", "demand", "enforce", "guarantee",
		"adopt", "reference", "wrap", "unwrap", "embed", "inline",
		"stage", "unstage", "stash", "checkout", "rebase", "squash",
		"pull", "download", "upload", "transfer", "sync", "reconcile",
		"drop", "truncate", "prune", "purge", "archive", "unarchive",
		"raise", "lower", "increase", "decrease", "scale", "resize",
		"attach", "detach", "mount", "unmount", "link", "unlink",
		"target", "focus", "narrow", "broaden", "expand", "extend",
		"restrict", "constrain", "relax", "loosen", "widen",
		"name", "declare", "expose", "hide", "show",
		"activate", "deactivate", "toggle", "swap", "exchange",
		"cross-check", "cross-reference", "re-check", "re-verify",
		// Conditional/quantifier prefixes common in procedural checklists
		"if", "when", "each", "only", "never", "always", "for",
		"after", "before", "once", "until", "while", "unless",
		// Additional domain verbs
	}
	for _, v := range verbs {
		imperativeVerbs[v] = true
	}
}

// checklistSectionKeywords lists substrings (lowercase) that indicate a heading
// is a step-by-step action checklist. Only numbered items under these headings
// are checked for imperative wording. Intentionally narrow to avoid false
// positives on enumerated policy rules or "quality rules" meta-sections.
var checklistSectionKeywords = []string{
	"pre-implementation checklist",
	"post-implementation verification",
	"during implementation",
	"shell-to-go migration checklist",
	"step ", "steps",
	"workflow", "cycle", "procedure", "sequence",
	"bootstrap", "initialization",
	"pre-check", "post-check",
	"task-driven development", "rag workflow",
	"verification output",
}

// crossRefRE matches file path references in markdown (backtick-wrapped).
var crossRefRE = regexp.MustCompile(
	"`(?:docs|rules|scripts|config)/[a-zA-Z0-9._/-]+`",
)

// numberedItemRE matches numbered list items: "1. ...", "2. ...", etc.
var numberedItemRE = regexp.MustCompile(`^\s*\d+\.\s+(.+)`)

var headingRE = regexp.MustCompile(`(?m)^#{1,4}\s+(.+)`)

type violation struct {
	File     string
	Code     string
	Severity string // "error" or "warning"
	Message  string
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var violations []violation

	// Collect all rule and doc files.
	var targets []string
	for _, dir := range []string{"rules", "docs"} {
		dirPath := filepath.Join(root, dir)
		entries, readErr := os.ReadDir(dirPath)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", dir, readErr)
			os.Exit(1)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			targets = append(targets, filepath.Join(dir, e.Name()))
		}
	}

	for _, rel := range targets {
		full := filepath.Join(root, rel)
		data, readErr := os.ReadFile(full)
		if readErr != nil {
			violations = append(violations, violation{
				File:    rel,
				Code:    "READ_ERROR",
				Message: readErr.Error(),
			})
			continue
		}

		content := string(data)
		lines := strings.Split(content, "\n")

		// Check 1: Cross-reference integrity.
		violations = append(violations, checkCrossRefs(root, rel, content)...)

		// Check 2: Structure contract (non-trivial files only).
		base := filepath.Base(rel)
		if !skipStructureCheck[base] && len(lines) >= minLinesForStructureCheck {
			violations = append(violations, checkStructureContract(rel, content)...)
		}

		// Check 3: Checklist quality — imperative wording.
		violations = append(violations, checkImperativeWording(rel, lines)...)
	}

	var errors, warnings []violation
	for _, v := range violations {
		if v.Severity == "warning" {
			warnings = append(warnings, v)
		} else {
			errors = append(errors, v)
		}
	}

	if len(warnings) > 0 {
		fmt.Println("--- Warnings (advisory, do not fail CI) ---")
		for _, v := range warnings {
			fmt.Printf("  WARN  %-18s %-50s %s\n", v.Code, v.File, v.Message)
		}
		fmt.Printf("  %d warning(s)\n\n", len(warnings))
	}

	if len(errors) > 0 {
		fmt.Println("--- Errors ---")
		for _, v := range errors {
			fmt.Printf("  ERR   %-18s %-50s %s\n", v.Code, v.File, v.Message)
		}
		fmt.Printf("\nFound %d requirement verification error(s).\n", len(errors))
		os.Exit(1)
	}

	fmt.Printf("Requirements verification check passed (%d files scanned, %d warnings).\n", len(targets), len(warnings))
}

// checkCrossRefs validates that file path references in a markdown file
// point to files that actually exist.
func checkCrossRefs(root, rel, content string) []violation {
	var out []violation
	matches := crossRefRE.FindAllString(content, -1)
	seen := make(map[string]bool)

	for _, m := range matches {
		// Strip backticks.
		ref := strings.Trim(m, "`")

		// Deduplicate within same file.
		if seen[ref] {
			continue
		}
		seen[ref] = true

		// Skip glob patterns and partial refs (contain *).
		if strings.Contains(ref, "*") {
			continue
		}

		full := filepath.Join(root, ref)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			out = append(out, violation{
				File:     rel,
				Code:     "BROKEN_REF",
				Severity: "error",
				Message:  fmt.Sprintf("referenced %q does not exist", ref),
			})
		}
	}
	return out
}

// checkStructureContract checks that a requirement doc includes the 5
// canonical sections (or N/A markers).
func checkStructureContract(rel, content string) []violation {
	var out []violation
	lower := strings.ToLower(content)

	// Extract all heading texts.
	headings := headingRE.FindAllStringSubmatch(content, -1)
	var headingTexts []string
	for _, h := range headings {
		if len(h) > 1 {
			headingTexts = append(headingTexts, strings.ToLower(strings.TrimSpace(h[1])))
		}
	}

	for _, section := range structureSections {
		sectionLower := strings.ToLower(section.Name)
		found := false

		// Check headings for main name or alternatives.
		candidates := append([]string{sectionLower}, section.Alternatives...)
		for _, candidate := range candidates {
			for _, ht := range headingTexts {
				if strings.Contains(ht, candidate) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		// Check N/A markers in body text.
		if !found {
			naPatterns := []string{
				sectionLower + ": n/a",
				sectionLower + " — n/a",
				sectionLower + ": `n/a`",
				sectionLower + " - n/a",
			}
			for _, p := range naPatterns {
				if strings.Contains(lower, p) {
					found = true
					break
				}
			}
		}

		if !found {
			out = append(out, violation{
				File:     rel,
				Code:     "MISSING_SECTION",
				Severity: "warning",
				Message:  fmt.Sprintf("missing structure contract section: %q (add heading or N/A)", section.Name),
			})
		}
	}
	return out
}

// checkImperativeWording validates that numbered list items start with an
// imperative verb. Skips items under sections marked as non-checklist
// (reference, examples, etc.).
func checkImperativeWording(rel string, lines []string) []violation {
	var out []violation

	inCodeBlock := false
	currentSection := ""

	for i, line := range lines {
		// Track code blocks.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Track current section heading.
		hm := headingRE.FindStringSubmatch(line)
		if hm != nil && len(hm) > 1 {
			currentSection = strings.ToLower(strings.TrimSpace(hm[1]))
			continue
		}

		// Skip items under non-checklist sections.
		// Allowlist: only check items under sections that look like checklists.
		isChecklistSection := false
		for _, kw := range checklistSectionKeywords {
			if strings.Contains(currentSection, kw) {
				isChecklistSection = true
				break
			}
		}
		if !isChecklistSection {
			continue
		}

		m := numberedItemRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		itemText := strings.TrimSpace(m[1])
		if itemText == "" {
			continue
		}

		// Skip items that are table headers, code references, links, or emphasis.
		if strings.HasPrefix(itemText, "|") ||
			strings.HasPrefix(itemText, "`") ||
			strings.HasPrefix(itemText, "[") ||
			strings.HasPrefix(itemText, "*") ||
			strings.HasPrefix(itemText, "~") {
			continue
		}

		// Extract first word.
		firstWord := strings.ToLower(strings.SplitN(itemText, " ", 2)[0])
		// Strip trailing punctuation.
		firstWord = strings.TrimRight(firstWord, ":.,;!?()")
		// Strip leading markdown bold markers.
		firstWord = strings.TrimLeft(firstWord, "*")

		if !imperativeVerbs[firstWord] {
			out = append(out, violation{
				File:     rel,
				Code:     "NON_IMPERATIVE",
				Severity: "error",
				Message:  fmt.Sprintf("line %d: numbered item starts with %q (expected imperative verb)", i+1, firstWord),
			})
		}
	}
	return out
}
