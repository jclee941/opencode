# MCP Tool Call Schema Hygiene

Apply this policy to every MCP tool invocation. This rule prevents JSON-RPC -32602
(Invalid params) errors caused by schema contract violations between the caller and
the MCP server.

Priority: Tier 1 baseline. Loaded globally via `opencode.jsonc` instructions.

## Root cause

-32602 errors are input contract errors, not server bugs. The caller sends parameters
that violate the tool's published schema. Common causes:

- Injecting synthetic keys not in the schema (e.g. `_placeholder` leaking into ES query body).
- Renaming parameter keys (e.g. `instruction` vs `instructions`, singular vs plural).
- Sending wrong types (array where object expected, string where number expected).
- Sending arguments to no-arg tools, or omitting required arguments.

## Rules

### 1. Pre-flight validation (fail-fast)

Before sending any MCP tool call:

1. Resolve the tool's published schema from the server's `tools/list` response.
2. Validate every parameter against the schema: name, type, required, enum constraints.
3. If validation fails, do not send the request. Fix the payload first.
4. Never guess parameter names from training data. Use only the names in the schema.

### 2. Strict schema compliance

1. Use exact key names from the published schema. No aliases, no camelCase/snake_case conversion.
2. Match exact types: `string` stays string, `integer` stays integer, `object` stays object.
3. For `enum` parameters, use only the listed values verbatim.
4. For `object`-typed parameters with defined `properties`, send only the declared properties.
   Do not inject extra keys (scaffolding, placeholders, metadata) into the object.
5. For nested objects, validate recursively. A valid top-level call with invalid nested
   structure still triggers -32602.

### 3. No-arg tool handling

1. Tools with no parameters: omit the `arguments` field entirely, or send `{}`.
2. Never send `[]`, `null`, or `{ "_placeholder": true }` as arguments to no-arg tools.
3. If the tool schema declares a placeholder parameter (e.g. `_placeholder: boolean REQUIRED`),
   send it but ensure it does not propagate into downstream API calls.

### 4. Name mapping prohibition

1. Use the server-published parameter key exactly as declared.
2. Do not auto-correct, pluralize, singularize, or alias parameter names.
3. If a tool expects `query`, do not send `q`. If it expects `file`, do not send `filePath`.
4. If uncertain about the correct key name, re-read the tool schema. Do not guess.

### 5. Error classification and routing

When a -32602 error occurs:

1. Classify it as **caller input error**, not server error.
2. Do not retry with the same payload. Fix the schema violation first.
3. Re-read the tool schema, identify the mismatch, correct, then retry.
4. If the same -32602 pattern recurs across sessions, escalate as a systemic schema
   mismatch requiring tool definition update.

### 6. Known pitfalls

| Pattern           | Wrong                                                                | Correct                                                                   |
| ----------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| Placeholder leak  | `{"_placeholder": true, "query": {...}}` forwarded to downstream API | Strip `_placeholder` before forwarding, or structure queryBody without it |
| Key aliasing      | `instructions` when schema says `instruction`                        | Use `instruction` exactly                                                 |
| Type coercion     | `"size": "5"` when schema says `integer`                             | `"size": 5`                                                               |
| Empty args        | `arguments: []` for no-arg tool                                      | Omit `arguments` or send `{}`                                             |
| Nested extra keys | Adding `_meta` inside a structured object param                      | Send only declared properties                                             |

## Verification

After fixing a -32602 error:

1. Confirm the corrected call succeeds (HTTP 200 / valid JSON-RPC response).
2. If the tool definition itself is ambiguous or contradicts server behavior,
   document the discrepancy and flag for tool maintainer review.

## Scope

This rule applies to all MCP tool calls regardless of server (mcphub, standalone, etc.).
It does not apply to direct HTTP/REST calls or non-MCP integrations.
