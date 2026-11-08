# VOIP-1690: default SMS sender and sender validation

## 1. Issue analysis

### 1.1 Ticket review (origin/main 7809711d4)

Every claim of the ticket was assumed wrong and checked against the code.

| Claim | Result | Evidence |
|---|---|---|
| `Send` and `SendMMS` reject a nil source with `SOURCE_REQUIRED` only; no ownership check, no OutboundConfig use | Holds | `bin-message-manager/pkg/messagehandler/send.go` (`source == nil` check, then `normalizeSendAddresses`), `send_mms.go` (same shape). No `NumberV1*` or `CallV1OutboundConfig*` call anywhere in message-manager. |
| A promoted MMS (long text) goes through `Send` | Holds | `send.go`: `promoted := count.Segments > smsPromoteSegments`, same function. |
| Voice path pattern | Holds | `bin-call-manager/pkg/callhandler/outgoing_call.go` `getValidatedSourceForOutgoingCall`: filter customer_id, number (or id), type=normal, status=active, deleted=false; falls back silently. The SMS path must NOT fall back silently. |
| `POST /messages` has `source` required | Holds | `bin-openapi-manager/openapi/paths/messages/main.yaml` required list. |
| Four callers | Partly wrong: five | The four plus `message-control send` (CLI, `cmd/message-control/main.go`), which calls `messageHandler.Send` directly. It builds a real request handler, so the new RPCs work there; it still requires `--source`. |
| Campaign sends SMS | No such path | `bin-campaign-manager` has no non-test `MessageV1*` caller. |
| "source in flow action / ai tool becomes optional" | Already optional | `OptionMessageSend.Source` is `*Address` with `omitempty`; the ai `send_message` tool lists only `destinations`, `text` as required. Today they fail with `SOURCE_REQUIRED` when the model or flow author leaves it out. |

New facts found (not in the ticket):

1. **api-manager never sends a nil source.** `server/messages.go` does `source := ConvertCommonAddress(req.Source)` and passes `&source`, so an omitted source arrives at message-manager as a zero-value address (empty target), not nil. "Missing" must therefore mean nil OR empty target. Once `source` is optional in OpenAPI the generated field becomes a pointer and the server passes nil.
2. `normalizeSendAddresses` mutates the caller's source in place and discards the error (loss-proof: an alphanumeric sender is kept verbatim). Validation must run after it, on the canonical form.
3. The identity-verification gate runs first in both functions and stays first. Sender resolution goes right after the destination check, before the segment count and the balance call (a rejected request has no side effect and no billing call).
4. Number list RPC: `NumberV1NumberList(ctx, token, size, filters)` with `nmnumber.FieldCustomerID/FieldNumber/FieldID/FieldType/FieldStatus/FieldDeleted`. OutboundConfig: `CallV1OutboundConfigList(ctx, customerID, 1, "")` returns the customer's live row from the DB (call-manager `OutboundConfigList`; the first page, newest first, soft-deleted rows excluded). Empty list means no config; `DefaultOutgoingSourceNumberID == uuid.Nil` means the fallback is disabled.

5. `call_outbound_configs` has `UNIQUE KEY uq_customer_id (customer_id)` (call-manager baseline migration), so there is at most one live row per customer and `pageSize=1` is exact. The voice path reads the same row through `GetByCustomerID` (Redis cache first); the only RPC message-manager can use is `CallV1OutboundConfigList` (DB direct), so for a short moment after an update the two paths can see different values. Accepted: the number is re-validated against number-manager every time.
6. Number types are `normal` and `virtual` (`bin-number-manager/models/number/number.go`). The filter type=normal rejects virtual numbers, the same set the voice path rejects; that a virtual number cannot send SMS is an assumption (not verified in code), the requirement only asks for customer numbers; a conversation whose `Self` is a virtual number therefore fails (it could not be delivered before either).
7. A valid sender is defined as: address `Type == tel`, canonical target matches the shared E.164 validator (`commonaddress.ValidateTarget`, `^\+[0-9]{7,15}$`), and a normal active number of the customer exists. Alphanumeric sender ids and any other type fail the format step, because `NormalizeTarget` keeps them verbatim and only this check rejects them.
8. Failure classes are kept apart: a lookup error (RPC failure) is an internal error and never turns into a silent default (fail closed, unlike the voice path, which logs and falls through); an empty lookup result is "not owned".

