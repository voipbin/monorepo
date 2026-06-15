# describe_action LLM tool design

Date: 2026-06-15
Service: bin-ai-manager
Type: LLM tool (new) + flow-manager sync note
Branch: NOJIRA-Add-describe-action-tool
Follow-up to: #994 (option catalog for create_call inline actions; create_call
shipped in PR #993)

## 1. Problem Statement

PR #993 let the AI assemble flow actions inline via `create_call`. The tool
schema offers the full set of valid action TYPES (enum locked to flow-manager
`action.TypeListAll`), but it does NOT tell the LLM each action's OPTION fields.
The LLM must guess option shapes (e.g. that `talk` takes `{text, language}`,
`connect` takes `{destinations}`, `branch` takes `{target_ids,
default_target_id}`). This makes assembling a correct non-trivial flow
unreliable.

We want a way for the LLM to look up, on demand, what option fields a given
action type accepts, without bloating every tool-call payload with all 42 option
schemas.

## 2. Scope

### In scope (Phase 1)

- A new LLM tool `describe_action(action_type)` in bin-ai-manager that returns a
  compact, human/LLM-readable description of a single action type: its purpose
  and its option fields (name, type, required/optional, one-line meaning).
- A hand-authored compact catalog (one entry per action type) living in
  bin-ai-manager, condensed from the canonical sources
  (`bin-flow-manager/models/action/option.go` comments and the user-facing
  `bin-api-manager/docsdev/source/flow_struct_action.rst`).
- Two automated drift tests over the catalog: a type-set test asserting one
  entry per `action.TypeListAll` type (no missing / extra / duplicate), and a
  reflection-based field-parity test asserting each entry's option field NAMES
  equal the real option struct's top-level json fields (closes the stale-field
  gap without codegen). Plus a render golden test. See 3.4.
- A new `action.OptionStructByType` map in flow-manager (Type -> zero-value
  option struct) that the field-parity test reflects over, itself drift-guarded.
- A sync note in `bin-flow-manager/models/action/option.go` telling future
  editors that changing an option struct requires updating the ai-manager
  describe_action catalog (naming the test that will fail), covering the residual
  semantic-text drift that cannot be machine-checked.

### Out of scope

- Auto-GENERATING the catalog from option.go via reflection or AST/codegen.
  Decision (owner, approach 2): reflection cannot read field comments (they are
  dropped at compile time), and an AST/go:generate pipeline is heavier to
  maintain than a hand-authored compact catalog at the current stage. NOTE the
  distinction: v2 uses reflection only to VERIFY field names (a test), never to
  generate catalog content — that is within approach 2.
- A `list_flow_actions` tool. The action-type list is already exposed by the
  `create_call` `actions[].type` enum; a separate listing tool is redundant.
- Per-action option VALUE validation (e.g. checking a `goto.target_id` resolves).
  That is flow-manager's runtime concern.
- Deep nested-type expansion (e.g. fully expanding `commonaddress.Address` or
  `email.Attachment` sub-fields). The catalog names the field and its shape at
  one level ("destinations: array of address objects {type, target,
  target_name}") rather than recursively documenting every nested model.

## 3. Design

### 3.1 Catalog data structure

A package-level slice in bin-ai-manager (co-located with the tool handler),
one entry per action type:

```go
// actionCatalogEntry describes a single flow action type for the LLM.
type actionCatalogEntry struct {
    Type        fmaction.Type        // the action type (e.g. fmaction.TypeTalk)
    Summary     string               // one-line purpose
    Options     []actionOptionField  // option fields; empty for option-less actions (e.g. answer)
}

type actionOptionField struct {
    Name        string // json field name, e.g. "text" (MUST match the option struct's json tag)
    Type        string // human type, e.g. "string", "int (ms)", "array of address {type,target}"
    Required    bool   // whether the action is meaningless without it
    Description string // one-line meaning, condensed from option.go / RST
}
```

The catalog is the authoritative source slice (`actionCatalog []actionCatalogEntry`).
A lookup map `map[fmaction.Type]actionCatalogEntry` is built ONCE from it via a
package-level `var` initialized at load (or `sync.Once`), never lazily on the
request path, so there is no concurrency hazard. The map build MUST reject
duplicate Types (panic at init or via the drift test, see 3.4) so a duplicated
catalog entry cannot silently shadow another.

