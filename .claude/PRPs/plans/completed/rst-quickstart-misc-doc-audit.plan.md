# Plan: RST Quickstart & Miscellaneous Documentation Audit

## Summary
Audit all non-struct, non-overview, non-tutorial RST files against the current codebase. This covers quickstart guides, architecture docs, call/flow specialized docs, intro/glossary/misc files, and toctree index containers. Verify endpoint URLs against OpenAPI spec, field references against WebhookMessage structs, AI-Native RST guideline compliance, and factual accuracy.

## User Story
As an external developer or AI agent, I want quickstart guides and reference docs to accurately reflect the current API, so I can onboard and integrate without encountering stale instructions.

## Problem → Solution
Quickstart guides, architecture docs, and specialized reference files have not been systematically audited → Verify each file against OpenAPI spec and WebhookMessage structs, fix discrepancies, ensure AI-Native compliance.

## Metadata
- **Complexity**: Large
- **Source PRD**: `.claude/PRPs/prds/rst-docs-full-audit-sync.prd.md`
- **PRD Phase**: Phase 4 — Quickstart & misc audit
- **Estimated Files**: ~42 content files + ~35 toctree index files + HTML rebuild

---

## UX Design

N/A — internal change (documentation accuracy improvement).

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `bin-api-manager/CLAUDE.md` | 176-315 | AI-Native RST Writing Guidelines |
| P0 | `bin-openapi-manager/openapi/openapi.yaml` | paths section | Endpoint verification |
| P1 | `bin-api-manager/docsdev/source/quickstart.rst` | all | Main quickstart container |
| P1 | `bin-api-manager/docsdev/source/index.rst` | all | Master toctree |
| P2 | `.claude/PRPs/reports/rst-overview-doc-audit-report.md` | all | Phase 2 patterns |
| P2 | `.claude/PRPs/reports/rst-tutorial-doc-audit-report.md` | all | Phase 3 patterns |

## External Documentation

No external research needed — feature uses established internal patterns from Phases 1-3.

---

## Patterns to Mirror

### FQDN_URL_PATTERN
// SOURCE: Phase 2/3 established pattern
All curl commands and endpoint references must use fully qualified URLs:
`https://api.voipbin.net/v1.0/` prefix, never bare paths like `/calls`.

### AI_IMPLEMENTATION_HINT
// SOURCE: bin-api-manager/CLAUDE.md:240-252
```rst
.. note:: **AI Implementation Hint**

   [Specific, actionable guidance for AI agents]
```
Required in quickstart and tutorial files. At least one per page.

### AI_CONTEXT_BLOCK
// SOURCE: bin-api-manager/CLAUDE.md:265-271
```rst
.. note:: **AI Context**

   * **Complexity:** Low | Medium | High
   * **Cost:** Free | Chargeable (credit deduction)
   * **Async:** Yes/No. If yes, state how to track status.
```
Required in overview files. Not required for quickstart/architecture/glossary.

### STRICT_TYPING
// SOURCE: bin-api-manager/CLAUDE.md:222-232
Use specific types: UUID, E.164, enum, ISO 8601, Boolean, Integer, Array, Object.
Never use vague terms like "text", "number", "the ID".

### TROUBLESHOOTING_PATTERN
// SOURCE: bin-api-manager/CLAUDE.md:254-264
```rst
* **400 Bad Request:**
    * **Cause:** ...
    * **Fix:** ...
```

---

## Files to Change

### Batch 4A: Quickstart Files (12 files)

| File | Action | Justification |
|---|---|---|
| `quickstart.rst` | AUDIT | Main container — verify includes and cross-refs |
| `quickstart_signup.rst` | AUDIT | Verify signup flow matches current API |
| `quickstart_authentication.rst` | AUDIT | Verify auth endpoints and token/accesskey flow |
| `quickstart_extension.rst` | AUDIT | Verify extension creation endpoint and fields |
| `quickstart_events.rst` | AUDIT | Verify webhook/websocket setup |
| `quickstart_call.rst` | AUDIT | Verify call creation endpoint and response fields |
| `quickstart_realtime.rst` | AUDIT | Verify real-time/streaming endpoints |
| `quickstart_sandbox.rst` | AUDIT | Verify sandbox mode description |
| `quickstart_email.rst` | AUDIT | Verify email endpoint and fields |
| `quickstart_message.rst` | AUDIT | Verify message endpoint and fields |
| `quickstart_queue.rst` | AUDIT | Verify queue endpoint and fields |
| `quickstart_transcribe.rst` | AUDIT | Verify transcribe endpoint and fields |

