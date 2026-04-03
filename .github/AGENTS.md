# GITHUB AUTOMATION KNOWLEDGE BASE

**Scope:** `.github/`
**Parent:** `AGENTS.md`

## OVERVIEW

This subtree owns repository automation: validation CI, Codex workflows, auto-merge/cleanup jobs, and issue/PR lifecycle templates.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Main validation pipeline | `.github/workflows/ci.yml` | commitlint + config/naming/phrasing/ref/requirements gates + `npm test` |
| Commit message workflow | `.github/workflows/commitlint.yml` | PR-targeted lint behavior |
| Drift / maintenance automation | `.github/workflows/config-drift-report.yml` | config health reporting |
| Codex automations | `.github/workflows/codex-*.yml` | issue, review, normalization, triage flows |
| Release metadata | `.github/workflows/release-drafter.yml` | release note automation |
| PR sizing / labels / stale | `.github/workflows/pr-size.yml` | policy-only repo hygiene |

## CONVENTIONS

- Pin third-party actions by full commit SHA.
- Keep CI aligned with local `npm run prepush:check` gates.
- Treat Codex workflows as policy automation, not generic CI.

## ANTI-PATTERNS (THIS SUBTREE)

- Do not introduce floating action versions.
- Do not add workflow-only checks that diverge from local validation without clear reason.
- Do not hide repo policy in workflow YAML when the canonical rule belongs in `rules/` or `scripts/`.
