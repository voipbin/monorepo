# AI-Native RST Writing Guidelines Design

**Date:** 2026-02-17
**Status:** Draft
**Scope:** bin-api-manager/CLAUDE.md — new section for RST documentation standards

## Problem Statement

The existing RST documentation in `bin-api-manager/docsdev/source/` (~80+ files) is written for human readers. AI agents (LLMs) that consume these docs to make API calls frequently fail because:

- ID parameters don't state where to obtain them (data provenance missing)
- Field types are vague ("The phone number" instead of "E.164 format")
- No machine-actionable hints for normalization, prerequisites, or error recovery
- Enum/status values are sometimes documented in overviews but missing from struct references
- Only some resources have troubleshooting sections

## Goal

Add AI-native writing guidelines to `bin-api-manager/CLAUDE.md` so that any writer (human or AI) editing RST files produces documentation that is both human-readable and machine-actionable.

## Relationship to OpenAPI AI-Native Rules

`bin-openapi-manager/CLAUDE.md` already contains "AI-Native OpenAPI Specification Rules" (4 rules covering oneOf polymorphism, strict structured strings, provenance in descriptions, and mandatory realistic examples). These RST guidelines are the **documentation counterpart** to those OpenAPI spec rules.

| Concern | OpenAPI Rules (`bin-openapi-manager/CLAUDE.md`) | RST Rules (`bin-api-manager/CLAUDE.md`) |
|---|---|---|
| Data Provenance | Rule 3: Provenance in YAML `description:` fields | Rule 1: Provenance in RST prose |
| Strict Typing | Rule 2: `format:`, `pattern:`, `enum` in YAML | Rule 2: Explicit types in prose (UUID, E.164, etc.) |
| Examples | Rule 4: `example:` values in YAML | Tutorial files: complete request/response examples |
| Polymorphism | Rule 1: `oneOf` for variant types | N/A (not applicable to prose) |
| AI Hints | N/A | Rule 3: `.. note:: **AI Implementation Hint**` blocks |
| Error Handling | N/A | Rule 5: Cause + Fix pairs |

When updating both an OpenAPI schema and its corresponding RST documentation, both guideline sets apply. The OpenAPI rules govern the YAML spec; the RST rules govern the human/AI-readable documentation pages.

## Decisions Made

| Decision | Choice | Rationale |
|---|---|---|
| Deliverable | Guideline document only | Apply to files incrementally, not all at once |
| Location | `bin-api-manager/CLAUDE.md` new section | Guaranteed to be read by Claude Code before any RST editing |
| Placement | Replaces existing `**RST Style:**` bullets, preserves `**Common Issues:**` | New guidelines subsume the style rules; common issues content is absorbed into RST Formatting Rules |
| Template style | Match existing multi-file pattern | Keep consistency with 80+ existing files |
| Scope | 5 rules + per-file-type guidance | Each file type (overview, struct, tutorial, troubleshooting) gets specific requirements |
| `.. meta::` directive | Intentionally excluded | Existing 80+ files don't use it; marginal value for the migration effort |

## Content Design

### Section Title and Location

New section: `#### AI-Native RST Writing Guidelines`

This is a `####` heading to sit at the same level as the existing `#### Documentation Structure`, `#### Building Documentation`, and `#### Documentation Conventions` — all of which are subsections under `### API Documentation`.

Placement: Replaces the existing `#### Documentation Conventions` section entirely (the `**File Naming:**` block, `**RST Style:**` bullets, and `**Common Issues:**` block). The File Naming content is preserved as-is at the top of the new section. The RST Style and Common Issues content is absorbed into the RST Formatting Rules subsection so nothing is lost.

### Applicability

These rules apply when **creating new RST pages or modifying existing ones**. Do not retroactively rewrite pages you are not otherwise touching. When editing an existing page, apply the rules to the sections you are changing.

### The 5 Commandments