### Batch 4B: Architecture Files (10 files)

| File | Action | Justification |
|---|---|---|
| `architecture.rst` | AUDIT | Container file — verify toctree |
| `architecture_backend.rst` | AUDIT | Verify service descriptions match current services |
| `architecture_communication.rst` | AUDIT | Verify communication patterns |
| `architecture_data.rst` | AUDIT | Verify data model descriptions |
| `architecture_dataflow.rst` | AUDIT | Verify data flow descriptions |
| `architecture_deployment.rst` | AUDIT | Verify deployment descriptions |
| `architecture_flow.rst` | AUDIT | Verify flow engine description |
| `architecture_rtc.rst` | AUDIT | Verify real-time communication descriptions |
| `architecture_security.rst` | AUDIT | Verify security model description |
| `architecture_sequences.rst` | AUDIT | Verify sequence diagrams |

### Batch 4C: Call Specialized Files (6 files)

| File | Action | Justification |
|---|---|---|
| `call_groupcall.rst` | AUDIT | Verify groupcall fields against WebhookMessage |
| `call_media.rst` | AUDIT | Verify media/recording descriptions |
| `call_scenarios.rst` | AUDIT | Verify call scenario descriptions |
| `call_sequences.rst` | AUDIT | Verify sequence diagrams |
| `call_transfer.rst` | AUDIT | Verify transfer fields against WebhookMessage |
| `call_troubleshooting.rst` | AUDIT | Verify troubleshooting entries |

### Batch 4D: Flow Specialized + Intro/Misc (11 files)

| File | Action | Justification |
|---|---|---|
| `flow_advanced_patterns.rst` | AUDIT | Verify flow action patterns |
| `flow_best_practices.rst` | AUDIT | Verify best practice recommendations |
| `flow_debugging.rst` | AUDIT | Verify debugging guidance |
| `flow_execution_internals.rst` | AUDIT | Verify execution model description |
| `intro.rst` | AUDIT | Verify intro accuracy |
| `intro_applications.rst` | AUDIT | Verify application descriptions |
| `intro_channels.rst` | AUDIT | Verify channel descriptions |
| `glossary.rst` | AUDIT | Verify term definitions are current |
| `restful_api.rst` | AUDIT | Verify API convention descriptions |
| `support.rst` | AUDIT | Verify support info |
| `variable_variable.rst` | AUDIT | Verify variables against codebase |

### Batch 4E: Index/Toctree Files + SDK (~36 files)

| File | Action | Justification |
|---|---|---|
| `index.rst` | AUDIT | Master toctree — verify all resources listed |
| `sdk.rst` | AUDIT | SDK index — verify content |
| ~35 resource index files | AUDIT | Verify toctree includes match existing files |

### Batch G: HTML Rebuild

| File | Action | Justification |
|---|---|---|
| `bin-api-manager/docsdev/build/` | REBUILD | Sphinx HTML must match RST sources |

## NOT Building

- New documentation pages for undocumented services
- Changes to overview, tutorial, or struct RST files (done in Phases 1-3)
- Automated CI-based doc drift detection
- Rewriting documentation style or structure

---

## Step-by-Step Tasks

### Task 4A: Quickstart Files Audit
- **ACTION**: Read each of 12 quickstart files. For each:
  1. Verify curl endpoint URLs exist in OpenAPI spec
  2. Verify response JSON fields match WebhookMessage structs
  3. Verify FQDN compliance (all URLs use `https://api.voipbin.net/v1.0/`)
  4. Verify at least one AI Implementation Hint per file
  5. Fix any discrepancies
- **MIRROR**: FQDN_URL_PATTERN, AI_IMPLEMENTATION_HINT, STRICT_TYPING
- **GOTCHA**: quickstart files use `.. include::` directives — each included file is standalone
- **VALIDATE**: All curl commands use FQDN URLs, response fields match WebhookMessage

### Task 4B: Architecture Files Audit
- **ACTION**: Read each of 10 architecture files. For each:
  1. Verify service names and descriptions are current
  2. Verify diagrams reference current services/components
  3. Verify no references to removed or renamed services
  4. Add AI Implementation Hint where useful (these are educational docs)
  5. Preserve ASCII art diagrams exactly (per Phase 2 convention)
- **MIRROR**: FQDN_URL_PATTERN (for any URL references)
- **GOTCHA**: Architecture docs are internal-facing — less strict on AI-Native rules than API docs. `architecture_overview.rst` was already audited in Phase 2 — skip it.
- **VALIDATE**: No references to non-existent services or endpoints

