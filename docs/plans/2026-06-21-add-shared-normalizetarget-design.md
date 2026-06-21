# Shared Address NormalizeTarget and Canonical-Form Enforcement

- Issue: voipbin/monorepo#1002
- Class: hardening / refactor (extract + promote an existing primitive; behavior-preserving for tel/email on ASCII inputs — non-ASCII digit handling deliberately tightened, see §3.1). Phase 2 (this revision) extends adoption to all channel resource managers.
- Date: 2026-06-21

## 0. Revision note (v4 — scope expansion)

v1-v3 delivered the shared primitive + contact-manager adoption only, deferring
the other channel managers. pchero directed (2026-06) that NormalizeTarget become
a system-wide core principle: EVERY channel resource manager that stores a
source/destination address must canonicalize it through the single shared
authority, and test coverage must be maximized. This revision adds call /
message / conversation / email-manager adoption at each manager's storage
chokepoint, verified safe against every send/dispatch path.

The earlier "defer because no contact_id yet" rationale was WRONG and is
retracted: normalization is independent of contact resolution. Canonicalizing
the STORED address has immediate value (data hygiene + consistent identity form)
and must happen as early as possible, because any address persisted in raw form
before normalization is introduced becomes a permanent mismatch.

## 1. Problem Statement

Identity resolution fragments across channels because there is no single
normalization authority.

Code-verified facts:

- `bin-common-handler/models/address/validate.go`: `ValidateTarget(type, target)`
  VALIDATES only, it does NOT normalize. The tel rule is the regex
  `^\+[0-9]{7,15}$`, so different-but-valid forms (`+155****4567` vs a
  space/punctuation-laden raw form) either both pass as distinct strings or the
  raw form fails outright before it can be canonicalized.
- email validation uses `net/mail.ParseAddress` (RFC 5322) which preserves case,
  so `John@Example.COM` and `john@example.com` both pass but differ as strings.
- The real normalizer historically lived ONLY inside `bin-contact-manager`.
- `sip` / `line` / `whatsapp` / `ai` / `ai_team` have NO normalizer anywhere.
- `whatsapp` (`TypeWhatsApp`) exists as an enum but had NO branch in
  `ValidateTarget` — it fell through to `default` and ERRORED. Exposed-but-broken.
- Every channel resource (Call.Source/Destination, Message source + targets,
  Conversation Self/Peer, Email source/destinations) is a `commonaddress.Address`
  (Type + Target), but NONE of call/message/conversation/email-manager normalizes
  the address it persists. They store raw, untrusted, customer-supplied strings.

Why it matters: two managers that store the same identifier in different forms
produce distinct `target` values, so a future `LookupByAddress` MISS,
`contact_id = NULL`, and a permanently fragmented Contact timeline. The canonical
form must be enforced at every write boundary, not just contact-manager's.

## 2. Scope

In scope:

1. New `bin-common-handler/models/address/NormalizeTarget(addressType Type,
   target string) (string, error)` — the single canonicalization authority.
   (Delivered in v1-v3, retained.)
2. `whatsapp` promoted to a SUPPORTED channel in `ValidateTarget` (reuse tel
   E.164 rule) + `NormalizeTarget` (reuse tel normalization). (Delivered.)
3. contact-manager: write + `LookupByPhone`/`LookupByEmail` both pass through
   `NormalizeTarget`; `LookupByPhone` input-normalization bug fixed. (Delivered.)
4. **NEW: each channel resource manager canonicalizes the source/destination
   address it stores, through the shared `NormalizeTarget`, at its creation
   chokepoint:**
   - **call-manager**: `callHandler.Create` (`pkg/callhandler/db.go`), normalize
     `*source` and `*destination` before the struct literal.
   - **message-manager**: at the resource entry — `messagehandler.Send` (outbound)
     and `messagehandler.hookTelnyx` (inbound) — normalize `source` + each
     `target.Destination` so BOTH the persisted Message and the provider wire
     value are canonical.
   - **conversation-manager**: `conversationHandler.GetOrCreateBySelfAndPeer`
     (normalize `self`/`peer` BEFORE the dedup lookup so the lookup key and the
     stored value are the same canonical form) and `conversationHandler.Create`
     (covers the API-POST and line/whatsapp RPC-create paths).
   - **email-manager**: `emailHandler.Create` destination loop, normalize each
     destination `Target` BEFORE validation (Normalize-then-Validate).