**Rule 1: Data Provenance (Where Do IDs Come From?)**
- Every ID parameter must state its source endpoint
- Pattern: `field_name: Type description. Obtained from \`GET /endpoint\`.`

Bad:
```
queue_id: The ID of the queue.
```

Good:
```
queue_id: The unique UUID of the queue. Obtained from the ``id`` field
of ``GET /queues``.
```

**Rule 2: Strict Typing in Prose**
- Explicit format for every field: UUID, E.164, ISO 8601, enum, Boolean, Integer
- Never use vague terms like "text" or "number"

Type mapping table:

| Vague Term | AI-Native Replacement |
|---|---|
| "The phone number" | "Phone number in E.164 format (e.g., `+821012345678`). Must start with `+`. No dashes or spaces." |
| "The ID" | "UUID string (e.g., `a1b2c3d4-e5f6-7890-abcd-ef1234567890`)" |
| "The timestamp" | "ISO 8601 / RFC 3339 timestamp (e.g., `2026-01-15T10:30:00Z`)" |
| "The status" | "Enum string. One of: `dialing`, `ringing`, `progressing`, `hangup`." |

**Rule 3: AI Hint Admonitions**
- RST directive: `.. note:: **AI Implementation Hint**`
- Use cases: input normalization, prerequisite checks, cost warnings, async behavior
- Required in overview and tutorial files (at least one per page)
- In struct files: include only when field usage has non-obvious requirements (not mandatory)

Example:
```rst
.. note:: **AI Implementation Hint**

   If the user provides a local phone number (e.g., ``010-1234-5678``),
   you **MUST** normalize it to E.164 format (``+821012345678``) before
   calling this API.
```

Use for:
- Input normalization (phone formats, timezone conversions)
- Prerequisite checks ("Verify the flow exists via `GET /flows/{id}` before assigning")
- Cost warnings ("This operation deducts credits from the account balance")
- Async behavior ("This returns immediately. Poll `GET /calls/{id}` or use WebSocket for status updates")

**Rule 4: Explicit State Transitions (Enums)**
- Every enum/status field must list all possible values with descriptions
- Applies only to resources that have a status or state field
- No "Returns the status of the call" without the full enum list

Bad:
```
Returns the status of the call.
```

Good:
```
status (enum string):

* ``dialing``: System is currently dialing the destination.
* ``ringing``: Destination device is ringing, awaiting answer.
* ``progressing``: Call answered. Audio is flowing between parties.
* ``terminating``: System is ending the call.
* ``canceling``: Originator is canceling before answer (outgoing calls only).
* ``hangup``: Call ended. Final state — no further changes possible.
```

**Rule 5: Self-Correcting Error Handling**
- HTTP error codes mapped to: Cause + Fix
- Enables AI self-healing on API call failure

Example:
```rst
Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** The ``to`` field contains dashes or spaces.
    * **Fix:** Remove all non-numeric characters except the leading ``+``.

* **402 Payment Required:**
    * **Cause:** Insufficient account balance.
    * **Fix:** Check balance via ``GET /billing-accounts``. Prompt user to top up.

* **404 Not Found:**
    * **Cause:** The resource UUID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET`` list call.

* **409 Conflict:**
    * **Cause:** Resource is in an incompatible state for this operation.
    * **Fix:** Check current status via ``GET /resource/{id}`` before retrying.
```

### Per-File-Type Requirements

**`*_overview.rst`:**
1. AI Context block at the top:
   ```rst
   .. note:: **AI Context**

      * **Complexity:** Low | Medium | High
      * **Cost:** Free | Chargeable (credit deduction)
      * **Async:** Yes/No. If yes, state how to track status.
   ```
2. State lifecycle with all enum values (if the resource has a status field)
3. Related documentation cross-refs using `:ref:`
4. At least one AI Implementation Hint (Rule 3)