### 1.2 Caller impact (breaking change)

| Caller | source passed | Effect after the change |
|---|---|---|
| api-manager `POST /messages` (sms and mms) | whatever the client sends; docs already say "must be a number you own" (`message_overview.rst`, `message_tutorial.rst`) | Non-owned, non-E.164 (including alphanumeric) senders now fail. This is the intended change. Omitted source now works when a default exists. |
| flow-manager `message_send` action | `opt.Source`, optional pointer | Missing source works with a default; a non-owned source fails the action (`actionHandleMessageSend` returns the error, `actionhandle.go:899-902`). |
| ai-manager `send_message` tool | `tmpOpt.Source`, optional | The model may omit it; a wrong source now returns a tool failure to the model (`fillFailed`, `tool.go:686-689`) instead of a send. |
| conversation-manager `smshandler.Send` | `&cv.Self` | `cv.Self` is the inbound destination (a number of the customer at receipt time) or the source of an earlier outbound message (`conversationhandler/message.go`), plus whatever `v1_conversations.go` GetOrCreate was given over RPC. It is E.164 and customer owned except when the number was released since, or a past outbound used an unowned sender. Those conversations fail until the number is owned again (accepted). |
| `message-control send` | `--source` | Same validation. |

Code cannot show the real distribution of sources used in production (production data was not read). Mitigating facts: the documented contract already requires an owned number; the provider (Telnyx) needs a number on its messaging profile. The change applies without a grace period: an invalid or foreign sender fails. A check of the sender distribution of production messages before the deploy is recommended.

### 1.3 SendMMS decision

Included. Same sender risk, same callers (flow, ai and conversation route to `SendMMS` when medias exist), and leaving it out would let a client bypass the validation by attaching a file. One shared helper is used by both functions.

### 1.4 api-validator and tests

`POST /messages` is not run by the validator (cost). Existing unit tests in `messagehandler` (`send_test.go`, `send_mms_test.go`, `error_paths_test.go`; `validate_test.go` for the gate; `cmd/message-control` tests; dispatch and provider tests do not call the handler `Send`) use mocks and must get the new expectations (`NumberV1NumberList`, and `CallV1OutboundConfigList` for the default path). `listenhandler` and the callers use mocked `MessageV1MessageSend*`, so they are unchanged. The shared mock request handler is already generated, no new RPC is added.

### 1.5 Documents, definitions and generated files in scope

Spec and generated code (all derive from `bin-openapi-manager/openapi/openapi.yaml` and `paths/messages/main.yaml`):

- `bin-openapi-manager/gens/models/gen.go`: `cd bin-openapi-manager && go generate ./...` (`configs/config_model/generate.go`).
- `bin-api-manager/gens/openapi_server/gen.go`: `cd bin-api-manager && go generate ./...` (`openapi/config_server/generate.go`, runs after the first one). `PostMessagesJSONBody.Source` is a value `CommonAddress` today and becomes `*CommonAddress`.
- Public bundle `bin-api-manager/gens/openapi_redoc/openapi.json` (CI drift check, VOIP-1689, `.circleci/config_work.yml`). `go generate` in bin-api-manager already runs an unpinned `npx @redocly/cli` bundle (`openapi/config_redoc/generate.go`, line 4) and `build-docs` for `api.html` (line 3), so it first overwrites `openapi.json` with whatever version npx resolves. Order after the two `go generate` runs: (1) re-run the pinned command so `openapi.json` matches CI, (2) `git checkout` `api.html`. Pinned command: `cd bin-api-manager/openapi/config_redoc && npx @redocly/cli@<REDOCLY_VERSION from config_work.yml> bundle --force ../../../bin-openapi-manager/openapi/openapi.yaml --output ../../gens/openapi_redoc/openapi.json`. `gens/openapi_redoc/api.html` is not byte-deterministic and the CI does not check it; it is regenerated only if the committed one is touched by the pinned run, otherwise it is restored (`git checkout`) to keep the diff small. Decided in the design.
- Flow option text in `openapi.yaml` (`FlowManagerActionOptionMessageSend.source`, currently "Source address for the outbound message").

