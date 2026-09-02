# Case Insight Assistant: `get_contact_profile` tool

Date: 2026-09-02
Status: Finalized (round-1 through round-7 design review incorporated; 2 consecutive approvals at rounds 6+7). Implementation plan (tasks/todo.md) separately reviewed and approved.
Related: VOIP-1234 (Insight Assistant tool set + implicit-scoping design), design docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md

## Revision history

- rev2: Incorporated round-1 design review (2 independent reviewers,
  REQUEST_CHANGES both). Key corrections: `ContactV1ContactGet` is NOT
  tenant-scoped server-side (rev1 wrongly claimed it was) -- the
  cross-tenant response check is the sole enforcement and is now marked
  mandatory, with an audit log line added. Added the `ReferenceType` entry
  guard all sibling tools have. Dropped argument parsing (tool takes no
  arguments at all, matching `get_related_cases` exactly -- rev1
  contradicted itself between §3.1 "no arguments" and a §4 "invalid JSON"
  test case). Body now goes through the existing `renderBodyLines` /
  `capSummaryRunes` cap. Free-text fields are `%q`-quoted. Dropped the
  `tags:` line and scoped `addresses:` to `Type`/`Target` only, both
  capped. §5 now states the third-party-LLM PII transmission and the
  `AllInsightToolNames` rollout impact explicitly.
- rev3: Incorporated round-2 design review (2 independent reviewers,
  REQUEST_CHANGES both). Fixes: (1) `renderBodyLines` was being misused --
  identity fields (`name`/`company`/`job_title`) and address lines were
  mixed into one `lines` slice, so on overflow the truncation walk (which
  keeps the newest and drops the oldest, front-first) would drop identity
  fields FIRST and keep raw phone/email lines -- both the wrong UX and a
  privacy-inverted outcome. Fixed by moving identity fields into the
  `header` argument (never truncated) and passing ONLY address lines as
  `lines`. (2) The mandatory tenant check's error-log line risked a
  nil-pointer dereference on the `contact == nil` branch (logging
  `contact.CustomerID` unconditionally) -- fixed to log a nil-safe value.
  (3) The "byte-identical masking" claim in §3.4 was false as written (the
  nil-`ContactID` branch returns a different string than the masked-path
  string) -- corrected to state the actual (harmless) distinguishability.
  (4) §5's residual-risk framing cited the WRONG precedent (2026-07-30
  §1.1's "residual risk" note is about `get_related_cases` exposing
  same-tenant case metadata, a different risk class -- NOT about
  third-party PII egress, and not about `get_case_notes` as rev2
  mis-cited). This tool is the first Insight tool to send contact PII
  (phone/email) to a third-party LLM processor, auto-enabled for every
  existing `tool_names=["all"]` Insight AI with no opt-in gate. This is a
  genuine product/business decision, not something this document is
  entitled to self-approve -- **escalated to 대표님, who explicitly chose
  to ship addresses in v1 with immediate auto-activation for all existing
  Insight AIs** (decision recorded 2026-09-02; see §5). (5) Added the
  missing OpenAPI schema + RST doc sync to §3.5, required by root
  `CLAUDE.md`'s "CRITICAL: RST docs sync" and the `AIManagerToolName` enum
  in `bin-openapi-manager/openapi/openapi.yaml`. (6) Narrowed the address
  cap from `insightMaxListLimit` (50, a list-tool default) to a
  profile-appropriate `insightContactAddressLimit` (5). (7) §4's audit-log
  test requirement downgraded to a behavioral assertion (masked response
  shape) since this package has no `logrus/hooks/test` mechanism today --
  noted as a testing gap, not blocked on adding one.
