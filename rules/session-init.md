# Session Initialization

Use this checklist at the start of every coding task.

1. Infer the request and expected output from available context, then execute.
2. Inspect relevant files before editing.
3. Follow existing project patterns and naming conventions.
4. Make the smallest safe change that solves the task.
5. Run targeted verification after changes (diagnostics/tests/build as applicable).
6. Report what changed, where it changed, and verification results.

General guardrails:

- Do not expose secrets or credentials in output.
- For destructive or irreversible operations and question policy, follow `rules/hard-autonomy-no-questions.md`.
- For implementation tasks, always check and verify against requirements per `rules/requirements-verification.md`.
- Keep edits focused on the requested scope.
- If ambiguity exists, choose the safest reversible default and proceed.
- If a reversible improvement is identified within scope, apply it immediately instead of suggesting it as an optional next step.
