# AI-Native OpenAPI Specification Guidelines

**Date:** 2026-02-17
**Status:** Approved

## Problem

AI agents (Claude, ChatGPT) struggle with the VoIPbin OpenAPI spec due to three issues:

1. **Black-box polymorphism** — `FlowManagerAction.option` uses `additionalProperties: true`, so AI guesses fields instead of following strict schemas.
2. **Ambiguous primitive types** — `type: string` without `format:` for UUIDs, timestamps, and phone numbers causes validation errors.
3. **No examples** — Without `example:` values, AI hallucinates literal strings like `"string"` or `"user_id"`.

## Approach

Add mandatory guidelines to `bin-openapi-manager/CLAUDE.md` that every Claude session must follow when creating or modifying OpenAPI schemas. Rules apply incrementally (new/modified fields only), not retroactively.

Add a one-line pointer in root `CLAUDE.md` to ensure visibility from other service directories.

## Changes

### 1. Root CLAUDE.md

Add one line to the existing "Model/Struct Changes Require OpenAPI Updates" section (after step 4):

> IMPORTANT: Before modifying any OpenAPI spec, you MUST read and follow the AI-Native Specification Rules located in `bin-openapi-manager/CLAUDE.md`.

### 2. bin-openapi-manager/CLAUDE.md — New Section

Add "AI-Native OpenAPI Specification Rules" section with 5 rules:

**Rule 1: Use `oneOf` for Polymorphism**
- Ban `additionalProperties: true` for type-dependent fields
- Use `oneOf` to list allowed schemas (no `discriminator` — it requires the discriminating property inside child schemas, which doesn't fit our structure)
- Caution: `oneOf` changes generated Go types from `map[string]interface{}` to struct wrappers. Must run `go generate ./...` in both `bin-openapi-manager` and `bin-api-manager`

**Rule 2: Strict Structured Strings vs Safe Free Text**
- Type A (structured): UUIDs → `format: uuid`, timestamps → `format: date-time`, phone numbers → E.164 regex, enums → `enum` keyword
- Type B (free text): `name`, `description`, `detail` → exempt from format, but must have `example:` and `maxLength:` if DB has constraints

**Rule 3: Provenance in Descriptions**
- Every ID field referencing another resource must state which endpoint returns it
- Pattern: "The [ID Name] returned from the `[Endpoint Path]` response."

**Rule 4: Mandatory Realistic Examples**
- Every new or modified leaf property must have `example:` with real-looking data
- Never use: `"string"`, `"text"`, `null`, `"user_id"`
- Always use: `"+821012345678"`, `"active"`, `"550e8400-e29b-41d4-a716-446655440000"`

**Rule 5: Explicit Array Constraints**
- Arrays that logically need at least one item must use `minItems: 1`

## Trade-offs

- **Incremental vs retroactive**: Chose incremental to avoid a massive single PR touching all ~4000 lines. Existing schemas get fixed when touched.
- **No discriminator**: Dropped `discriminator` because it requires the discriminating property inside child schemas, which doesn't fit `FlowManagerAction` structure. `oneOf` alone provides sufficient AI guidance.
- **No automated linter**: Rules 3 and 4 are documentation quality rules that rely on human/AI judgment. Acceptable for guidelines.

## Files Changed

1. `CLAUDE.md` (root) — one line added
2. `bin-openapi-manager/CLAUDE.md` — new section added