### 3.2 Tool definition (definitions.go)

```
Name: describe_action
RunLLM: true
Description: "Returns the option fields a given flow action type accepts, so you
  can correctly assemble actions for create_call's 'actions' parameter. Call this
  before assembling an action whose options you are unsure of (e.g. connect,
  branch, talk, play). The action_type must be one of the create_call action
  types."
Parameters:
  action_type (string, required, enum = same 42 types as create_call): the action
    type to describe.
  run_llm (boolean, default true)
```

The `action_type` enum is the same 42-value set as `create_call`. To avoid a
THIRD hand-maintained copy of the 42 strings, both enums are built from a single
shared helper (see 3.5).

### 3.3 Handler (toolhandler or aicallhandler)

`describe_action` is a pure, read-only, customer-agnostic lookup (it returns
static schema text, touches no customer data and no RPC). It therefore does NOT
need the aicall context the way create_call does. It can live as a toolhandler-
level handler.

Flow:
1. Parse `action_type` from arguments.
2. Look up the catalog entry. If not found (LLM passed a non-enum value), return
   a `fillFailed` whose message ECHOES the received value and names the valid
   types (so the LLM can self-correct a typo/casing). Because the schema enum
   constrains it, this is a defense-in-depth branch.
3. Render the entry to a compact readable string (summary + each option field as
   `name (type, required|optional): description`). For option-less actions,
   state "This action takes no options."
4. `fillSuccess` with the rendered text.

No RPC, no goroutine, no DB. Failure modes: empty/missing action_type -> fillFailed;
unknown type -> fillFailed (with echo + valid-type hint).

The render format is a load-bearing contract (it goes into the LLM prompt), so a
GOLDEN test pins the exact rendered string for at least one option-bearing entry
(e.g. talk) and one option-less entry, so accidental format changes are caught.

### 3.4 Drift defense (the core of approach 2)

Hand-authored catalogs drift from the source. Three layered defenses, the first
two automated (no codegen — these VERIFY the hand-written catalog, they do not
GENERATE it), the third a human backstop:

1. **Type-set drift-lock test** `TestActionCatalogMatchesTypeListAll`: asserts the
   multiset of `Type` values in the catalog equals `action.TypeListAll` exactly.
   It iterates the raw `actionCatalog` SLICE (not the lookup map) and (a) rejects
   duplicate Types explicitly, then (b) compares sorted length + per-index against
   `TypeListAll`. Iterating the slice (not the map) is required so a duplicated
   entry is caught rather than silently deduped. This catches MISSING / EXTRA /
   DUPLICATE type entries.

