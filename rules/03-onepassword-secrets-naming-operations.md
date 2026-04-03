# 1Password Secrets Naming — Operations

Migration, verification, and safety procedures for 1Password secrets naming.
Part of: `01-onepassword-secrets-naming.md` (split for modularity)

## Known inconsistencies and migration

| Item | Issue | Fix |
|------|-------|-----|
| `elk` | Section `secrets` (lowercase) exists alongside `Passwords` | Rename `secrets` → merge fields into `Passwords`, delete `secrets` section |
| `elk` | Duplicate `elastic_password` in both `secrets` and `Passwords` | Keep `Passwords/elastic_password`, remove duplicate from `secrets` |
| `safetywallet` | Uses section UUID references (`Section_x4jg...`) in `op://` URIs | Create named sections, migrate references to use labels |

Migration command pattern:

```bash
# Rename a section (requires item edit via JSON)
op item get elk --format json | jq '.sections[] | select(.label=="secrets") | .label = "Passwords"' 
# Then re-import or manually edit via 1P UI

# Move a field between sections
op item edit elk "Passwords.elastic_password=<value>"
```

## Verification

1. Verify all section names are Title Case: no lowercase section names in the vault.
2. Verify all field names are snake_case: no camelCase or UPPER_CASE field names.
3. Verify field type selection: all passwords/tokens/keys use `CONCEALED`, all IDs use `STRING`, all URLs use `URL`.
4. Verify every item has the `homelab` tag plus at least one domain tag.
5. Verify `op://` URIs in `.env.tpl` files match the naming conventions in this spec.
6. Verify no duplicate fields exist across sections within the same item.
7. Audit command: `op item list --vault homelab --format json | jq '.[].title'` lists all items.

## Rollback/safety

1. Section and field renames in 1Password are non-destructive — old values are preserved in item history.
2. `op://` URI changes require updating all referencing files (`.env.tpl`, Go constants, CI configs).
3. Use `op item get <item> --format json` to export item state before bulk edits.
4. 1Password item history provides a 365-day audit trail for any field changes.

## Reference composition

1. Tier 1 on-demand rule — not loaded in every session.
2. Complements `01-onepassword-integration.md` which covers integration patterns (how to use secrets).
3. This spec covers schema and naming (how to structure items).
4. Defers to `00-hard-autonomy-no-questions.md` on execution posture.
