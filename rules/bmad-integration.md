# BMAD-METHOD Integration

Apply this policy when a project contains BMAD artifacts (`_bmad-output/` directory).
Enables Sisyphus to autonomously consume BMAD planning outputs and implement stories.

Priority: Tier 1 baseline. Loaded globally via `opencode.jsonc` instructions.

## Scope

**In scope:** Projects with `_bmad-output/` directory containing BMAD planning artifacts.
**Out of scope:** BMAD authoring (Phase 1–3). Sisyphus consumes artifacts, does not create them.

## Detection

At session start (after session-init bootstrap), check for BMAD artifacts:

1. Glob for `_bmad-output/` in the project root.
2. If absent, skip all BMAD rules silently. No further action.
3. If present, load project context and enter BMAD implementation mode.

## Artifact inventory

When `_bmad-output/` is detected, read these files in order:

| Priority | File                                                       | Purpose                                       | Required  |
| -------- | ---------------------------------------------------------- | --------------------------------------------- | --------- |
| 1        | `_bmad-output/project-context.md`                          | Implementation rules, conventions, tech stack | Yes       |
| 2        | `_bmad-output/planning-artifacts/architecture.md`          | Technical decisions, ADRs, system design      | Yes       |
| 3        | `_bmad-output/planning-artifacts/PRD.md`                   | Requirements (FRs/NFRs), acceptance criteria  | Yes       |
| 4        | `_bmad-output/implementation-artifacts/sprint-status.yaml` | Current sprint state, story queue             | Yes       |
| 5        | `_bmad-output/planning-artifacts/ux-spec.md`               | UX design spec                                | If exists |
| 6        | `_bmad-output/planning-artifacts/epics/*.md`               | Epic definitions with story breakdowns        | If exists |

## Story selection

Parse `sprint-status.yaml` to find the next story to implement:

1. Read `sprint-status.yaml` and parse the story list.
2. Select the first story with status `ready` or `todo` (in sprint order).
3. If no ready stories remain, report sprint complete and stop.
4. Read the story file: `_bmad-output/story-{slug}.md`.
5. Cross-reference story acceptance criteria against PRD requirements.

### Story status values

| Status        | Meaning                              | Action                                |
| ------------- | ------------------------------------ | ------------------------------------- |
| `todo`        | Not started, not yet refined         | Skip unless no `ready` stories exist  |
| `ready`       | Refined, acceptance criteria defined | Implement (preferred)                 |
| `in-progress` | Started but incomplete               | Resume — check for partial work first |
| `review`      | Implementation done, needs QA        | Run verification only                 |
| `done`        | Completed and verified               | Skip                                  |
| `blocked`     | Cannot proceed                       | Report blocker and skip to next       |

## Implementation cycle

For each story, follow this sequence:

1. **Load context**: Read architecture.md + PRD.md + project-context.md + story file.
2. **Plan**: Create todo list from story acceptance criteria and technical tasks.
3. **Implement**: Write code following architecture decisions and project conventions.
4. **Test**: Run tests specified in story acceptance criteria.
5. **Verify**: Run diagnostics, build, and lint on all changed files.
6. **Update sprint**: Update `sprint-status.yaml` — set story status to `review`.
7. **Report**: Summarize what was implemented, files changed, and verification results.

## Sprint status update format

When updating `sprint-status.yaml`, preserve the existing YAML structure. Only modify:

- Story `status` field (e.g., `todo` → `in-progress` → `review`).
- Story `notes` field (append implementation notes).
- Do not modify story definitions, acceptance criteria, or sprint metadata.

## Architecture compliance

All implementation must respect BMAD architecture decisions:

1. Read `architecture.md` ADRs before making technical choices.
2. If a story requires a decision not covered by existing ADRs, document the gap and proceed with the safest choice.
3. Never contradict an existing ADR. If an ADR seems wrong, report it as a blocker.
4. Follow the tech stack and patterns defined in `project-context.md`.

## Interaction with other rules

1. BMAD stories feed into the normal implementation workflow — all existing rules still apply.
2. Archon tasks can be created from BMAD stories for tracking (optional, not required).
3. Code modularization thresholds apply to all BMAD-generated code.
4. Deployment automation rules apply to any CI/CD changes from BMAD stories.

## Multi-story sessions

When implementing multiple stories in one session:

1. Complete one story fully before starting the next.
2. Commit after each story (if user has requested commits).
3. Update `sprint-status.yaml` after each story completion.
4. Stop after 3 stories per session to avoid context degradation.

## Verification checklist

| #   | Check                           | Evidence                                |
| --- | ------------------------------- | --------------------------------------- |
| 1   | Story acceptance criteria met   | Each criterion mapped to implementation |
| 2   | Architecture decisions followed | ADR cross-reference                     |
| 3   | Tests pass                      | Test runner output                      |
| 4   | Build succeeds                  | Exit code 0                             |
| 5   | Diagnostics clean               | `lsp_diagnostics` on changed files      |
| 6   | Sprint status updated           | `sprint-status.yaml` diff               |
| 7   | Project conventions followed    | `project-context.md` compliance         |

## Reference composition

1. Loaded as Tier 1 baseline rule via `opencode.jsonc`.
2. Defers to `hard-autonomy-no-questions.md` on execution posture.
3. Defers to `requirements-verification.md` for acceptance criteria verification.
4. Defers to `code-modularization.md` for file size governance.
5. BMAD artifacts are authored externally (Claude Code / Cursor); this rule is read-only consumer.