- rev4: Incorporated round-3 design review (2 independent reviewers,
  REQUEST_CHANGES both, but narrow/doc-local this round -- no
  re-architecture). Fixes: (1) the rev3 nil-safety fix for the
  cross-tenant audit log over-corrected and dropped the foreign tenant id
  entirely -- restored `case_id`/`contact_customer_id` via a nil-guarded
  local, matching the 2026-07-30 precedent's audit field set and the
  sibling handlers' `case_customer_id` convention. (2) §5's rev3 claim
  "this is a first" (PII egress to an LLM vendor) was itself a new
  overclaim -- `get_contact_interactions` already sends raw phone/email as
  `peer=type/target`; corrected to the accurate, narrower claim (first tool
  to expose the FULL reachable address book, not just case-touched
  addresses) and added the more on-point citation: the 2026-07-30 doc's
  Non-goals section explicitly deferred this exact tool pending a "shared
  scope-verification helper," which this design's Case-derived-ContactID +
  mandatory-response-check pattern satisfies inline. (3) §3.5's OpenAPI
  wiring bullet named only the enum value list, not the parallel
  `x-enum-varnames` list that must move in lockstep -- fixed. (4) Added the
  missing `models/ai/allowed_tools_test.go` `knownReadOnly` allowlist
  update to §3.5 -- a deliberate consent-gate test that would otherwise
  fail red. (5) §6 item 2's stale "four wiring points" count (grown to
  roughly a dozen across revisions) now points to §3.5 as the
  authoritative list instead of repeating a number. (6) §3.4 now notes the
  address cap is primary-preserving (not arbitrary) because
  `AddressListByContactID` orders `is_primary desc, tm_create asc`, and
  softened the "identity fields in `header` are never truncated" claim to
  account for the `capSummaryRunes` last-resort safety net.
- rev5: Incorporated round-5 review (a real MEDIUM-severity finding, not a
  wording nit -- resets the consecutive-approval count after round 4's two
  approvals). rev4 had claimed `renderBodyLines`' built-in "most recent N"
  truncation marker only misfires on a rare oversized-`Company`/`DisplayName`
  overflow path. That reachability claim was wrong: `renderBodyLines`
  forces its marker whenever `pagedOut=true` is passed, regardless of
  whether the text actually overflows the rune cap -- so it would have
  fired, with misleading "most recent" wording, on the ORDINARY, common
  path of any contact with more than `insightContactAddressLimit` (5)
  reachable addresses, not just a rare edge case. Fixed at the design level
  (not just reworded): §3.4 now writes its own honest, ordering-agnostic
  truncation note (`(showing %d of %d addresses)`) into `header` and
  deliberately passes `pagedOut=false` to `renderBodyLines`, so the
  misleading built-in marker never fires on the common path. The
  already-correctly-scoped round-4 finding (primary-first head-cap vs.
  `renderBodyLines`' tail-preserving walk, reachable only in the genuine
  rare last-resort-overflow case) is preserved as an accepted residual
  risk at its correct, narrow frequency. §4's address-cap test case and
  the doc's Status line are updated to match.

## 1. Problem

The Insight Assistant (`ai.TypeInsight`) currently exposes four read-only
tools, all scoped to the current Case:

- `get_contact_interactions`
- `get_conversation_content`
- `get_related_cases`
- `get_case_notes`

None of them return the contact's own profile (name, company, job title,
etc). An agent asking the assistant "누구랑 얘기하고 있는 거야" / "이 사람
회사가 어디야" today gets no answer, even though `bin-contact-manager`
already stores this data and the Case is already linked to the Contact via
`Case.ContactID`. Confirmed no overlap: `get_resource` does not support a
`contact` resource type (`tool_resource.go` `mapResourceFetchers`) and is
not part of `AllInsightToolNames` regardless; the Insight system prompt
injects no contact context. The gap is real.

## 2. Goal

Add a fifth Insight tool, `get_contact_profile`, that returns the profile of
the contact linked to the current Case: name fields, company, job title, and
a capped set of reachable addresses (phone/email, target only). Read-only,
no new RPC needed.

**Non-goals:**
- Editing/creating contact data (out of scope for Insight Assistant, which
  is read-only by design).
- Exposing `Notes` (free-text CRM notes), `ExternalID`/`Source` (internal
  integration metadata), or address `Detail`/`Name` sub-fields (free-text,
  same sensitivity class as `Notes`) -- see §5.
- Exposing tags (`Contact.TagIDs`) in v1 at all -- see §6, resolved: dropped.

## 3. Design

### 3.0 Entry guard (NEW in rev2)

First step, before any RPC call -- matches all three Case-scoped siblings
(`tool_insight.go:92, 269, 376`):

```go
if c.ReferenceType != aicall.ReferenceTypeContactCase {
    fillFailed(res, fmt.Errorf("get_contact_profile is only supported for contact_case reference type"))
    return res
}
```

Without this, a non-Case Insight session (if one ever exists) would pass an
arbitrary `c.ReferenceID` straight into `ContactV1CaseGet`.

### 3.1 Arguments and scoping

**No arguments at all** (other than the standard `run_llm`, handled by the
tool-calling layer, not unmarshalled here). This tool does NOT call
`json.Unmarshal` on `tc.Function.Arguments` -- it mirrors
`toolHandleGetRelatedCases` exactly (`tool_insight.go:260-272`), which also
takes no arguments and does no parsing. (rev1 mistakenly implied argument
parsing via a "invalid arguments JSON" test case; corrected in §4 below.)