5. Golden-vector + idempotency contract tests for `NormalizeTarget` (delivered),
   plus a per-manager regression test asserting the stored/forwarded address is
   the canonical form, plus expanded golden vectors (see §4).

Out of scope:

- The CRM read model itself (`contact_interactions`, timeline, dedup). This issue
  delivers the normalization primitive + its enforcement; the read model is later.
- `line` / `ai` / `ai_team` normalization beyond identity. `line` and the
  UUID/opaque-id types canonicalize to themselves (identity) so they remain
  idempotent and never error. `ai`/`ai_team` validation is a separate concern
  (see §3.1 asymmetry note).
- `bin-agent-manager` address normalization. agent-manager DOES persist a
  `commonaddress.Address` source/destination and looks it up via exact-match
  (`AgentGetByCustomerIDAndAddress`, `json_contains`). It is deliberately NOT
  normalized in this PR because the matching call-manager lookup
  (`getAddressOwner` -> `AgentV1AgentGetByCustomerIDAndAddress`) runs on the RAW
  pre-`Create` value by design (§3.3). Normalizing the agent STORE in isolation
  would create a stored-vs-lookup-key mismatch and break the exact-match join.
  Closing it correctly requires normalizing BOTH the agent store and the
  call-manager agent lookup together (a larger, coordinated change), so it is
  deferred to a dedicated follow-up rather than half-done here.
- `bin-queue-manager` (queuecall source) and `bin-campaign-manager`
  (outplan/campaigncall source/destination). These persist DERIVED copies that
  ultimately flow into `callHandler.Create`, which re-normalizes at the canonical
  chokepoint, so the canonical-form invariant holds at dispatch regardless. No
  isolated normalization needed; covered transitively.

## 3. Design

### 3.1 NormalizeTarget contract (revised v5 — loss-proof + 2 sentinel errors)

```go
// Sentinel errors (callers may errors.Is on them):
var (
	// ErrUnknownType: addressType is not a known enum value (almost always a
	// programming error — a misconfigured or empty Type at a call site).
	ErrUnknownType = errors.New("unknown address type")
	// ErrNotNormalizable: the type is known but the value cannot be reduced to a
	// canonical form for it (a legitimate domain case, e.g. a tel carrier
	// sentinel like "anonymous"/"Restricted" that contains no digits). The
	// ORIGINAL value is returned unchanged; the caller may store it verbatim.
	ErrNotNormalizable = errors.New("value not normalizable for type")
)

func NormalizeTarget(addressType Type, target string) (string, error)
```

**Loss-proof contract (the core principle):** the FIRST return value is ALWAYS
a value safe to store. If the input can be canonicalized, it is the canonical
form with `err == nil`. If it cannot, the ORIGINAL input is returned unchanged
with a non-nil sentinel error. `NormalizeTarget` NEVER blanks a MEANINGFUL
non-empty input (the sole "empty out" case is whitespace/empty input to the
lossless email/sip trim, which is not data loss). A caller that discards the
error (`out, _ := NormalizeTarget(...)`) therefore never loses real data; a
caller that checks the error learns the value is not in canonical form.

Per-type rules:

| Type | Normalization | Error |
|------|---------------|-------|
| `tel` | trim; keep `+` only at index 0; keep ASCII `[0-9]` only. **If the result contains NO digit** (e.g. `anonymous`, `Restricted`, `+`, empty), return the ORIGINAL input unchanged. | `nil` if ≥1 digit; `ErrNotNormalizable` if no digit |
| `whatsapp` | identical to `tel` | same as tel |
| `email` | `strings.ToLower(strings.TrimSpace(target))` | always `nil` (lossless) |
| `sip` | trim; params boundary first `;`/`?`; split pre-boundary on LAST `@`; preserve user; lowercase host token; preserve params tail; no `@` → trimmed input unchanged | always `nil` (lossless) |
| `none`, `agent`, `conference`, `extension`, `line`, `ai`, `ai_team` | identity (input UNCHANGED, no trim) | always `nil` |
| unknown | return the ORIGINAL `target` unchanged (NOT empty) | `ErrUnknownType` |