2. **Field-parity reflection test** `TestActionCatalogFieldsMatchOptionStructs`:
   for each `action.TypeListAll` type, look up its option struct via a new
   `action.OptionStructByType map[Type]any` (zero-value structs), reflect over the
   struct's TOP-LEVEL json field names, and assert that set equals the set of
   `Options[].Name` in the catalog entry. Reflection CAN read field names and json
   tags (only comments are lost at compile time), so this closes the stale-FIELD
   gap automatically. Rules:
   - Compare TOP-LEVEL json field NAMES only (matches the catalog's deliberate
     one-level, non-recursive scope; nested structs are named by shape in the
     option field's `Type` string, not expanded). The test compares the NAME set,
     not the `Type` string, so a field whose value is itself a struct/slice still
     has a top-level json name (e.g. `destinations`) present in both sets.
   - Honor json tag semantics: split on `,` to drop `omitempty`; skip `json:"-"`
     fields; skip unexported fields (`PkgPath != ""`); for an exported field with
     NO json tag, fall back to the Go field name (Go's default json key), so an
     untagged field added later is still checked.
   - The `OptionStructByType` map is itself drift-guarded: the same test asserts
     every `TypeListAll` type has an entry in it, so the map cannot go stale
     silently. All 42 option structs exist 1:1 with TypeListAll; an action with
     no options maps to a zero-FIELD (empty) struct, NOT a missing entry, and its
     catalog entry has an empty `Options`.

   What this test does NOT catch (and the sync note therefore must cover): the
   semantic TEXT fields the LLM reads — `Summary`, `Description`, `Required`, and
   the human option `Type` string (e.g. if an option struct field changes from
   `string` to `int`, the json NAME is unchanged so this test still passes; the
   catalog `Type` string would go stale). These cannot be machine-checked against
   comments and are the residual covered by defense 3.

3. **Sync note in option.go**: a comment in
   `bin-flow-manager/models/action/option.go` (and next to `OptionStructByType`)
   instructing editors that changing an option struct's fields requires updating
   the ai-manager describe_action catalog, naming the exact tests that will fail
   (`TestActionCatalogFieldsMatchOptionStructs`) so a CI failure points straight
   to the fix. The note also reminds editors that the user-facing RST
   (`flow_struct_action.rst`) documents the same fields and may need updating.
   This is the human backstop for the description text only.

### 3.5 Single source for the 42-type enum

`create_call` (definitions.go) and `describe_action` both need the 42-value
`action_type` enum. To avoid a third hand-maintained copy, extract a helper that
returns the enum as `[]string` from `action.TypeListAll`:

```go
// in toolhandler (or a shared spot)
func actionTypeEnum() []string {
    out := make([]string, len(fmaction.TypeListAll))
    for i, t := range fmaction.TypeListAll {
        out[i] = string(t)
    }
    return out
}
```

Both `create_call` and `describe_action` use `actionTypeEnum()` for their schema
enum. This makes the create_call enum self-syncing too (and the existing
`TestCreateCallActionsEnumMatchesTypeListAll` keeps guarding it). NOTE: this
changes create_call's enum from a hardcoded literal to the helper; the existing
drift test still passes because it compares against TypeListAll either way.

## 4. Catalog content sourcing

Each entry condensed from, in priority order:
1. `bin-flow-manager/models/action/option.go` field comments (authoritative field
   names + meanings).
2. `bin-api-manager/docsdev/source/flow_struct_action.rst` (user-facing prose,
   examples) for the Summary line and clarifications.

Option-less / internal action types (e.g. `answer`, `hangup`, `beep`, `echo`,
`ai_summary`) still get an entry: Summary + "no options" (or their minimal
options). Internal/infra types (`external_media_start`, `confbridge_join`, etc.)
are included for completeness since they are in the enum, with a brief note that
they are advanced/infra actions.

## 5. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-ai-manager | new describe_action tool: catalog data, tool definition, handler + render fn, drift-lock test (type-set), field-parity reflection test, render golden test; refactor create_call enum to shared `actionTypeEnum()` helper | 1 |
| bin-flow-manager | new `OptionStructByType` map in models/action; sync note comment next to it and at option.go top | 1 |

No DB migration. No RPC change. No OpenAPI change (internal LLM tool). Internal
tool -> no RST `docsdev` update required.

## 6. Tool registration

Register `describe_action` in:
- `models/tool/main.go`: `ToolNameDescribeAction` + add to `AllToolNames`.
- `models/message/tool.go`: `FunctionCallNameDescribeAction` (if the dispatch
  path requires it).
- `toolhandler/definitions.go`: the tool definition.
- the dispatch map (toolhandler or aicallhandler `ToolHandle` mapFunctions).
- whitelist (`toolhandler/whitelist.go`) if tools are gated by a whitelist.

(Confirm the exact 5 touch points against the create_call/get_resource
registration during implementation.)

## 7. Observability

- Debug log on describe_action invocation (action_type requested). No metrics
  (static lookup, no cost, no async).

## 8. Security

- describe_action returns only STATIC schema text (action option shapes). No
  customer data, no RPC, no resource ids. No IDOR / ownership surface. It is
  customer-agnostic by construction.
- It does reveal the full set of action types and their option fields to any
  caller, but that is public product documentation (already on docs.voipbin.net),
  so there is no information-disclosure concern.

## 9. Implementation Order

1. flow-manager: add `OptionStructByType map[Type]any` + sync notes in
   models/action/option.go.
2. ai-manager: `actionTypeEnum()` helper; refactor create_call enum to use it
   (keep `TestCreateCallActionsEnumMatchesTypeListAll` green).
3. ai-manager: catalog data (one entry per TypeListAll type, fields matching the
   option struct json tags).
4. ai-manager: `ToolNameDescribeAction` registration (5 touch points).
5. ai-manager: describe_action tool definition + handler + render function.
6. ai-manager tests: type-set drift-lock, field-parity reflection, render golden.
7. Full verification workflow in BOTH bin-flow-manager (new map + comment) and
   bin-ai-manager (go mod tidy/vendor/generate/test/lint each).

## 10. Open Questions

| Question | Recommendation | Owner |
|---|---|---|
| Should the catalog expand nested model fields (Address, Attachment)? | No; describe one level + name the nested shape. Keep compact | CPO |
| Should describe_action be in the default tool whitelist? | Yes; it is safe (static, read-only) and directly supports create_call assembly | CPO |
| Phase 2 codegen from option.go AST? | Defer; revisit only if catalog drift becomes a recurring problem | CTO |

## 11. Review Summary

### v1 -> v2 (first design review round)

Two independent design reviews (soundness + maintainability/drift) both returned
CHANGES REQUESTED, converging on the same key gap. v2 applied:

- **Field-parity reflection test added (the key finding).** Both reviewers
  independently flagged that the v1 type-set test only catches missing/extra
  TYPES, not stale option FIELDS — leaving the tool's whole reason for existing
  (field accuracy) on a single human comment. Since reflection CAN read field
  names + json tags (only comments are lost), v2 adds
  `TestActionCatalogFieldsMatchOptionStructs` + a new flow-manager
  `OptionStructByType` map. This VERIFIES (does not GENERATE) the hand-written
  catalog, so it stays within the owner's approach-2 (no codegen) decision.
- **Duplicate-Type masking fixed.** v2's type-set test iterates the raw slice and
  explicitly rejects duplicate Types (the map-keyed comparison could silently
  dedup a duplicated entry).
- **OptionStructByType self-guarded.** The field-parity test asserts every
  TypeListAll type has a map entry, so the new map cannot go stale silently.
- **Failure-message quality.** describe_action's not-found path now echoes the
  received value + names valid types for LLM self-correction; empty/missing
  action_type is handled.
- **Render golden test.** The rendered string is a prompt contract, so a golden
  test pins it for one option-bearing + one option-less entry.
- **Thread-safety / init clarified.** Lookup map built once at load (var/sync.Once),
  never lazily on the request path.
- **Sync note strengthened.** Names the exact failing test and reminds editors the
  RST doc also documents the fields. Residual covered by the note is now only the
  semantic text (Summary/Description/Required), which is genuinely un-checkable.
- **PR reference corrected** (#993 shipped create_call; this is #994's catalog).

Deferred (Open Questions): nested-model field expansion (no, one level);
Phase-2 codegen from option.go AST (defer unless drift recurs).

## Implementation Addendum (2026-06-15)

During implementation a real import cycle surfaced: an existing `toolhandler`
test imports `aicallhandler` (get_resource enum parity), and the dispatch handler
lives in `aicallhandler`, which now needed to call the catalog. Putting the
catalog in `toolhandler` would have created `toolhandler -> aicallhandler ->
toolhandler` (test-time) cycle.

Resolution: the catalog, `ActionTypeEnum()`, `DescribeAction()`, render, and the
drift tests live in a new leaf package `bin-ai-manager/pkg/actioncatalog` that
imports only `bin-flow-manager/models/action`. Both `toolhandler/definitions.go`
(enum) and `aicallhandler/tool_describeaction.go` (handler) import it. No cycle.

Type->option-struct mapping: `bin-flow-manager/models/action.OptionStructByType`
plus a self-guard test `Test_OptionStructByType_CoversTypeListAll`. Two action
types (`mute`, `stop`) have no option struct in flow-manager; they map to
`struct{}{}` (zero top-level json fields), so the field-parity test treats them
as option-less. Catalog entries for them are rendered "this action takes no
options."

Final touch points (5 confirmed):
- `models/tool/main.go`: `ToolNameDescribeAction` + `AllToolNames`.
- `models/message/tool.go`: `FunctionCallNameDescribeAction`.
- `toolhandler/definitions.go`: the tool definition (+ create_call enum now uses
  `actioncatalog.ActionTypeEnum()` instead of a hardcoded literal).
- `aicallhandler/tool.go` mapFunctions + `aicallhandler/tool_describeaction.go`.
- `toolhandler/whitelist.go`: added to `ConversationSafeTools`.