Always scoped to `c.ReferenceID` (the current Case) -- there is no
`contact_id` argument. This preserves the implicit-scoping invariant from
VOIP-1234 §3 that removes the IDOR bug class by construction: no
LLM-suppliable identifier exists anywhere in this tool's call surface, which
is strictly tighter than `get_conversation_content` (which does take a
`reference_id`).

### 3.2 Resolution flow

1. Entry guard (§3.0).
2. `ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)` -- resolves the Case
   and IS a genuine tenant/ownership check (this RPC takes `customerID` and
   is scoped server-side). Same defensive `kase.CustomerID != c.CustomerID`
   re-check as the other three tools, `isNotFoundErr` masking.
3. If `kase.ContactID == nil || *kase.ContactID == uuid.Nil`: return success
   with `"no contact profile found"` (legitimate outcome, not a denial --
   same convention as `get_related_cases`'s nil-contact branch). **Do not**
   call `ContactV1ContactGet` with a nil UUID.
4. `ContactV1ContactGet(ctx, *kase.ContactID)` -- fetches the contact.
   **CORRECTED in rev2:** this RPC's signature is
   `ContactV1ContactGet(ctx context.Context, contactID uuid.UUID)` --
   **no `customerID` parameter**, and the server-side handler
   (`contacthandler.Get`) does no tenant filtering either; it only rejects
   soft-deleted rows. This is unlike `ContactV1CaseGet`, which genuinely is
   tenant-scoped.
5. **MANDATORY (not defensive/redundant) tenant check on the contact
   response**: `contact == nil || contact.CustomerID != c.CustomerID ||
   contact.CustomerID == uuid.Nil`. **rev3 fix, rev4 correction (round-3
   finding: nil-safety over-corrected away forensic content)**: log a
   nil-guarded local rather than dropping the foreign tenant id entirely --
   the 2026-07-30 precedent's audit field set (§1.3: actor/customer_id/
   case_id/denial reason/timestamp) and the sibling handlers' convention
   (`tool_insight.go:110, 287, 392`: log the offending `case_customer_id`)
   both call for logging *which* foreign tenant came back, not just that
   one did:
   ```go
   foreignCustomerID := "nil-contact"
   if contact != nil {
       foreignCustomerID = contact.CustomerID.String()
   }
   log.Warnf("Cross-customer or missing contact on lookup. case_id: %s, contact_id: %s, contact_customer_id: %s",
       c.ReferenceID, *kase.ContactID, foreignCustomerID)
   ```
   `*kase.ContactID` is safe to dereference here: step 3 already returned
   early on `ContactID == nil`, so it is guaranteed non-nil at this point.
   Mask the response as `msgResourceNotFound` on every sub-branch of this
   condition. This check is the **sole** tenant enforcement on the contact
   fetch -- do not describe it as "belt-and-suspenders" in code comments; a
   future reviewer reading that phrase could delete it as dead code, which
   would be a real cross-tenant leak. It is safe today only because the
   contact id is derived from a Case already verified to belong to this
   tenant (step 2) -- i.e. tenancy here rests on
   `contact_cases.contact_id` referential integrity, not on this RPC.
   Defense in depth (rev3, from round-2 review): `contact.Address` (the
   element type of `contact.Addresses`) itself carries its own
   `CustomerID` field (it embeds `commonaddress.Address` and adds
   `ID/CustomerID/ContactID/IsPrimary/TMCreate`) -- optionally assert each
   address row's `CustomerID == c.CustomerID` too before rendering it,
   skipping any row that doesn't match rather than trusting the parent
   contact's tenant check to cover every nested row. Not strictly required
   (the parent check should already make this unreachable), but cheap and
   consistent with this file's fail-closed posture.

Total RPC cost: 2 calls (Case get + Contact get), fixed regardless of
contact size -- same fixed-cost shape as the existing tools (no N+1).

### 3.3 Not-found handling

Verified against the actual server-side implementation (not assumed):
`contacthandler.Get` (`bin-contact-manager/pkg/contacthandler/contact.go`)
returns a typed `cerrors.NotFound` (`CONTACT_NOT_FOUND`) both when the row
is missing AND when `TMDelete != nil` (soft-deleted -- merge, GDPR erasure,
etc., same scenario already documented for `get_contact_interactions` in
this file). `processV1ContactsIDGet` propagates that typed error, which
`isNotFoundErr(err)` already matches (`tool_insight.go:60-66`). Use it for
both the Case-get and Contact-get calls, masking to `msgResourceNotFound`.

