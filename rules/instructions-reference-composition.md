# Instructions Reference Composition

This document defines a standard way to compose instruction references for tasks.

## Composition order

Two composition modes are used in this repository:

1. Runtime baseline: `opencode.jsonc` loads Tier 1 rules via explicit file paths (not glob). `AGENTS.md` is loaded natively by OpenCode and must not be duplicated in the instructions array.
2. Task-time references: prompts/contracts cite only relevant files.

Task-time composition order:

1. Base execution rules (Tier 1 — always loaded):
   - `rules/hard-autonomy-no-questions.md`
   - `rules/session-init.md`
   - `rules/requirements-verification.md`
2. Domain rules (Tier 2 — only when relevant):
   - global ELK: `rules/elk-troubleshooting-global.md`
   - global error-tracking runbook: `rules/elk-error-tracking-runbook.md`
   - OpenCode ELK delta: `rules/elk-opencode-troubleshooting.md`
   - OpenCode runbook: `rules/opencode-elk-triage-runbook.md`
   - Proxmox ELK delta: `rules/elk-proxmox-troubleshooting.md`
   - Proxmox runbook: `rules/proxmox-elk-triage-runbook.md`
3. Repository governance (Tier 1 baseline + Tier 3 process):
   - `rules/deployment-automation.md` (Tier 1)
   - `rules/monorepo-standards.md` (Tier 1)
   - `rules/requirements-modularization-checklist.md` (Tier 3 — requirement/checklist tasks)
   - `rules/document-normalization-runbook.md` (Tier 3 — document move/rename tasks)

## Conflict handling template

When conflicts appear, resolve and document in this order:

1. `hard-autonomy-no-questions.md`
2. domain rule
3. `session-init.md`
4. repository standards

Write conflict decisions in one line:

`applied: <base rule>, overridden by: <exception rule>, scope: <bounded scope>`

## Reference hygiene

1. Use exact relative paths under `rules/`.
2. Avoid duplicate references to equivalent rules.
3. Keep the reference set minimal and task-specific.
4. If a referenced file is canonical SSoT, point secondary rules to it instead of copying text.
5. With `rules/*.md` baseline loading, never copy shared normative text into multiple files; use one canonical file and cross-reference it.
