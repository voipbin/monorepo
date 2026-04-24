# Provider tech prefix/postfix/headers — apply on outbound dial

Date: 2026-04-25
Status: Approved (design)
Scope: `bin-call-manager` only

## Problem

`Provider.TechPrefix`, `Provider.TechPostfix`, and `Provider.TechHeaders` are stored on the provider model, editable via the Providers API, and documented in `bin-api-manager/docsdev/source/provider_overview.rst` and `provider_struct_provider.rst` as user-visible behavior (prefix/suffix on dialed numbers, custom SIP headers on the outgoing INVITE).

They are silently dropped on outbound dial. The only consumer is `bin-call-manager/pkg/callhandler/outgoing_call.go:getDialURITel`, which builds the dial URI as:

```go
res := fmt.Sprintf("pjsip/%s/sip:%s@%s;transport=%s",
    pjsipEndpointOutgoing, c.Destination.Target, pr.Hostname, constTransportUDP)
```

Only `pr.Hostname` is used. `TechPrefix`, `TechPostfix`, `TechHeaders` are never applied anywhere in the call flow. The docs contradict the code.

## Goal

On an outbound tel-type call that routes through a Provider:

1. Wrap the user part of the SIP URI with `TechPrefix` and `TechPostfix`:
   `pjsip/call-out/sip:<TechPrefix><Target><TechPostfix>@<Hostname>;transport=udp`.
2. Attach each entry in `TechHeaders` to the outgoing INVITE as a custom SIP header using the existing `PJSIP_HEADER(add,<name>) = <value>` channel-variable pattern.
3. Protect correctness/security-critical system-set headers from override by operator tech_headers.
4. Sanitize operator-supplied header keys/values to prevent SIP header injection.

Non-goals:

- SIP-direct path (`getDialURISIPDirect`) — does not route through a Provider.
- Value templating (`{{call_id}}`, etc.) — static key/value only.
- Provider-save-time validation in `bin-route-manager` — defense-in-depth at dial time only for this PR. Save-time validation tracked as a follow-up.

## Change shape

One focused patch in `bin-call-manager/pkg/callhandler/`:

1. Change `getDialURITel` signature from `(string, error)` to `(string, map[string]string, error)`. It already fetches the Provider — it now also returns the sanitized `tech_headers` map and embeds prefix/postfix in the URI.
2. Update `getDialURI` dispatcher signature to `(string, map[string]string, error)`.
3. `getDialURISIP` and `getDialURISIPDirect` return a nil map — no Provider in those paths.
4. New helper `mergeTechHeaders(dst, src, log)` enforces a reserved-key denylist and sanitization; skipped entries are logged at `Warn` level.
5. In `createChannelOutgoing`:
   - Obtain dial URI + tech_headers from `getDialURI`.
   - Seed `channelVariables` with `mergeTechHeaders(channelVariables, techHeaders, log)`.
   - Then call `setChannelVariableTransport` and `setChannelVariablesCallerID` — these overwrite any surviving collisions for headers they set directly.
   - One `Info` log line when any tech config is applied.

Rationale:

- Returning the header map is more explicit than mutating a shared variables map inside `getDialURITel`, and keeps URI construction and provider lookup co-located.
- A named `mergeTechHeaders` helper is the right home for the denylist + sanitization rules, vs. scattering them across the merge site.
- Ordering (seed-then-overwrite) handles the CALLERID/PAI/SDP-Transport case cleanly for headers the system code path actually writes; the explicit denylist in `mergeTechHeaders` covers the gap for headers the system only writes conditionally (e.g., `P-Asserted-Identity` is only set when `anonymous == true`).

## Collision & sanitization rules

`mergeTechHeaders(dst map[string]string, src map[string]string, log *logrus.Entry)`:

For each `(k, v)` in `src`:

1. Skip if `k == ""`. Log `Warn: empty tech_header key`.
2. Skip if `k` contains any of `\r`, `\n`, `(`, `)`, `,`. Log `Warn: invalid tech_header key char`. This blocks operators from passing `PJSIP_HEADER(add,X)` as a key (which would double-wrap) or injecting control chars.
3. Wrap to channel-variable form: `varKey := "PJSIP_HEADER(add," + k + ")"`.
4. Skip if `varKey` is in the reserved set. Log `Warn: tech_header collides with system-reserved header`.
5. Skip if `v` contains `\r` or `\n`. Log `Warn: tech_header value contains CRLF`.
6. Otherwise `dst[varKey] = v`.

Reserved set (explicit denylist):

```go
var reservedTechHeaderKeys = map[string]struct{}{
    "PJSIP_HEADER(add,P-Asserted-Identity)":     {},
    "PJSIP_HEADER(add,Privacy)":                 {},
    "PJSIP_HEADER(add,SDP-Transport)":           {},
    "PJSIP_HEADER(add,X-VoIPBin-Call-ID)":       {},
    "PJSIP_HEADER(add,X-VoIPBin-Confbridge-ID)": {},
    "CALLERID(name)": {},
    "CALLERID(num)":  {},
    "CALLERID(pres)": {},
}
```