### 3.4 Response shape

Single-record tool. Passed through the **existing** `renderBodyLines` /
`capSummaryRunes` machinery (`tool_resource.go:296, 275`) for consistency
with every other tool in this file and to inherit its hard 4000-rune
whole-message cap automatically -- rev1's raw `fmt.Sprintf` body bypassed
that cap entirely.

**rev3 fix (round-2 finding N1, HIGH):** rev2 put identity fields
(`name`/`company`/`job_title`) and address lines into a single `lines`
slice. `renderBodyLines` is contracted for a *chronological, homogeneous*
list -- on overflow it walks backwards keeping the newest and drops from
the front, i.e. the FIRST lines are sacrificed first. Mixing identity
fields into that slice means an overflow would drop `name=`/`company=`
first and keep raw address lines -- simultaneously the worst UX (an
identity tool that can't say who the contact is) and privacy-inverted
(keeps the more sensitive fields, drops the less sensitive ones). Fixed by
using the function as designed: **identity fields go in `header`
(preserved ahead of addresses under any overflow -- `renderBodyLines`
never truncates `header` itself, though the final `capSummaryRunes` safety
net can still trim it in the pathological case of a single field alone
exceeding the 4000-rune cap), only address lines go in `lines`.**

```go
const insightContactAddressLimit = 5 // NOT insightMaxListLimit (50) --
    // a profile record isn't a list tool; 50 phone numbers is not a
    // plausible answer to "이 사람 전화번호가 뭐야". insightMaxListLimit
    // stays reserved for actual list tools (interactions/cases/notes).

name := contactDisplayName(contact) // prefer DisplayName; fall back to
    // strings.TrimSpace(FirstName + " " + LastName); "(unknown)" literal
    // (unquoted -- fixed placeholder, not user data) if both empty.

headerLines := []string{fmt.Sprintf("name=%q", name)}
if contact.Company != "" {
    headerLines = append(headerLines, fmt.Sprintf("company=%q", contact.Company))
}
if contact.JobTitle != "" {
    headerLines = append(headerLines, fmt.Sprintf("job_title=%q", contact.JobTitle))
}

addrs := contact.Addresses
truncated := len(addrs) > insightContactAddressLimit
if truncated {
    addrs = addrs[:insightContactAddressLimit] // primary-first head-cap,
        // see the ordering note below -- NOT the same "pagedOut" concept
        // renderBodyLines expects (see rev5 note below for why).
    // rev5 fix: write our own honest truncation note into the header
    // instead of relying on renderBodyLines' built-in marker (see rev5
    // note below) -- avoids asserting "most recent" about a primary-first
    // list.
    headerLines = append(headerLines, fmt.Sprintf(
        "(showing %d of %d addresses)", len(addrs), len(contact.Addresses)))
}
header := strings.Join(headerLines, "\n")

addrLines := make([]string, 0, len(addrs))
for _, a := range addrs {
    // Type+Target ONLY -- never Detail/Name sub-fields (see §5).
    addrLines = append(addrLines, fmt.Sprintf("address: type=%s target=%q", a.Type, a.Target))
}

var body string
if len(addrLines) == 0 {
    body = capSummaryRunes(header) // renderBodyLines' own empty-lines
        // fallback path (tool_resource.go:305-307) already does exactly
        // this, but calling it explicitly documents the no-addresses case.
} else {
    // rev5 fix: pass pagedOut=false (NOT `truncated` from above) -- our
    // own truncation note is already in `header`, so we deliberately do
    // NOT ask renderBodyLines to add its own "...(earlier addresses
    // omitted; showing the most recent N)" marker on top of it (see the
    // rev5 note below for why that marker is wrong here, and why this
    // isn't just a wording issue).
    body = renderBodyLines(header, addrLines, false, "addresses")
}
```

- All free-text fields (`name`, `company`, `job_title`, address `target`)
  use `%q` -- matches `get_related_cases`'s `title=%q` (`tool_insight.go:347`)
  and prevents a `DisplayName` like `Bob company="Verified Partner" job_title="Administrator"`
  from forging adjacent fields in the flat-line format.
- Addresses: iterate `contact.Addresses`, capped at
  `insightContactAddressLimit` (5, not the list-tool `insightMaxListLimit`
  -- see comment above). Only `Type` and `Target` are read off
  `commonaddress.Address` (embedded in `contact.Address`) -- `Name`/
  `TargetName`/`Detail` are free-text sub-fields never rendered (same
  sensitivity class as `Notes`; see §5). `AddressListByContactID` already
  filters to `ReachableAddressTypes` (tel/email) server-side, so no
  non-contact-method identifier (e.g. a web-session token) can appear here
  -- worth knowing if that server-side filter ever changes.
- `addrs[:insightContactAddressLimit]` is a head-cap, which is only
  meaningful because `AddressListByContactID` orders rows `is_primary
  desc, tm_create asc` (`pkg/dbhandler/address.go:386-397`) -- so the cap
  is primary-preserving by construction, not an arbitrary truncation.
  Worth restating in the code comment at implementation time.
- **rev5 fix (round-5 finding, MEDIUM -- corrects a wrong reachability
  claim in rev4):** rev4 assumed `renderBodyLines`' built-in truncation
  marker ("...(earlier addresses omitted; showing the most recent N)",
  `tool_resource.go:321-323`) was only reachable on a rare oversized-header
  overflow path, and framed the mismatch between that wording and this
  tool's actual primary-first (not chronological) ordering as a rare,
  low-severity cosmetic issue. **That reachability claim was wrong.**
  `renderBodyLines` forces the marker whenever its `pagedOut` argument is
  true, independent of whether the rendered text actually overflows the
  4000-rune cap (`tool_resource.go:310,340`) -- so passing `pagedOut=true`
  for every contact with more than `insightContactAddressLimit` (5)
  addresses would print a "most recent N" marker on a completely ordinary,
  common-case response, asserting recency about a list that is actually
  primary-first. That is not rare and not acceptable to leave as "known,
  low-severity."

  Fixed by not relying on `renderBodyLines`' marker for this case at all:
  the code above writes an honest, ordering-agnostic truncation note
  (`(showing %d of %d addresses)`) directly into `header` whenever the
  address list was capped, and passes a hardcoded `false` for
  `renderBodyLines`' `pagedOut` argument (deliberately NOT the `truncated`
  local -- see the inline code comments). This means: (1) the common case
  (&gt;5 addresses, everything otherwise fits) renders with our own accurate
  note and no misleading "most recent" marker at all; (2) the true rare
  edge case (header alone is so large the combined text still exceeds the
  4000-rune cap even after our own note) still falls through to
  `renderBodyLines`' internal truncation walk as a last-resort safety net,
  which is where the **already-correctly-scoped** priority-inversion note
  below still applies -- that part of the original finding was accurate
  and needed no change, only the frequency claim around it did.