Rationale for the tel no-digit rule (F2 fix): a raw tel `tel` Target may legitimately
carry a non-numeric carrier sentinel for withheld/anonymous caller-ID
(`anonymous`, `Restricted`, `unknown`). The old "strip to digits" rule blanked
these to `""`, silently destroying telecom semantics on the STORED source. The
no-digit guard preserves them verbatim and flags `ErrNotNormalizable` so the
value is never canonicalized into nothing.

Idempotency (value AND error are deterministic):
`NormalizeTarget(t, NormalizeTarget(t, x).0)` returns the same value and the same
error class as `NormalizeTarget(t, x)`. E.g. `("anonymous", ErrNotNormalizable)`
on both passes; `("+155****4567", nil)` on both passes.

ASCII-only digit rule (not `unicode.IsDigit`): the tel validator regex is ASCII
`^\+[0-9]{7,15}$`. Pin ASCII `[0-9]`. Deliberate, documented, test-locked.

`none` consistency: `ValidateTarget(TypeNone, _)` returns `nil`, so `none` is a
no-error identity in BOTH functions. `ai` / `ai_team`: `NormalizeTarget` identity
(no error), `ValidateTarget` errors via `default` (pre-existing, out of scope).

### 3.2 Ordering: Normalize THEN Validate

```
canonical, err := NormalizeTarget(type, raw)   // 1. canonicalize
if err != nil { ... }
if err := ValidateTarget(type, canonical); err != nil { ... }  // 2. validate canonical
// store / dispatch / lookup with `canonical`
```

A raw tel like `+1 (555) 123-4567` only passes the regex AFTER normalization
strips punctuation. This ordering is invariant.

### 3.3 Per-manager wiring

Each manager normalizes at its own storage chokepoint (each resource owns its
normalization, consistent with VoIPBin layering). NormalizeTarget is idempotent,
so a value already canonical passes through unchanged.

