# Document Normalization Runbook

Use this runbook to reorganize existing documents to match repository structure and naming rules.

## Scope

Apply when moving, renaming, deduplicating, or replacing markdown documents.

## Execution checklist

1. Inventory current documents and classify each as:
   - runtime instruction (`rules/`)
   - standards/reference (`docs/`)
   - skill contract (`skills/**/AGENTS.md`, `skills/**/SKILL.md`)
   - root contract (`README.md`, `AGENTS.md`, `CHANGELOG.md`, `LICENSE`)
2. Choose canonical targets:
   - keep one canonical file per policy/topic
   - mark duplicates for deletion or redirect note
3. Apply structure moves:
   - move standards/guides/runbooks to `docs/`
   - keep operational instructions in `rules/`
4. Apply naming normalization:
   - directories: `kebab-case`
   - regular files: lowercase with dots/hyphens
   - contract files: uppercase only when standard
5. Update references in the same change:
   - markdown links
   - config/script references
   - rule cross-references
6. Verify:
   - run `npm run lint:naming`
   - spot-check moved/renamed links

## Safety rules

1. Preserve document meaning during rename/move.
2. Avoid policy drift by removing stale duplicate documents.
3. Do not place runtime artifacts in source domains.

## Output contract for normalization work

1. List files moved/renamed/deleted.
2. List references updated.
3. Report naming lint result.