The `CALLERID(*)` entries exist because operators could attempt to set them via `tech_headers` even though those keys don't fit the `PJSIP_HEADER(add,...)` wrapping — the helper checks the raw `k` as well for those three before wrapping.

## URI construction

In `getDialURITel`, after successful provider fetch:

```go
userPart := pr.TechPrefix + c.Destination.Target + pr.TechPostfix
dialURI  := fmt.Sprintf("pjsip/%s/sip:%s@%s;transport=%s",
    pjsipEndpointOutgoing, userPart, pr.Hostname, constTransportUDP)
```

Empty `TechPrefix`/`TechPostfix` are harmless no-ops (empty-string concat). This preserves backwards compatibility for providers that don't set these fields.

## Error handling

- Provider fetch error → return error from `getDialURITel`, same as today. Call fails.
- Invalid tech_header entries (reserved key, CRLF, malformed key) → skip the offending entry only, log `Warn`, continue the call. Partial tech config is safer than a dropped call.
- `tech_headers == nil` or empty → no-op, no log.

## Observability

- One `Info` log in `createChannelOutgoing` when any tech config is applied:
  `"Applied provider tech config. provider_id=%s prefix_len=%d postfix_len=%d headers_applied=%d headers_skipped=%d"`
- One `Warn` log per skipped header with reason (empty key / invalid char / reserved / CRLF value) and provider_id.
- No metrics. The log line is sufficient for production debugging; adding a counter here would be overkill.

Per monorepo convention (memory: external-integration logging), these are `Info`/`Warn`, not `Debug`, because tech config changes the on-wire INVITE and must be traceable in production without raising the log level.

## Testing strategy

All tests in `bin-call-manager/pkg/callhandler/`.

New unit tests on the `mergeTechHeaders` helper (single file, table-driven):

- empty src → dst unchanged
- nil src → dst unchanged
- normal header → added with `PJSIP_HEADER(add,...)` wrap
- empty key → skipped + warn logged
- key with `\r` / `\n` / `(` / `)` / `,` → skipped + warn logged
- key whose wrapped form matches reserved set → skipped + warn logged
- `CALLERID(name)` / `CALLERID(num)` / `CALLERID(pres)` → skipped + warn logged
- value with `\r` / `\n` → skipped + warn logged
- dst pre-populated with same wrapped key → dst value overwritten (merge semantics), with note that the system-set functions re-overwrite in `createChannelOutgoing` ordering

New/updated table-driven tests for `getDialURITel`:

- prefix only (`"0011"`, postfix `""`, headers nil)
- postfix only (`""`, `"#"`, nil)
- both (`"0011"`, `"#"`, nil)
- headers only (`""`, `""`, `{"X-Carrier-Auth": "tok"}`)
- all three
- empty (all zero values — backwards compat; existing tests must still pass)
- provider fetch error

Existing `outgoing_call_test.go` test cases at lines ~152, 790, 1293, 1417 already set `PJSIP_HEADER(add,SDP-Transport)` channel variables. The plan must enumerate every test file and every call site of `getDialURITel` / `getDialURI` / `createChannelOutgoing` and update them to the new signature. No test is expected to fail behaviorally after the fix — they just need the new return values plumbed through.

## Provider-call verification endpoint

`bin-api-manager/docsdev/source/providercall_overview.rst` documents a post-config verification flow. During implementation, grep for `providercall` / `ProviderCall` and confirm its call path terminates in `getDialURITel`. Expected outcome: yes (it would be odd to have a separate dial path just for provider verification). If the path diverges, flag it as a separate follow-up, do **not** expand scope.

## Out-of-scope follow-ups

- Save-time validation of tech_header keys/values in `bin-route-manager` (today they are stored as-is; dial-time sanitization is sufficient for this PR but UX is better at save time).
- Value templating (`{{source_target}}`, etc.) — not needed by any current carrier integration.
- Applying tech_headers on SIP-direct path — no Provider involved there.

## Affected files (preview)

Code:

- `bin-call-manager/pkg/callhandler/outgoing_call.go` — signature change + URI wrap + Info log.
- `bin-call-manager/pkg/callhandler/tech_headers.go` (new) — `mergeTechHeaders` helper + reserved set.
- `bin-call-manager/pkg/callhandler/outgoing_call_test.go` — update existing tests for new signature; add new cases.
- `bin-call-manager/pkg/callhandler/tech_headers_test.go` (new) — table-driven tests for the helper.

Verification:

- Pre-commit: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in `bin-call-manager`.
- Manual grep for `providercall` to confirm the verification endpoint shares the fixed code path.

Docs: none. RST already describes the behavior correctly; this PR makes code match the docs.
OpenAPI: none. `WebhookMessage` already exposes `tech_prefix` / `tech_postfix` / `tech_headers`.
Database: none.