### Task 4C: Call Specialized Files Audit
- **ACTION**: Read each of 6 call specialized files. For each:
  1. Verify field references match Call/Groupcall/Transfer WebhookMessage structs
  2. Verify endpoint URLs in examples
  3. Verify sequence diagrams match current call flow
  4. Add/verify AI Implementation Hints
  5. Verify troubleshooting entries have cause+fix pairs
- **MIRROR**: FQDN_URL_PATTERN, STRICT_TYPING, TROUBLESHOOTING_PATTERN
- **GOTCHA**: call_groupcall.rst may reference GroupcallManagerGroupcall WebhookMessage
- **VALIDATE**: All field references exist in corresponding WebhookMessage structs

### Task 4D: Flow Specialized + Intro/Misc Files Audit
- **ACTION**: Read each of 11 files. For each:
  1. Verify flow action names/parameters match current codebase
  2. Verify variable names in variable_variable.rst match code
  3. Verify glossary terms are current
  4. Verify intro descriptions match current platform capabilities
  5. Add AI Implementation Hints where appropriate
- **MIRROR**: AI_IMPLEMENTATION_HINT, STRICT_TYPING
- **GOTCHA**: variable_variable.rst should be checked against actual flow variable implementations
- **VALIDATE**: All variable names and glossary terms are current

### Task 4E: Index/Toctree Files Audit
- **ACTION**: Read index.rst and all ~35 resource index files. For each:
  1. Verify toctree includes match files that exist
  2. Verify no missing or extra entries
  3. Verify cross-reference labels are valid
- **MIRROR**: Standard RST toctree format
- **GOTCHA**: These are small files (8-19 lines) — quick to audit
- **VALIDATE**: All toctree entries reference existing files

### Task G: Sphinx HTML Rebuild
- **ACTION**: Clean rebuild all HTML
  ```bash
  cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
  ```
- **VALIDATE**: Build exits cleanly with zero new warnings. Force-add: `git add -f bin-api-manager/docsdev/build/`

---

## Testing Strategy

### Validation
- Sphinx build with zero new warnings
- FQDN URL compliance (grep for bare paths in modified files)
- AI Implementation Hint presence (grep for "AI Implementation Hint" in quickstart/tutorial files)
- No references to non-existent endpoints or fields

### Edge Cases Checklist
- [ ] Files that are pure toctree containers (no content to audit)
- [ ] Architecture ASCII art diagrams (preserve exactly)
- [ ] Files already audited in Phase 2 (architecture_overview.rst — skip)
- [ ] Variable names that may have changed since docs were written

---

## Validation Commands

### Sphinx Build
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```
EXPECT: Zero errors, pre-existing warnings only (~518)

### FQDN Compliance
```bash
grep -rn "curl.*'/v1\." bin-api-manager/docsdev/source/quickstart*.rst | grep -v "api.voipbin.net"
```
EXPECT: No results (all curl commands use FQDN)

### AI Hint Check
```bash
grep -l "AI Implementation Hint" bin-api-manager/docsdev/source/quickstart*.rst | wc -l
```
EXPECT: 12 (one per quickstart file, or at least in the container)

---

## Acceptance Criteria
- [ ] All 12 quickstart files audited — endpoints and fields verified
- [ ] All 10 architecture files audited — service descriptions current
- [ ] All 6 call specialized files audited — fields match WebhookMessage
- [ ] All 4 flow specialized files audited — action patterns current
- [ ] All intro/glossary/misc files audited — content current
- [ ] All index/toctree files audited — includes match existing files
- [ ] Sphinx HTML rebuilt cleanly
- [ ] Zero new Sphinx warnings introduced

## Completion Checklist
- [ ] AI-Native RST guidelines followed for all modified files
- [ ] FQDN URLs used in all curl examples
- [ ] No references to non-existent endpoints or fields
- [ ] Architecture diagrams preserved exactly
- [ ] HTML build committed with `git add -f`
- [ ] Report created at `.claude/PRPs/reports/rst-quickstart-misc-doc-audit-report.md`

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Large file count causes context overflow | HIGH | Medium | Batch by category, use subagents |
| Variable names changed without doc updates | MEDIUM | Low | Grep codebase for variable definitions |
| Architecture docs reference removed services | LOW | Low | Cross-reference with current service list |

## Notes
- `architecture_overview.rst` was already audited in Phase 2 — skip it
- Toctree index files are trivial (~8-19 lines) — batch them together
- Quickstart files are the highest priority — they're the first thing new users see
- Architecture docs are internal-facing, so AI-Native rules are less strict