api-manager Go code: `server/messages.go` (`ConvertCommonAddress` takes a value; with a pointer it needs a nil branch, `source` stays nil down to `servicehandler.MessageSend/MessageSendMMS`, which already pass `*Address` unchanged to `reqHandler`, so a nil source reaches message-manager on the wire), `server/messages_test.go` (the expectations already use `*Address`; the change is a new request case without `source`) and the servicehandler tests.

Other definitions: ai tool `send_message` `source` parameter and description (`bin-ai-manager/pkg/toolhandler/definitions.go`, the `source` property near line 194), `bin-ai-manager/pkg/builderhandler/prompt.go` (mentions `send_message`; check wording only).

RST (and the rebuilt `docsdev/build`): `message_overview.rst`, `message_tutorial.rst`, `message_struct_message.rst`, `flow_struct_action.rst` (message_send), `outbound_config_overview.rst`, `outbound_config_struct_outbound_config.rst` (the default number is now also the SMS default), `ai_struct_tool.rst` (line ~196 says the source must be an account-owned number: rewrite for optional), `flow_tutorial_scenario.rst` (lines ~286 and ~340 `message_send` examples), and to be checked for `source` wording: `ai_overview.rst`, `flow_overview.rst`, `activeflow_overview.rst`, `webhook_tutorial.rst`; `call_troubleshooting.rst` only if it describes SMS. `bin-message-manager/docs/*.md` (domain, operations error list).

File path correction: the ai tool failure propagation is `bin-ai-manager/pkg/aicallhandler/tool.go:686-689`.
The uniqueness of `uq_customer_id` is taken from the call-manager baseline. If a database still had duplicate rows (older Alembic history), the helper takes the first row of the newest-first list, which is also what the customer API shows, so behaviour stays deterministic.

## 2. Design

### 2.1 Behavior

The check has two parts, both on a copy; the caller's `*Address` is never modified (today `normalizeSendAddresses` rewrites it in place; that function is replaced, see section 3).

- `prepareSource(source *Address) (*Address, error)`, local only, no RPC, runs on the RAW input before anything normalizes it:
  - nil, or `strings.TrimSpace(Target) == ""` whatever `Type` is (api-manager sends `Type ""`, `Target ""` for an absent source; the OpenAPI `type` enum includes the empty string): returns `(nil, nil)`, meaning "missing".
  - Otherwise an empty `Type` is read as `tel` (an SMS sender can only be a phone number and clients may leave out the optional `type`); any other type is `SOURCE_INVALID`.
  - The trimmed raw target must start with `+` and contain only ASCII digits `[0-9]`, spaces, `-`, `.`, `(` and `)`. `NormalizeTarget` strips every other character, so without this step `+Acme1234567` would normalize into a number; with it, letters are rejected. Then the copy is normalized (`NormalizeTarget(tel, ...)`) and `ValidateTarget(tel, target)` must pass (`+` and 7 to 15 digits). Any failure: `SOURCE_INVALID`.
- `resolveSource(ctx, customerID, prepared) (*Address, error)`, the RPC part:
  - Source missing: read the customer's OutboundConfig (`CallV1OutboundConfigList(ctx, customerID, 1, "")`). No row, or `DefaultOutgoingSourceNumberID == uuid.Nil`: `SOURCE_REQUIRED`. Else `NumberV1NumberList` with customer id, number id, type normal, status active, deleted false. Empty: `DEFAULT_SOURCE_INVALID`. Otherwise the result is `Address{Type: tel, Target: number}` (number as stored, canonical E.164; no target name is invented).
  - Source given: `NumberV1NumberList` with customer id, number (the canonical target), type normal, status active, deleted false. Empty: `SOURCE_NOT_OWNED`. No fallback to the default in this branch; the supplied `TargetName` is kept.
  - Any RPC error in the lookups, including a cancelled or expired `ctx`: `SOURCE_LOOKUP_FAILED` (Internal, 500; fail closed, the cause is wrapped and logged only, so a 404 or 503 of the callee is intentionally not passed on; never converted into a default or into a pass).

Order in the two functions (cheap local checks first, RPC based checks last before the balance call):

- `Send`: identity gate, destination check, `prepareSource`, destinations normalization, `TEXT_TOO_LONG`, `resolveSource`, balance call, create, dispatch.
- `SendMMS`: identity gate, destination check, `prepareSource`, destinations normalization, `validateMMSText`, `resolveSource` (parent `ctx`, called before `tmStart` and the `checkCtx` deadline of the media check are created, so the lookups never count against `mmsMediaCheckTimeout` or `promMMSMediaCheckSeconds`), `validateMedias` (RPC), balance, refresh, create, dispatch.