**`*_struct_*.rst`:**
1. Type for every field (UUID, String(E.164), enum, ISO 8601, Boolean, Integer, Array, Object)
2. Provenance for every ID/reference field ("Obtained from `GET /endpoint`") (Rule 1)
3. Required/Optional marker for request body structs
4. Enum values listed inline or via `:ref:` to a dedicated section (Rule 4)
5. AI Implementation Hint only when field usage has non-obvious requirements

Example field description format:
```
* ``flow_id`` (UUID, Optional): The flow to execute when the call is answered.
  Obtained from the ``id`` field of ``GET /flows``.
  Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
```

**`*_tutorial.rst`:**
1. Prerequisites block listing what IDs/resources are needed and how to obtain them:
   ```rst
   Prerequisites
   +++++++++++++

   Before creating a call, you need:

   * A source phone number (E.164 format). Obtain one via ``GET /numbers``.
   * A destination phone number (E.164 format) or extension.
   * (Optional) A flow ID (UUID). Create one via ``POST /flows`` or obtain from ``GET /flows``.
   ```
2. Complete request AND response examples for every operation
3. At least one AI Implementation Hint for common gotchas (Rule 3)
4. Response field annotations — comment key fields (e.g., `// Save this as call_id`)

**`*_troubleshooting.rst`:**
1. Debugging tools — list relevant API endpoints for diagnosis
2. Symptom -> Cause -> Fix pattern for each issue (Rule 5)
3. HTTP error code reference table
4. Diagnostic steps with actual API calls to run

### Quick Checklist

Verification checklist for writers before considering a page complete:

- [ ] Every ID field states its source endpoint (Rule 1)
- [ ] Every field has an explicit type: UUID, E.164, enum, ISO 8601, etc. (Rule 2)
- [ ] Overview and tutorial pages have at least one `.. note:: **AI Implementation Hint**` (Rule 3)
- [ ] Every enum/status lists all possible values with descriptions (Rule 4)
- [ ] Request struct fields are marked Required or Optional (Rule 2)
- [ ] Error scenarios include cause + fix pairs (Rule 5)
- [ ] Cross-references use `:ref:` to link related documentation
- [ ] Code examples include both request AND response

### RST Formatting Rules

Preserved from existing conventions:
- Use `.. code::` for code blocks (not `.. code-block::`)
- Reference other sections: `:ref:\`link-target\``
- Use `**bold**` for emphasis, not `*italic*`
- Keep lines under 120 characters when practical
- Use "VoIPBIN" consistently (not "Voipbin" or "voipbin")
- Always provide content in sections — no empty stubs like "Common used."
- Check for common typos: "Ovewview", "Acesskey", "comming", "existed"
- Grammar: "The doesn't affect" -> "This doesn't affect"

## Implementation Plan

1. Create worktree (done: `NOJIRA-AI-native-rst-writing-guidelines`)
2. Edit `bin-api-manager/CLAUDE.md`:
   - Replace `#### Documentation Conventions` with `#### AI-Native RST Writing Guidelines`
   - Preserve `**File Naming:**` block at the top of the new section
   - Absorb `**RST Style:**` and `**Common Issues:**` into RST Formatting Rules — no content lost
   - Keep `#### Documentation Structure`, `#### Building Documentation`, `#### Documentation Maintenance`, and all other existing content unchanged
3. No RST files modified (guideline-only deliverable)
4. Commit and create PR

### Follow-Up (Out of Scope)

- Consider adding a cross-reference in root `CLAUDE.md` pointing to these RST guidelines, similar to the existing reference to `bin-openapi-manager/CLAUDE.md` for OpenAPI rules (line 570)

## What This Does NOT Cover

- Rewriting existing RST files (future work, file-by-file)
- OpenAPI spec guidelines (covered in `bin-openapi-manager/CLAUDE.md`, cross-referenced above)
- Non-RST documentation (Swagger annotations, code comments)
- `.. meta::` directives (intentionally excluded — existing 80+ files don't use them, marginal value)