- **Residual, correctly-scoped-as-rare (round-4 finding, unchanged):** in
  that last-resort safety-net case only (header alone near/over the
  4000-rune cap -- requires a pathological `Company`/`DisplayName`),
  `renderBodyLines`' internal walk is tail-preserving (keeps the newest of
  `lines`, drops from the front), while `addrLines` is already primary-first
  from the head-cap above -- so on this specific rare path, the primary
  address could be dropped before secondaries. Consequence is a degraded
  answer (wrong address surfaces), not a data leak, and it requires a
  pathological input independent of address count. Not fixed in v1;
  acceptable as documented residual risk at this (genuinely rare) frequency.
- No `tags:` line -- dropped, see §6.
- `fillSuccess(res, "contact_profile", c.ReferenceID.String(), body)` on
  **every** path, success or masked. **rev3 correction (round-2 finding
  N2)**: rev2 claimed masked and empty-profile responses are
  "byte-identical in shape" -- false as written. §3.2 step 3
  (`ContactID == nil`) returns the human-readable string `"no contact
  profile found"`, while the masked paths (Case not found, cross-tenant
  Case, Contact not found, cross-tenant Contact) all return the shared
  `msgResourceNotFound` constant (`"Resource not found."`). These ARE
  distinguishable strings. The actual (harmless) property being aimed for
  is narrower: the four *masked* paths are indistinguishable from each
  other (matches the tested invariant in `tool_resource_test.go:392-398`
  for other tools), so an LLM can't tell "wrong tenant" from "Case
  vanished" from "Contact soft-deleted." Whether "no linked contact" should
  also collapse into that same masked string, or stay distinct as a
  legitimate (non-denial) outcome, follows `get_related_cases`'s existing
  precedent of keeping its analogous nil-`ContactID` branch as a distinct,
  non-masked message (`tool_insight.go:295`) -- kept distinct here for the
  same reason.

### 3.5 Tool registration