The resolved address (not the caller's pointer) is used for `create`, the dispatch copy and the stored message. A rejected sender never reaches the media check, billing or create. The destination check now runs before the source check; with both missing the error is `DESTINATION_REQUIRED` instead of `SOURCE_REQUIRED`; the existing "no source" case passes a destination and keeps `SOURCE_REQUIRED` (it needs the default lookup mocked as empty).

### 2.2 Error contract (service `message-manager`, `bin-common-handler/models/errors`)

| Reason | Constructor (HTTP) | When |
|---|---|---|
| `SOURCE_REQUIRED` | InvalidArgument (400), existing reason, message extended | no source and no default configured |
| `SOURCE_INVALID` | InvalidArgument (400) | not tel, not E.164 |
| `SOURCE_NOT_OWNED` | PermissionDenied (403) | not a normal active number of the customer |
| `DEFAULT_SOURCE_INVALID` | FailedPrecondition (409, `HTTPStatusFor` in `models/errors/rpc.go`) | default configured but released, inactive, virtual or foreign; message tells the customer to check the default outgoing source number of the outbound config |
| `SOURCE_LOOKUP_FAILED` | Internal (500) | number-manager or call-manager RPC failed |

Messages carry no phone number and no id. Logs at Info or above carry no number either (the existing Send debug log is unchanged).

### 2.3 API contract

- `POST /messages`: `source` leaves `required` (destinations and text remain). Description: omitted means the customer's OutboundConfig default outgoing source number is used (same field as outgoing calls); a given source must be a normal active number of the customer in E.164. New 400/403 reasons documented in the RST.
- api-manager passes `nil` when the field is absent; nothing else in api-manager changes.
- Flow `message_send` option and ai `send_message` tool: already optional; only the descriptions change (no schema change, no new field).
- No new global flag, no config, no cache (a cache would reintroduce the stale-ownership problem the re-validation avoids). Cost: one number-manager RPC per send, plus one call-manager RPC when the source is omitted.
- No internal-customer bypass. `IsInternalSystemID` covers four system ids (call-manager, ai-manager, system, basic route). A repo search shows no sender of SMS with those ids: the four SMS callers pass the end customer's id (flow `af.CustomerID`, ai `c.CustomerID`, `cv.CustomerID`, the authenticated customer). Only a flow or conversation owned by a system customer id could be affected, and none exists in the code.
- Availability: number-manager (every send) and call-manager (source omitted) become hard dependencies of sending. Both fail closed (`SOURCE_LOOKUP_FAILED`), unlike the identity gate, which fails open when customer-manager is down; ownership of the sender is a fraud and abuse control, so closed is chosen on purpose.
- Rollout: message-manager first, api-manager after. Old api-manager with new message-manager sends a zero-value address, handled as missing; new api-manager with old message-manager sends nil, answered `SOURCE_REQUIRED` as before. Customers using a non-owned or non-E.164 sender fail at the message-manager deploy; the sender distribution of production messages should be checked before that deploy (recommended; production data was not read).
- `message-control send` keeps requiring `--source`; the validation applies to it too.

### 2.4 Rejected alternatives

- Copying the voice fallback (replace an invalid source by the default): contradicts the requirement that an invalid source fails.
- A shared helper in `bin-common-handler` for "validated source": two consumers (call-manager returns an address or nil, message-manager returns errors), below the admission rule of three.
- Doing the check in api-manager: flow, ai and conversation do not pass through it.
- A grace period or a warn-only mode: needs a flag (not allowed) and an invalid source must fail.

## 3. Detailed plan (file level)

Tests are written first in the coding stage.

1. `bin-message-manager/pkg/messagehandler/source.go` (new): `prepareSource`, `resolveSource`, a small `lookupNumber(ctx, filters)` helper. Imports `nmnumber` and `cmoutboundconfig`; go.mod already has `replace` for both; `go mod tidy` moves call-manager from indirect to direct (the only go.mod change; number-manager is already direct).
2. `send.go`, `send_mms.go`: remove the `source == nil` guard and `normalizeSendAddresses`; add `normalizeDestinations(destinations)` (the loop of the old function, destinations only, in `send.go`); call `prepareSource` and `resolveSource` at the positions in 2.1; use the returned address afterwards. `normalizeSendAddresses` has no test of its own (only `Send`/`SendMMS` tests reach it, e.g. the `+1 555 010 0001` destination in `Test_SendMMS`), so `normalizeDestinations` gets new cases (spaces, hyphens, per index) in (m). It still normalizes the caller's `destinations` slice in place, as before (documented; only the source is copied).
3. Tests that call `h.Send` / `h.SendMMS` and need the new RPC expectations through one fixture helper (`expectSourceOwned`): `send_test.go`, `send_mms_test.go`, `error_paths_test.go`, `validate_test.go` (identity gate case: unchanged, a source is never looked up for an unverified customer). `dispatch_test.go` and `provider_mms_test.go` call the provider `SendMMS`, not the handler, and are unaffected. `listenhandler` tests mock the handler and are unaffected. `cmd/message-control` tests (`run_commands_test.go` `newControlEnv` uses a real request handler over a mock sock; `Test_runSend_sendFails` counts billing calls and needs the number-list branch, see (o)); `main_test.go` `Test_runSend_validation` is unchanged.
   New file `source_test.go` with the verify items:
   - (a) nil, empty target, whitespace target, with `Type` empty and `tel`: default path; a given target with empty `Type` is treated as `tel`.
   - (b) no config row, nil default id: `SOURCE_REQUIRED` and no number RPC.
   - (c) default id set but list empty: `DEFAULT_SOURCE_INVALID`, with the filter asserted (id, customer, normal, active, deleted false).
   - (d) alphanumeric, `sip` type, no `+`, `+Acme1234567`, `+82ab1012345678`, too short: `SOURCE_INVALID` and no RPC at all.
   - (e) valid format but list empty: `SOURCE_NOT_OWNED`; the default is NOT used and no OutboundConfig RPC happens.
   - (f) each of the three RPCs failing, and a cancelled ctx: `SOURCE_LOOKUP_FAILED`.
   - (g) the filter holds the canonical number: input `+1 (555) 010-0001` style is accepted and the filter number is `+15550100001`-style canonical (fake numbers only).
   - (h) the caller's `*Address` is unchanged after `Send`/`SendMMS` (pointer and value).
   - (i) on any `SOURCE_*` rejection: no media check (SendMMS), no billing RPC, no create, no dispatch; an unverified customer causes no number RPC.
   - (j) default path: the message is created with the default number as source and the dispatch gets that source; the supplied path keeps `TargetName`.
   - (k) SendMMS: same subset, the lookups run before `validateMedias` (mock order) and use the parent context (a media-check deadline test is unchanged).
   - (l) promoted MMS: the existing long text case in `send_test.go`, now with the number lookup expected, still ends as type mms with the MMS rate.
   - (m) `normalizeDestinations`: new cases with spaces, hyphens, several destinations (per index), unchanged alphanumeric; (d) also covers full-width digits.
   - (n) Existing expectations: `Test_SendMMS` compares `MessageCreate` and the dispatch with the source value; those now come from the returned copy, so the fixtures keep `Type tel` and the supplied `TargetName`; `expectSourceOwned` is added only to cases that reach `resolveSource`. Cases that end earlier (`TEXT_TOO_LONG` 11 segments, `ucs2 671`, `validateMMSText` failures, identity gate, destination missing) get no number RPC expectation (gomock would fail on an unexpected or a missing call).
   - (o) `message-control` builds its source with `Type: address.TypeTel` (`runSend`, `main.go`), so it does not use the empty-Type rule; that rule is tested only through (a) and the api-manager path. `run_commands_test.go` `Test_runSend_sendFails` answers every URI except `is_valid_balance` with an error (customer lookup fails open today), so with the new code the number lookup would fail and the test would see `SOURCE_LOOKUP_FAILED`, `billingCalls == 0` and a different `expectErr`: the mock gets a branch that answers the number list URI with a customer-owned number, so the test keeps proving its billing path. `main_test.go` `Test_runSend_validation` stops before `Send` (flag validation and DSN failure) and needs no change; `runSend` has no other test that reaches `Send`.
4. `bin-openapi-manager/openapi/paths/messages/main.yaml` and `openapi.yaml` (`FlowManagerActionOptionMessageSend.source` text): drop `source` from required, descriptions. Regenerate in this order: `cd bin-openapi-manager && go generate ./...`, `cd bin-api-manager && go generate ./...`, then `cd bin-api-manager/openapi/config_redoc && npx --yes @redocly/cli@2.76.0 bundle --force ../../../bin-openapi-manager/openapi/openapi.yaml --output ../../gens/openapi_redoc/openapi.json` (value of `REDOCLY_VERSION` at `.circleci/config_work.yml:1776`), then `git checkout bin-api-manager/gens/openapi_redoc/api.html`. `go generate` needs network and `npx`; when `npx` is missing it skips silently, so check `git diff --stat` on both `gen.go` files and the bundle after each run, and `git diff --stat -- '*mock_*.go'` must be empty (no new RPC). Check as CI does: re-bundle to a temp file with the same pinned command and `cmp` it with the committed `openapi.json`.
5. `bin-api-manager/server/messages.go`: `var source *commonaddress.Address`, set only when `req.Source != nil` (both the SMS and the MMS branch pass it). `ConvertCommonAddress` keeps its signature (shared with other handlers). `servicehandler/message.go` and its tests: no change. `server/messages_test.go`: new cases without `source` for the SMS and the MMS branch, expecting `nil` at `MessageSend` / `MessageSendMMS`.
6. `bin-ai-manager/pkg/toolhandler/definitions.go`, the `source` property of `send_message` (line 194 area; other `source` properties in that file belong to other tools): description says it is optional and defaults to the customer's default outgoing number. `definitions_send_message_test.go` only pins `medias`, so add an assertion that `source` is not in `required` and its description mentions the default. `bin-ai-manager/pkg/builderhandler/prompt.go` and the flow option text: wording check only.
7. Docs: RST pages of section 1.5 with the places named: `message_overview.rst` (request field text and the reason table near line 123 for `SOURCE_REQUIRED`, `SOURCE_INVALID`, `SOURCE_NOT_OWNED`, `DEFAULT_SOURCE_INVALID`, `SOURCE_LOOKUP_FAILED`), `restful_api_errors.rst` (reason table, near the `IDENTITY_VERIFICATION_REQUIRED` row), `message_tutorial.rst`, `message_struct_message.rst`, `flow_struct_action.rst`, `ai_struct_tool.rst`, `flow_tutorial_scenario.rst`, outbound config pages. Build: the committed HTML was built with Sphinx 9.1.1 and Furo 2025.12.19 and the installed `sphinx-build` is 9.1.0, so a plain rebuild would change every page. Create a scratch venv (needs network; installed Furo and `sphinxcontrib.youtube` already match, only Sphinx differs) with `sphinx==9.1.1`, `furo==2025.12.19` and the packages the `extensions` list of `docsdev/source/conf.py` needs, run `rm -rf build && python -m sphinx -M html source build` there, and check `git diff --stat` on `docsdev/build`: only pages whose source changed (plus the search index) may differ. If the tool versions cannot be matched, stop and block with `[NEEDS-TOOL]` instead of committing a whole-tree rebuild. `git add -f` the build output. Message-manager docs: `docs/domain.md` (send flow and the reason list, next to `TEXT_TOO_LONG`), `docs/dependencies.md` (the two new outbound RPCs; fix the "indirect" remark for number-manager and add call-manager). `docs/operations.md` has no error list and is not changed unless a failure-mode line fits.
8. Gates: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in message-manager, api-manager, openapi-manager, ai-manager; in flow-manager and conversation-manager the code does not change, so `go build ./... && go vet ./... && go test ./...` only (no generate); `bash scripts/check-test-conventions.sh` and the other `scripts/check-*.sh` the CI workflow lists; the pinned redoc `cmp`; `git fetch origin main` merge-tree check; no `go.mod` change outside message-manager.

Operations note for `docs/domain.md`: a cancelled or expired request context also ends as `SOURCE_LOOKUP_FAILED`, so a client that hangs up raises the 500 count.

Out of scope (suggestions only): square front-end source field handling, `voipbin/mcp` send_sms tool schema, a pre-merge check of the sender distribution of production messages (recommended before the deploy), internal-system-customer SMS senders.