**Error-handling policy at every call site:** because NormalizeTarget is
loss-proof (returns the original value on any error), call sites take the first
return value unconditionally and discard the error with `_`. The value is always
safe to store. (Rationale: `ErrNotNormalizable` is a normal domain case like a
tel carrier sentinel, and `ErrUnknownType` cannot occur here because every call
site passes a known constant or the resource's own validated `.Type`.) Sites
that want observability MAY log on `ErrNotNormalizable`, but no site should
branch on it for control flow.

**Nil-pointer policy (F1):** several models hold the address by pointer
(`Message.Source *commonaddress.Address`, call `*source`/`*destination`). Every
call site MUST nil-guard before dereferencing: `if addr != nil { addr.Target, _ =
NormalizeTarget(addr.Type, addr.Target) }`.

**Value-slice policy (F7):** destinations held as a value slice
(`message.Send` `[]commonaddress.Address`, email-manager `Destinations`) MUST be
normalized BY INDEX (`destinations[i].Target, _ = NormalizeTarget(...)`), NEVER
via `for _, d := range` (which mutates a copy and silently discards the result —
a silent no-op that would skip normalization entirely).

**ValidateTarget boundary (F6):** the normalize call sites here are storage
write paths; they do NOT feed the normalized value into `ValidateTarget`. This
matters because a loss-proof non-normalizable value (`anonymous`, bare `+`) and
`whatsapp`/`ai`/`ai_team` targets would FAIL `ValidateTarget` (the regex rejects
`anonymous`; `ValidateTarget` has no whatsapp/ai/ai_team case). The
Normalize-then-Validate pipeline (§3.2) applies to USER-SUPPLIED OUTBOUND
addresses validated at their own boundary, not to these stored-address
canonicalization sites. No v5 site introduces a Normalize→Validate path on a
non-normalizable or whatsapp value.

**contact-manager** (delivered v1-v3): `normalizeE164` delegates to
`NormalizeTarget(TypeTel, ...)`; email Create/AddEmail/UpdateEmail/LookupByEmail
route through `NormalizeTarget(TypeEmail, ...)`; `LookupByPhone` normalizes its
input. (The loss-proof revision only strengthens these — a contact phone that is
all non-digit now stays verbatim instead of blanking.)

**call-manager**: `callHandler.Create` (`pkg/callhandler/db.go:30`) is the SOLE
writer of `Source`/`Destination`, reached by both `CreateCallOutgoing`
(outgoing_call.go:294, also every groupcall leg via RPC) and `startCallTypeFlow`
(start.go:623, incoming). Nil-guard then normalize `source.Target` and
`destination.Target` (keyed on each address's own `.Type`) at the top of `Create`
before the struct literal at db.go:99-100.

- Send-path safety (verified): outgoing dial reads the PERSISTED destination
  (`c.Destination.Target`) AFTER `Create`, so the dial string gets the canonical
  form. tel goes into the provider SIP user part where E.164 is already assumed;
  sip lowercases only the host while preserving `;transport=ws`/`;outbound_proxy=`
  params. SAFE.
- F2 protection: an incoming PSTN call with a withheld caller-ID
  (source.Target = `anonymous` / `Restricted`) is preserved verbatim by the
  loss-proof tel rule — the stored source keeps its telecom sentinel.
- MUST NOT normalize in `channelhandler/address.go` or before the incoming
  number-manager / trunk / extension lookups in `start*.go` — those lookups run
  on the raw Asterisk value BEFORE `Create`. `getAddressOwner` and `getDialroutes`
  also resolve on the raw pre-Create value (best-effort, errors only logged); the
  stored canonical form differs from those lookup keys by design.

**groupcall (call-manager, F5-codeverify gap):** `groupcallHandler.Create`
(`pkg/groupcallhandler/db.go:68-69`) ALSO persists `Source`/`Destinations` on the
Groupcall record. Per the "EVERY resource that stores an address canonicalizes
it" principle, normalize `source` + each `destinations[i]` there too (nil-guard,
keyed on `.Type`). Each dialed leg already re-normalizes at `callHandler.Create`,
so routing is unaffected; this closes the stored-raw-address gap on the Groupcall
record itself.

**message-manager**: Message has `Source *commonaddress.Address` and
`Targets []target.Target` where each `Target.Destination` is a
`commonaddress.Address`. Normalize at the resource entry so BOTH storage and the
provider wire are canonical:
- `messagehandler.Send` (send.go): nil-guard + normalize `source`, normalize each
  `destinations[i]` before building targets (send.go:~34).
- `messagehandler.hookTelnyx` (hook.go): normalize the inbound source/targets
  before `Create` (hook.go:~86).
- Send-path safety (verified): providers (Telnyx/MessageBird) take
  `destination.Target` verbatim as the `to`/`from` and EXPECT E.164. Normalizing
  tel to `+`+digits yields exactly E.164. SAFE, and fixes punctuated-input rejects.
- F3 protection: an alphanumeric SMS sender ID submitted as a `tel` source (e.g.
  `from = "VOIPBIN"`) has no digits, so the loss-proof tel rule preserves it
  verbatim instead of blanking the `from`. (Implementation MUST confirm whether
  alphanumeric senders are modeled as `TypeTel`; if they are a distinct type the
  point is moot, but the loss-proof guard protects either way.)
- MUST NOT normalize inside the provider handlers, `requestexternal/*`, or the
  async goroutine (data-race risk on shared slices).

**conversation-manager**: Conversation has `Self`/`Peer commonaddress.Address`.
The conversation Message model has NO address fields, so nothing to normalize
there.
- `conversationHandler.GetOrCreateBySelfAndPeer` (db.go:~56): normalize
  `self`/`peer` BEFORE the `ConversationGetBySelfAndPeer` dedup lookup, so the
  lookup key and the stored value share one canonical form.
- `conversationHandler.Create` (db.go:~113): normalize `self`/`peer` before the
  struct literal — covers the API-POST and line/whatsapp RPC-create paths.
- F5 dedup-consistency (verified against code): SMS dispatch uses `Self`/`Peer`
  (tel); LINE/WhatsApp dispatch + dedup use `conversation.DialogID`, NOT
  Self/Peer. `line` Self/Peer are `TypeLine` → identity normalization (unchanged).
  For `whatsapp`, the inbound hook (`whatsapphandler/hook.go:209-211`) ALWAYS sets
  `peer.Target = waID` (Meta's `+`-less form) and dedups on `DialogID = waID`, so
  tel normalization (which injects no `+`) leaves the hook path's peer unchanged
  and consistent with DialogID. The ONLY path that can supply a `+`-prefixed
  whatsapp peer is the customer-supplied API POST (`v1_conversations.go:107-110`,
  `req.Peer`), which is a PRE-EXISTING ambiguity: that path keys dedup on its own
  supplied DialogID, independent of peer normalization. Normalizing peer does NOT
  create a new split — it cannot make two waID-hook conversations diverge (they
  share the `+`-less form). Conclusion: normalization introduces NO new dedup bug;
  the API-POST `+`-vs-waID ambiguity predates this change and is out of scope.
  A regression test asserts the waID-hook peer normalizes to itself (idempotent,
  no `+` injected).
- MUST NOT normalize `conversation.DialogID` — provider wire target + LINE/WhatsApp
  dedup key; leave verbatim.

**email-manager**: Email has `Source *commonaddress.Address` and
`Destinations []commonaddress.Address` (all `TypeEmail`).
- `emailHandler.Create` (email.go:~31): normalize each `destinations[i].Target`
  (by index, in place) BEFORE the existing `validateEmailAddress` check. Optionally
  normalize the source defensively in `create` (today source = canonical constant).
- Send-path safety (verified): SendGrid/Mailgun take the bare `.Target` verbatim;
  email normalization is lossless (always `nil` error). SAFE.
- MUST NOT pass the composite `"Name <addr>"` string to NormalizeTarget — normalize
  only `.Target`, preserve `.TargetName`. MUST NOT normalize inside engine send funcs.

### 3.4 Affected files

| Service | File | Change |
|---------|------|--------|
| bin-common-handler | `models/address/normalize.go` (new, delivered) | `NormalizeTarget` + helpers |
| bin-common-handler | `models/address/normalize_test.go` (new, delivered) | golden-vector + idempotency |
| bin-common-handler | `models/address/validate.go` (delivered) | whatsapp branch |
| bin-common-handler | `models/address/validate_test.go` (delivered) | whatsapp rows |
| bin-contact-manager | `pkg/contacthandler/contact.go` (delivered) | delegate + lookups |
| bin-contact-manager | `pkg/contacthandler/contact_test.go` (delivered) | LookupByPhone test |
| bin-call-manager | `pkg/callhandler/db.go` | normalize source/destination in `Create` |
| bin-call-manager | `pkg/callhandler/db_test.go` | normalization regression test |
| bin-call-manager | `pkg/groupcallhandler/db.go` | normalize source/destinations in `Create` |
| bin-call-manager | `pkg/groupcallhandler/db_test.go` | normalization regression test |
| bin-message-manager | `pkg/messagehandler/send.go` | normalize source/destinations in `Send` |
| bin-message-manager | `pkg/messagehandler/hook.go` | normalize inbound source/targets |
| bin-message-manager | `pkg/messagehandler/*_test.go` | regression tests |
| bin-conversation-manager | `pkg/conversationhandler/db.go` | normalize self/peer in GetOrCreate + Create |
| bin-conversation-manager | `pkg/conversationhandler/db_test.go` | regression tests |
| bin-email-manager | `pkg/emailhandler/email.go` | normalize destinations before validate |
| bin-email-manager | `pkg/emailhandler/email_test.go` | regression test |

## 4. Test Strategy (maximized — pchero directive)

This normalizer is a core system principle; coverage must be exhaustive.

### 4.1 Shared golden-vector table (`normalize_test.go`)

Expand the table to cover every branch and adversarial edge, each row also run
through the idempotency double-apply assertion:

- tel: punctuation, dashes+spaces, leading-ws+`+`, `+` not at index 0, double
  `+`, Arabic-Indic digit stripped, full-width digit stripped, **letters-only
  (`anonymous`) → ORIGINAL unchanged + `ErrNotNormalizable`**, **lone `+` →
  original + `ErrNotNormalizable`**, **empty → empty + `ErrNotNormalizable`**,
  internal `+` mid-string, tab/newline whitespace, very long digit run,
  already-canonical (idempotent, `nil`).
- whatsapp: spaces, no-`+` waID form (digits preserved, no `+` injected, `nil`),
  alphanumeric-only (`VOIPBIN`) → original + `ErrNotNormalizable`, empty,
  already-canonical.
- email: trim+lower, mixed-case domain, display-name form, leading/trailing ws,
  already-lower (idempotent), uppercase local-part. All `nil` error.
- sip: host lower, with port, `;transport` param preserved, `;maddr` value-case
  preserved, userinfo `user:pass@host`, IPv6 `[2001:DB8::1]:5060`, `@`-inside-
  header, whole-input trim, no-`@`, multiple `@` before boundary, empty,
  already-canonical. All `nil` error.
- identity types: line/agent/ai/ai_team/conference/extension/none all return
  input unchanged (no trim), idempotent, `nil` error.
- unknown type → ORIGINAL input unchanged + `ErrUnknownType` (NOT empty). Assert
  via `errors.Is(err, address.ErrUnknownType)`.
- Error-class idempotency: a second NormalizeTarget on the first result returns
  the SAME error class (both passes `ErrNotNormalizable`, or both `nil`).
- Loss-proof property test: for every row, assert the result is either the
  canonical form OR byte-equal to the input — NEVER an empty string for a
  non-empty input.

### 4.2 Per-manager regression tests

- **contact-manager** (delivered): `Test_LookupByPhone_NormalizesInput`,
  `Test_LookupByEmail_NormalizesEmail`.
- **call-manager**: in `db_test.go`, assert `Create` persists the canonical
  Source/Destination — pass a punctuated tel destination + a mixed-case sip, and
  a mixed-case email source, assert the stored `Source.Target`/`Destination.Target`
  via the `CallCreate` gomock matcher. Add a row proving an already-canonical
  input is unchanged (idempotency at the boundary).
- **message-manager**: assert `Send` forwards the canonical destination to the
  `MessageV1...`/provider call AND persists canonical targets; assert
  `hookTelnyx` persists canonical inbound source/targets.
- **conversation-manager**: assert `GetOrCreateBySelfAndPeer` performs the dedup
  lookup with the CANONICAL self/peer (gomock arg match on
  `ConversationGetBySelfAndPeer`), and that `Create` stores canonical self/peer;
  assert `DialogID` is NOT altered.
- **email-manager**: assert `Create` validates+stores canonical (lowercased)
  destinations and preserves `TargetName`.

All per-manager tests use strict gomock arg matchers on the persist/forward call
so a regression in the normalization wiring fails the test.

## 5. Verification

Per CLAUDE.md, every touched service runs the full workflow before commit:
`go mod tidy && go mod vendor && go generate ./... && go test ./... &&
golangci-lint run -v --timeout 5m`. Touched services: bin-common-handler,
bin-contact-manager, bin-call-manager, bin-message-manager,
bin-conversation-manager, bin-email-manager.

bin-common-handler is consumed by 37 services; the `NormalizeTarget` addition is
purely additive. The `whatsapp` `ValidateTarget` branch is a behavior change
(error->pass) for whatsapp inputs only, with no production caller (grep-verified:
only the address package's own tests call ValidateTarget). A
`for d in bin-*/; do (cd "$d" && go build ./...); done` sweep confirms no
compile break.

## 6. Sections marked N/A (hardening-class)

Domain model / DB schema / new REST API / webhook events / flow variables /
RabbitMQ actions / Prometheus metrics / PII-LLM: N/A. This issue adds no entity,
table, endpoint, event, or external call. It extracts/promotes a pure-function
primitive, fixes one lookup bug, and inserts canonicalization at existing
storage boundaries.

## 7. Open Questions

| # | Question | Recommendation | Owner |
|---|----------|----------------|-------|
| 1 | message-manager: normalize at `Send` entry (canonical reaches provider too) vs only in `Create` (provider gets raw)? | Normalize at `Send`/`hook` entry — normalized tel = E.164 = what providers want, so it also fixes punctuated-input provider rejections. Safe. | CTO |
| 2 | call-manager: normalize at `Create` chokepoint vs per-path entry? | `Create` chokepoint — sole writer, dial reads persisted value, incoming lookups stay upstream on raw input. | CTO |
| 3 | Should normalization failure (unknown type) hard-fail the create, or fall back to raw? | For the known channel types in scope, NormalizeTarget never errors. Use the canonical value; on the impossible error path, keep the raw target and log (do not silently drop). | CPO |

## 8. Review Summary

### Round 1-2 (v1-v3, primitive + contact-manager) — both reviewers APPROVE
(See git history of this doc.) tel order pinned, sip narrowed to host token,
ASCII-digit pin documented, whatsapp behavior-change disclosed, ai/ai_team
asymmetry documented.

### v4 scope expansion (this revision)
Recon (3 file-access subagents) mapped the storage chokepoint and send-path
safety for call/message/conversation/email-manager; the highest-risk claim
(call-manager `Create` is the sole Source/Destination writer; incoming routing
lookups run upstream of `Create`) was re-verified directly against db.go.

### v4 review round (2 reviewers: 1 code-verification + 1 adversarial)
Code-verification reviewer: APPROVE, all per-manager insertion points + send-path
safety claims TRUE against the repo, baselines green. Flagged the groupcall
stored-address gap (Finding 5).
Adversarial reviewer: CHANGES REQUESTED with HIGH-severity data-integrity issues.

### v5 (this revision) — findings applied
- (F1, HIGH) Nil-pointer: every call site nil-guards the pointer address before
  deref. Documented as an explicit policy in §3.3.
- (F2, HIGH) tel normalization corrupting carrier sentinels (`anonymous`):
  NormalizeTarget is now LOSS-PROOF — tel/whatsapp with no digit returns the
  ORIGINAL value + `ErrNotNormalizable`. Never blanks a non-empty input.
- (F3, MEDIUM) Alphanumeric SMS sender ID: protected by the same loss-proof tel
  rule (no-digit → preserved). Code-verified that message-manager source `.Type`
  is caller-supplied, so loss-proof guards either modeling.
- (F4, MEDIUM) Discarded-error data loss: contract changed so unknown type returns
  the ORIGINAL value (not `""`) + `ErrUnknownType`. Two sentinel errors
  (`ErrUnknownType`, `ErrNotNormalizable`) per pchero decision, `errors.Is`-able.
- (F5, MEDIUM/code-verify) conversation whatsapp peer dedup: code-verified the
  hook always uses the `+`-less waID and dedups on DialogID; normalization
  introduces NO new split. The API-POST `+`-ambiguity predates this change and is
  out of scope. groupcall stored-address gap: closed by also normalizing in
  `groupcallHandler.Create`.
- Error-handling policy: every call site discards the error (`_`) and stores the
  loss-proof first value; sites MAY log on `ErrNotNormalizable`. Documented.

Contract change note: the loss-proof + sentinel-error revision also strengthens
the already-delivered contact-manager code (a non-digit phone now stays verbatim
instead of blanking) and requires updating the v1-v3 `normalize.go` +
`normalize_test.go` to the new contract before this lands.

A fresh design re-review round on v5 follows (mandatory after the contract change).

### v5 design re-review — APPROVE
Fresh adversarial reviewer confirmed all 5 prior findings resolved by the
loss-proof + sentinel-error contract, no new contradiction. Applied 3 advisory
notes: value-slice by-index policy (F7), ValidateTarget boundary note (F6),
loss-proof wording for whitespace-only email (F8).

### Implementation + PR review loop (3 rounds, all APPROVE)
Implementation: shared normalize.go updated to the loss-proof contract; adopted
in 6 services at their storage chokepoints; per-manager regression tests added;
one call start_test.go fixture corrected (UUID wrongly typed TypeTel ->
TypeConference). Parent re-verified all 6 services (go test, cleared cache) and
golangci-lint (0 new issues; one pre-existing SA5011 in email-manager
listenhandler is untouched).
- Round 1 (code-verification, ran all tests): APPROVE, 0 blocking. Confirmed
  loss-proof contract in code, nil-guards, by-index slices, DialogID untouched,
  Create chokepoint, async goroutine intact, gomock matchers assert canonical.
- Round 2 (adversarial, analytic): APPROVE, 0 blocking. Walked normalizeTel/SIP
  idempotency, SIP slice index safety (no panic), DialogID non-divergence,
  contact-manager garbage-vs-empty (benign; number_e164 is a plain INDEX not
  UNIQUE, code-verified), in-place mutation aliasing (safe, request-scoped).
- Round 3 (final, code-verification): APPROVE. Acceptance criteria 1-5 all met.
  error-discard pattern safe at all sites. Flagged agent-manager as an
  intentionally-deferred address-storing manager (normalizing it in isolation
  would break the deliberately-raw call-manager agent exact-match lookup);
  documented in §2 Out-of-scope along with queue/campaign (covered transitively
  via callHandler.Create re-normalization). PR body synced to the expanded scope.