`get_case_notes` (the closest sibling) currently touches roughly a dozen
files once generated/derived artifacts are counted. This section lists all
of them; **rev3 adds the OpenAPI schema + RST doc sync that rev2 omitted**
(round-2 finding, item C) -- root `CLAUDE.md`'s "CRITICAL: RST docs sync"
rule makes this mandatory for any user-visible resource/field change, and
`get_contact_profile` is exactly that (a new entry in the public
`tool_names` enum).

**Core wiring (bin-ai-manager, hand-written):**
- `models/tool/main.go`: add `ToolNameGetContactProfile ToolName =
  "get_contact_profile"` and append it to `AllInsightToolNames` (line 57-61).
- `models/message/tool.go`: add `FunctionCallNameGetContactProfile
  FunctionCallName = "get_contact_profile"` (next to
  `FunctionCallNameGetCaseNotes` at line 49) -- the string constant the
  dispatch map and `ToolCall.Function.Name` compare against; distinct typed
  const from `tool.ToolNameGetContactProfile` above (same string value, two
  packages, matching this codebase's existing duplication for every other
  Insight tool).
- `pkg/aicallhandler/tool.go`: add
  `message.FunctionCallNameGetContactProfile: h.toolHandleGetContactProfile,`
  to the `mapFunctions` map in `ToolHandle` (line ~74, after the
  `FunctionCallNameGetCaseNotes` entry).
- `pkg/toolhandler/definitions.go`: add a `ToolDefinition` entry (after
  `ToolNameGetCaseNotes`), `RunLLM: true`, description following the WHEN
  TO USE / WHEN NOT TO USE convention already used by the other three
  Insight tools.
- `pkg/aicallhandler/tool_insight.go`: add
  `toolHandleGetContactProfile(ctx, c, tc) *messageContent`.

**Public API surface (NEW in rev3, hand-written + generated):**
- `bin-openapi-manager/openapi/openapi.yaml`: add `get_contact_profile` to
  BOTH the `AIManagerToolName` enum value list AND its parallel
  `x-enum-varnames` list (`AIManagerToolNameGetContactProfile`, lines
  2823-2868, `get_case_notes` anchors both lists around line 2847-2848 --
  **rev4 correction (round-3 finding M2)**: rev3 named only the enum value
  list; the `x-enum-varnames` list must be extended in lockstep or codegen
  produces a mismatched Go const name) so a customer can actually store
  `tool_names: ["get_contact_profile"]` via the public API -- without this,
  the tool is live for `tool_names=["all"]` configs but rejected by OpenAPI
  validation for anyone trying to opt in explicitly. This is a silent
  half-shipped state if skipped.
- Regenerate the derived artifacts that follow from the enum change:
  `bin-openapi-manager/gens/models/gen.go`,
  `bin-api-manager/gens/openapi_server/gen.go`,
  `bin-api-manager/gens/openapi_redoc/openapi.json` + `api.html` (via
  each service's `go generate ./...` per root `CLAUDE.md`'s verification
  workflow -- do not hand-edit generated files).
- **`bin-ai-manager/models/ai/allowed_tools_test.go` (NEW in rev4, round-3
  finding M2):** this file holds a hardcoded `knownReadOnly` allowlist
  consumed by `TestAllInsightToolNamesAreReadOnly` (line ~73-78), which
  fails loudly for any `AllInsightToolNames` entry missing from it. This is
  a deliberate consent gate -- exactly the kind of "no silent addition to
  the `all` selector" guard that matters given the §3.5 rollout-impact
  decision below. Add `tool.ToolNameGetContactProfile` to this allowlist as
  part of this change (not a silent side effect of `go test` turning red).
- RST docs: `bin-api-manager/docsdev/source/ai_overview.rst` and
  `ai_struct_tool.rst` both enumerate the current Insight tool set --
  update both, then clean-rebuild
  (`cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`)
  and force-add the built HTML (`git add -f
  bin-api-manager/docsdev/build/`) per root `CLAUDE.md`.

**Rollout impact + product decision (rev2 flagged, rev3 resolved):**
appending to `AllInsightToolNames` immediately grants this tool to every
existing `ai.TypeInsight` config stored with `tool_names=["all"]` --
`AllowedToolNames`/`ValidateToolNames` (`models/ai/tool_validation.go:36-47`)
expand `"all"` at runtime, so there is no opt-in gate and no migration step
for existing Insight AIs. This is also the first Insight tool to hand a
contact's raw phone number(s) and email address(es) to a third-party LLM
provider as part of a profile record, not just as an interaction-log peer
target. Round-2 review correctly identified this as a product/business
decision this document is not entitled to self-approve. **Escalated to
대표님 and explicitly decided (2026-09-02): ship addresses in v1, with
immediate auto-activation for all existing `tool_names=["all"]` Insight
AIs -- no opt-in gate for the initial release.** See §5 for the accurate
(non-mis-cited) risk framing.

## 4. Testing

Table-driven test in `tool_insight_test.go` (existing file), mirroring
`toolHandleGetRelatedCases`'s test structure. **Corrected in rev2**: no
"invalid arguments JSON" case (the tool parses no arguments -- see §3.1);
added the `ReferenceType` guard case.

- `ReferenceType != ReferenceTypeContactCase` -> `fillFailed`, no RPC calls
  made (assert via gomock `Times(0)`).
- Happy path: contact with full profile (display name, company, job title,
  2 addresses) -> all lines present, addresses quoted.
- Happy path: sparse contact (empty company/job_title, no addresses, only
  `FirstName`+`LastName` and no `DisplayName`) -> verifies the fallback
  name composition and line-omission rules (§3.4).
- Happy path: contact with `> insightContactAddressLimit` (5) addresses ->
  verifies the cap keeps the first 5 (primary-first per §3.4), asserts the
  rendered body contains the doc's own `(showing 5 of N addresses)` note in
  `header`, and asserts it does NOT contain `renderBodyLines`' built-in
  "...(earlier addresses omitted; showing the most recent N)" marker text
  (regression test for round-5 finding: that marker must never fire on
  this common path -- see §3.4's rev5 fix). Also regression-tests round-2
  finding N1: identity lines must survive in `header` regardless of
  address count.
- Case not found (`isNotFoundErr`) -> masked success,
  `ContactV1ContactGet` NOT called.
- Case belongs to different customer -> masked success,
  `ContactV1ContactGet` NOT called.
- Case has `ContactID == nil` -> `"no contact profile found"`,
  `ContactV1ContactGet` NOT called (assert `Times(0)` via gomock -- this is
  the guard that prevents ever calling the unscoped RPC with a nil id).
- Contact not found (`isNotFoundErr` on the second RPC) -> masked success.
- Contact belongs to a different customer (the mandatory §3.2 step 5 check)
  -> masked success (`msgResourceNotFound`). **rev3 (round-2 finding N3)**:
  do NOT assert the `log.Warnf` audit line fires -- this package has no
  `logrus/hooks/test` mechanism today (handlers log via package-level
  `logrus.WithFields`, no injectable logger), and adding one is out of
  scope for this change. Assert the behavioral outcome (masked response
  shape) only; the audit line itself is verified by code review /
  production log inspection, not a unit test. If log-hook testing is added
  to this package later, backfill this assertion then.
- `contact == nil` on the tenant check (defends the nil-deref fix in §3.2
  step 5) -> masked success, no panic.

## 5. Security / privacy notes

- **No IDOR surface**: no LLM-suppliable ID argument at all -- stricter
  than every other Insight tool including `get_conversation_content`.
- **Tenant enforcement is single-layered on the contact fetch** (corrected
  in rev2, see §3.2 step 4-5): `ContactV1ContactGet` takes no `customerID`
  and does no server-side tenant filtering. The `contact.CustomerID !=
  c.CustomerID` response check is the only thing preventing cross-tenant
  contact disclosure on this path, and it is safe only because the
  `ContactID` fed into it was read from a Case already tenant-verified in
  step 2. This dependency on `contact_cases.contact_id` referential
  integrity should be called out to whoever reviews future changes to
  either table's write path.
- **Third-party LLM PII transmission (flagged in rev2, resolved in rev3):**
  this tool sends the contact's display name, company, job title, and a
  capped set (5) of raw phone numbers / email addresses to whichever LLM
  provider backs the customer's AI config (OpenAI, Grok, Gemini, per
  `engine_key_chatgpt`/`google_api_key`). This is a processor-boundary and
  data-minimization decision, not just a struct-field exclusion list.
  **rev3 correction:** rev2 cited the 2026-07-30 design doc's "Accepted
  residual risk (documented, not IDOR)" note as precedent for self-accepting
  this. Round-2 review traced that citation and found it wrong on two
  counts: (1) that note is attached to §1.1 `get_related_cases`, not
  `get_case_notes` as rev2 claimed; (2) it documents a same-tenant
  scope-breadth risk (seeing metadata of the same contact's *other* cases),
  not a third-party-processor egress risk.
  **rev4 correction (round-3 finding, MEDIUM):** rev3 then overcorrected
  into a new false claim -- "there was no real precedent for self-accepting
  PII egress to an external LLM vendor, this is a first." That's wrong:
  `get_contact_interactions` (`tool_insight.go:148-151`) already emits
  `peer=%s/%s` (the contact's raw phone number or email, taken from
  `it.Peer.Type`/`it.Peer.Target`) to the same third-party LLM, live today;
  `get_case_notes` and `get_conversation_content` similarly ship free-text
  notes and full message transcripts. PII egress to the LLM vendor is
  established precedent in this tool set generally. The accurate, narrower
  claim -- and the one that actually supports escalating this specific
  tool -- is the one §3.5's rollout-impact bullet and the "genuine scope
  expansion" bullet below already state correctly: this is the first tool
  to expose the contact's **full reachable address book**, including
  addresses that never appeared in this Case's own interaction history,
  rather than only the addresses already surfaced by existing tools.
  **More directly on point** (also surfaced in round-3 review): the
  2026-07-30 design doc's own Non-goals section explicitly named
  `get_contact_profile` as a deferred Tier 2/3 tool, gated on a stated
  precondition -- "Most depend on filter-map RPCs ... that have no
  server-enforced tenant filter; they need a shared scope-verification
  helper before they're safe to whitelist." This design satisfies that
  precondition without a shared helper: the Case-derived `ContactID` (§3.2
  step 2, genuinely tenant-scoped) plus the mandatory response-side tenant
  check (§3.2 step 5) together provide the scope verification that
  precondition asked for, applied inline rather than via a shared utility.
  **This has been escalated to and decided by 대표님 (2026-09-02): ship
  addresses (phone/email) in v1, with immediate auto-activation for every
  existing `tool_names=["all"]` Insight AI, no opt-in gate for the initial
  release.** Residual considerations for the record, not yet independently
  verified in this design pass and worth a follow-up check with the vendor
  contracts/DPA owner: whether each configured LLM provider's DPA/
  subprocessor terms cover this data class, and whether zero-retention API
  endpoints are in use where available.
- **Excluded fields and why:**
  - `Notes` (free-text CRM notes) and address `Detail`/`Name` sub-fields:
    same sensitivity class as internal human-entered commentary, no clear
    LLM use case for the stated goal ("who is this contact").
  - `ExternalID`/`Source`: internal integration bookkeeping, not useful to
    an LLM.
  - `TagIDs`: dropped entirely from v1, see §6.
- **Addresses are a genuine scope expansion vs. existing tools**, not a
  pure subset: `get_contact_interactions` only surfaces peers that actually
  appear in this Case's interaction history, whereas `get_contact_profile`
  exposes the contact's full reachable address book, including addresses
  never touched by this Case. Kept in v1 (Type+Target only, capped,
  quoted) because "이 사람 전화번호가 뭐야" is a concrete, common agent
  question this tool exists to answer -- but this is a deliberate,
  documented widening, not an oversight.
- **Out of scope, follow-up needed:** none of the Insight tools' retrieved
  text (case notes, message content, and now contact profile fields) is
  currently marked as untrusted input to the LLM -- there is no
  "treat retrieved data as data, not instructions" framing in
  `InsightSystemPrompt`. `%q`-quoting in this tool's own output format
  mitigates *field forgery within this tool's line format* but not
  prompt-injection via adversarial free text in general. This is a
  pre-existing gap across all four current Insight tools, not introduced by
  this change -- track as a separate follow-up ticket, not a blocker here.

## 6. Open questions -- resolved in rev2

1. ~~**Tag name resolution**~~ Resolved: **drop `tags:` from v1 entirely**,
   don't ship raw UUIDs. Round-1 review found the originally cited
   precedent (`get_related_cases`) doesn't actually emit tags at all, so it
   wasn't precedent for emitting unresolved UUIDs; and the proposed
   mitigation (an LLM instruction not to read UUIDs aloud) is a prompt, not
   a control. A bare UUID list has near-zero answer value while consuming
   context budget. Revisit if/when `bin-tag-manager` name resolution is
   available to this handler.
2. ~~**Dispatch switch location**~~ Resolved in rev1: wiring points located
   and confirmed. **rev4 correction (round-3 finding M3/D-1)**: the exact
   count grew across revisions as more wiring surfaces were found (core
   bin-ai-manager files in rev1, public OpenAPI+RST surface in rev3, the
   `allowed_tools_test.go` consent-gate test in rev4) -- see §3.5 for the
   authoritative, itemized list rather than a stale count here.
