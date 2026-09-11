# PUT /mcpservers/:id: fix full-field-required semantics to true partial update

Status: DRAFT (Round 0)
Author: CPO review of PR #1285/#1290 follow-up
Related: `docs/plans/2026-09-11-mcp-tool-integration-design.md` §10 (original
API surface design), PR #1285 (feature), PR #1290 (`ValidateMcpServerIDs`
500->400 fix).

## 1. Problem statement

`PUT /mcpservers/:id` was designed with exactly one field carrying tri-state
(omitted/cleared/set) pointer semantics: `secret`. Every other mutable field
-- `name`, `detail`, `url`, `status`, `auth_type`, `api_key_header` -- is a
plain (non-pointer) value at the `bin-ai-manager` business-handler layer
(`mcpserverhandler.Update`) and gets **unconditionally included** in the SQL
`UPDATE ... SET` field map, even when the caller omitted it from the JSON
body.

This was flagged during a CPO-level review as "PUT re-validates url/status
unconditionally, forcing clients to resend the whole object" -- investigation
during design confirmed the actual defect is broader and, for some fields,
more severe than re-validation friction alone:

| Field | OpenAPI DTO type | `bin-api-manager` dereference | Effect of omitting the field on PUT |
|---|---|---|---|
| `name` | `*string` | `if req.Name != nil { name = *req.Name }` else `""` | **Silently wipes the name to `""`.** No validation rejects an empty name today (confirmed: `mcpserverhandler.Update` has no name-empty check), so this succeeds with HTTP 200 and destroys the customer's data with no warning. |
| `detail` | `*string` | same pattern | Silently wipes `detail` to `""`. Same as `name` (no data-loss guard, but `detail` is optional/cosmetic so the blast radius is smaller). |
| `url` | `*string` | same pattern | Wiped to `""` -> `ValidateURL("")` in `mcpserverhandler.Update` rejects it -> **HTTP 400 on every partial update that doesn't resend the same URL.** This is the "forces resending everything" symptom the CPO review originally named. |
| `status` | `*enum` | same pattern | Wiped to `Status("")` -> `!status.IsValid()` -> **HTTP 400**, same as `url`. |
| `auth_type` | `*enum` | same pattern | Wiped to `AuthType("")` == `AuthTypeNone` (the zero value is a **valid** enum member, not an error) -> **passes validation and silently disables authentication** on every partial update that omits it -- the single worst case in this table: it is both silent AND security-relevant (the customer's outbound MCP call to their own server stops sending `Authorization`/API-key headers without any error surfaced). |
| `api_key_header` | `*string` | same pattern | Silently wiped to `""`. Combined with the `auth_type` bug above, a customer who only meant to rotate their `secret` can end up with `auth_type=""` and `api_key_header=""`, fully de-authenticating the connector, with a `200 OK` response telling them nothing happened. |
| `secret` | `*string` | passed through as `*string`, untouched | **Correct today** -- this is the one field the original design (§10) explicitly gave tri-state pointer semantics, and it works as intended (round-trip tested in `dbhandler/mcpserver_test.go`). |

So the fix is not "loosen url/status validation" -- it is: **give every
mutable field the same nil-means-unchanged pointer semantics `secret`
already has**, at every layer between the OpenAPI-generated DTO and the SQL
`UPDATE` field map. `url`/`status` failing loudly (400) was an accidental
safety net for two of six fields; `name`/`detail`/`auth_type`/`api_key_header`
failing silently is the actual production-impacting bug.

## 2. Scope

**In scope:** `PUT /mcpservers/:id` end to end -- OpenAPI request-body
`nullable`/optional field semantics are already correct (they're already
`omitempty` pointers, generated as `*string`/`*enum` in `gen.go`); the bug is
entirely below the transport layer, in how those pointers get flattened to
plain values and then re-inflated into a full field map. Layers touched:

1. `bin-api-manager/server/mcpservers.go` (`PutMcpserversId`) -- stop
   dereferencing to `""`/zero-value; pass the raw `*string`/`*mcpserver.AuthType`/
   etc. straight through.
2. `bin-api-manager/pkg/servicehandler/mcpserver.go` (`McpServerUpdate`) --
   signature changes to accept pointers.
3. `bin-common-handler/pkg/requesthandler/ai_mcpservers.go`
   (`AIV1McpServerUpdate`) -- signature changes, wire DTO changes.
4. `bin-ai-manager/pkg/listenhandler/models/request/mcpservers.go`
   (`V1DataMcpServersIDPut`) -- wire DTO fields become pointers so
   `encoding/json` preserves the omitted-vs-empty-string distinction over
   the RabbitMQ hop too (same reasoning the original design already applied
   to `secret` in §10 MN2 -- extending it to the other five fields is not a
   new pattern, it is closing a gap in applying an already-established one).
5. `bin-ai-manager/pkg/listenhandler/v1_mcpservers.go`
   (`processV1McpServersIDPut`) -- pass pointers through unchanged.
6. `bin-ai-manager/pkg/mcpserverhandler/handler.go` (`Update`) -- this is
   where the actual "build the SQL field map" logic lives; each field's
   presence in `fields map[mcpserver.Field]any` becomes conditional on its
   pointer being non-nil, mirroring the existing `secret` branch exactly.
7. `bin-openapi-manager/openapi/paths/mcpservers/id.yaml` -- description-only
   update; the schema's field types do not change (they are already
   optional), only the prose needs to state the new "omitted = unchanged"
   contract per field instead of implying (via the current 200-line combined
   description) that omission just means "use the zero value."
8. `bin-api-manager/docsdev/source/ai_struct_mcpserver.rst` -- per this
   monorepo's mandatory RST-docs-sync rule (`CLAUDE.md` "CRITICAL: RST docs
   sync": any user-visible API behavior change requires an RST update in
   the same commit). The existing "MCP Server Implementation Hint" note
   already documents this exact omit/clear/set contract for `secret`
   alone; extend it (or add a sibling note) covering the other five
   mutable fields now getting the same contract. See §5.1.
9. Test files at every layer above (see §7).

**Out of scope (explicitly):**
- `square-admin` frontend code changes -- it already sends `name`/`detail`/
  `url`/`status`/`auth_type` unconditionally on every PUT (confirmed:
  `mcpservers_detail.js`'s `handleUpdate`, not `handleSave` as an earlier
  draft of this doc mistakenly said). A true partial-update UI (e.g. a
  "rename only" quick-action) is a possible future enhancement, not part of
  this fix. **Note the one real, if minor, behavior change this causes for
  the frontend as-is -- see the `api_key_header` staleness callout in §2.1
  below; it is accepted, not something this fix needs to also patch in the
  frontend.**
- `POST /mcpservers` (create) -- create has no "existing value to preserve"
  concept; every field is either provided or defaults per `mcpserverhandler.Create`'s
  existing logic (`status` defaults to `Active` if empty, `authType`
  defaults to `AuthTypeNone` if the pointer... wait, `POST`'s DTO
  (`V1DataMcpServersPost`) is NOT a pointer today -- but that is correct for
  Create, where there is no prior value to distinguish "omitted" from
  "default"; explicitly confirming this is NOT a bug for Create, only for
  Update, in §3 below.
- `mcp_server_ids` on `PUT /ais/:id` (the AI-to-McpServer whitelist
  association) -- that field already has correct tri-state semantics
  (`*[]uuid.UUID`, confirmed live via
  `Test_processV1AIsIDPut_McpServerIDsOmittedLeavesUntouched` in PR #467's
  antecedent work) and is unrelated to this fix.

### 2.1 Accepted behavior change for `square-admin`: `api_key_header` staleness when switching away from `api_key`

`mcpservers_detail.js`'s `handleAuthTypeChange` clears the LOCAL UI state
(`setApiKeyHeader('')`) whenever the user picks an `auth_type` other than
`api_key`, but `handleUpdate`'s PUT body only includes `api_key_header`
when `authType === 'api_key'` (`if (authType === 'api_key') { body.api_key_header
= apiKeyHeader.trim() }`) -- so switching from `api_key` to `bearer`/`none`
and saving sends a body that OMITS `api_key_header` entirely.

- **Today (pre-fix):** omitted `api_key_header` is wiped to `""`
  server-side regardless -- this happens to match what the UI just showed
  the user (it locally cleared the field too), so the bug is invisible for
  this specific flow.
- **After this fix:** omitted `api_key_header` means "leave the current
  value untouched" -- the previously-stored `api_key_header` string stays
  in the database even though the UI cleared its own local state and the
  user is no longer looking at it.

This is a real, observable behavior change, but it is **functionally inert
and accepted, not a defect this PR needs to also fix in the frontend**:
`api_key_header` is only ever read/used when `auth_type == api_key`
(confirmed: `mcpserverhandler` only sends the API-key header when
`AuthType == AuthTypeAPIKey`); once `auth_type` is `bearer`/`none`, a
stale `api_key_header` value sitting in the row is inert dead data with no
behavioral effect, and it gets correctly overwritten the next time the
customer switches back to `api_key` and provides a header name. Fixing
this cosmetically (having the frontend send an explicit `api_key_header:
""` whenever `auth_type` changes away from `api_key`) is a trivial,
independent frontend follow-up -- noted here so a reviewer doesn't have to
re-derive it, but deliberately NOT bundled into this backend-only PR (see
§2's scope: no `square-admin` code changes in this fix).

## 3. Why POST (Create) does not need the same fix

`POST /mcpservers`'s DTO (`request.V1DataMcpServersPost`,
`bin-ai-manager/pkg/listenhandler/models/request/mcpservers.go`) uses plain
`string`/`AuthType` fields, not pointers, and this is correct as-is:
Create has no prior row to "leave unchanged" -- every field is either
supplied by the caller or takes its from-scratch default
(`mcpserverhandler.Create`: `status` defaults to `StatusActive` if empty,
`secret` is encrypted only if non-empty; `authType`/`apiKeyHeader` have no
default-substitution logic in `Create` at all -- an omitted `auth_type`
simply arrives as the Go zero value `""`, which is `AuthTypeNone`, a
legitimately correct "no default needed" outcome for a brand-new row with
no prior auth configuration to preserve). There is no "omitted vs
explicitly empty" ambiguity to preserve on a brand-new row -- confirmed by
inspecting `mcpserverhandler.Create`'s full body (`handler.go:23-90`): it
never reads a value it wasn't given, so there is nothing for a pointer
type to protect against being silently overwritten. Do not touch `POST`'s
DTO or handler in this PR -- the fields being flattened to `""` /
zero-value there was always the intended default-application behavior, not
a bug.

## 4. Design: pointer-conditional field map, mirroring the existing `secret` branch

### 4.1 `mcpserverhandler.Update` (the actual fix site)

Change the signature from plain values to pointers for every field except
the ones that must remain identifiers (`ctx`, `id`):

```go
// Update updates the McpServer. Every field below follows the same PUT
// pointer semantics established for secret in design §10 MN2 and extended
// here to close the gap that left name/detail/url/status/auth_type/
// api_key_header silently or loudly clobbered by omission (see
// docs/plans/2026-09-12-mcp-server-put-partial-update-design.md): nil means
// "leave the existing value untouched", a non-nil pointer (including one
// pointing at the zero value, e.g. auth_type: "" or url: "") means
// "set to exactly this value, validate it as normal". secret additionally
// treats a non-nil pointer to "" as "explicitly clear" (distinct from
// "set to empty string" for the other string fields, since an empty secret
// is a real clear operation, not a validatable value) -- this asymmetry is
// intentional and pre-existing, not something this change alters.
func (h *mcpServerHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	detail *string,
	url *string,
	status *mcpserver.Status,
	authType *mcpserver.AuthType,
	apiKeyHeader *string,
	secret *string,
) (*mcpserver.McpServer, error) {
	log := logrus.WithFields(logrus.Fields{"func": "Update"})

	if url != nil {
		if err := ValidateURL(*url); err != nil {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_URL", err.Error()).Wrap(err)
		}
	}
	if status != nil && !status.IsValid() {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_STATUS", "invalid status: "+string(*status))
	}
	if authType != nil && !authType.IsValid() {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_AUTH_TYPE", "invalid auth_type: "+string(*authType))
	}

	fields := map[mcpserver.Field]any{}
	if name != nil {
		fields[mcpserver.FieldName] = *name
	}
	if detail != nil {
		fields[mcpserver.FieldDetail] = *detail
	}
	if url != nil {
		fields[mcpserver.FieldURL] = *url
	}
	if status != nil {
		fields[mcpserver.FieldStatus] = *status
	}
	if authType != nil {
		fields[mcpserver.FieldAuthType] = *authType
	}
	if apiKeyHeader != nil {
		fields[mcpserver.FieldAPIKeyHeader] = *apiKeyHeader
	}
	if secret != nil {
		if *secret == "" {
			fields[mcpserver.FieldSecretCiphertext] = []byte(nil)
			fields[mcpserver.FieldSecretNonce] = []byte(nil)
			fields[mcpserver.FieldKeyVersion] = 0
		} else {
			ciphertext, nonce, version, err := h.crypto.Encrypt(*secret)
			if err != nil {
				return nil, errors.Wrap(err, "could not encrypt secret")
			}
			fields[mcpserver.FieldSecretCiphertext] = ciphertext
			fields[mcpserver.FieldSecretNonce] = nonce
			fields[mcpserver.FieldKeyVersion] = version
		}
	}

	if len(fields) == 0 {
		// A PUT with every field omitted (or only unchanged/invalid-nil
		// pointers) is a client no-op, not a server error. Returning the
		// current row (instead of erroring or issuing a zero-column
		// UPDATE, which some SQL builders reject) matches List/Get's
		// "always return current state" contract and avoids a special
		// "PATCH-with-nothing-to-patch" error class nothing else in this
		// API returns.
		return h.Get(ctx, id)
	}

	if err := h.db.McpServerUpdate(ctx, id, fields); err != nil {
		return nil, errors.Wrapf(err, "could not update mcp server")
	}

	res, err := h.db.McpServerGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated mcp server")
	}
	log.WithField("mcp_server", res).Debugf("Updated mcp server. mcp_server_id: %s", res.ID)
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, mcpserver.EventTypeUpdated, res)

	return res, nil
}
```

Key points a reviewer should check against this snippet:
- `fields` starts **empty**, not pre-populated -- this is the actual bug
  fix. The old code built `fields` with all six non-secret keys always
  present; the new code only adds a key when its pointer is non-nil.
- Validation (`ValidateURL`, `status.IsValid()`, `authType.IsValid()`) only
  runs when the caller actually supplied that field -- this directly
  resolves the "forces resending everything to pass validation" complaint,
  because an omitted `url`/`status` is no longer coerced into an empty
  string that then fails validation.
- The `len(fields) == 0` short-circuit calls the existing `Get`, reusing its
  existing `ErrNotFound` -> `cerrors.NotFound` mapping (see `Get`'s body,
  §6.1) rather than duplicating that error-translation logic.

### 4.2 `bin-api-manager/server/mcpservers.go` (`PutMcpserversId`)

Delete every `if req.X != nil { x = *req.X }` dereference block for
`name`/`detail`/`url`/`status`/`auth_type`/`api_key_header` and pass the
OpenAPI-generated pointers straight through, doing only the necessary type
conversion (the generated DTO uses a locally-scoped
`PutMcpserversIdJSONBodyStatus`/`PutMcpserversIdJSONBodyAuthType` string
type per field, same as today; convert element-wise, not by dereferencing):

```go
var statusPtr *ammcpserver.Status
if req.Status != nil {
	v := ammcpserver.Status(*req.Status)
	statusPtr = &v
}

var authTypePtr *ammcpserver.AuthType
if req.AuthType != nil {
	v := ammcpserver.AuthType(*req.AuthType)
	authTypePtr = &v
}

res, err := h.serviceHandler.McpServerUpdate(c.Request.Context(), a, target, req.Name, req.Detail, req.Url, statusPtr, authTypePtr, req.ApiKeyHeader, req.Secret)
```

`req.Name`, `req.Detail`, `req.Url`, `req.ApiKeyHeader`, `req.Secret` are
already `*string` in the generated `PutMcpserversIdJSONBody` (confirmed:
`gen.go:9293-9307`) -- no conversion needed for those five, only the two enum
fields need the type-rebind shown above (Go's type system does not allow
converting `*A` to `*B` directly even when `A`/`B` share an underlying
type).

### 4.3 `servicehandler.McpServerUpdate` / `requestHandler.AIV1McpServerUpdate` / `V1DataMcpServersIDPut`

Mechanical signature/DTO propagation of the same pointer types through the
three intermediate layers between `bin-api-manager`'s HTTP handler and
`bin-ai-manager`'s business handler. `V1DataMcpServersIDPut`
(`bin-ai-manager/pkg/listenhandler/models/request/mcpservers.go`) becomes:

```go
// V1DataMcpServersIDPut is v1 data type request struct for
// /v1/mcp_servers/<mcp-server-id> PUT
//
// Every field is a pointer: nil means "leave the existing value
// untouched", matching the OpenAPI-layer contract this DTO carries over
// the RabbitMQ hop. This mirrors the pre-existing Secret pointer pattern,
// extended to the other five mutable fields -- see
// docs/plans/2026-09-12-mcp-server-put-partial-update-design.md.
type V1DataMcpServersIDPut struct {
	Name         *string             `json:"name,omitempty"`
	Detail       *string             `json:"detail,omitempty"`
	URL          *string             `json:"url,omitempty"`
	Status       *mcpserver.Status   `json:"status,omitempty"`
	AuthType     *mcpserver.AuthType `json:"auth_type,omitempty"`
	APIKeyHeader *string             `json:"api_key_header,omitempty"`
	Secret       *string             `json:"secret,omitempty"`
}
```

`processV1McpServersIDPut` (`bin-ai-manager/pkg/listenhandler/v1_mcpservers.go`)
passes `req.Name`, `req.Detail`, etc. straight through to
`h.mcpServerHandler.Update(...)` unchanged in shape -- no dereferencing, no
new logic, since the fields are already the right pointer types after §4.1
lands.

### 4.4 Why the RabbitMQ-hop DTO must be pointers too, not just the HTTP-layer one

A design that fixed only `bin-api-manager`'s dereferencing (§4.2) while
leaving `V1DataMcpServersIDPut` as plain strings would silently reintroduce
the exact same bug one hop later: `bin-api-manager` would correctly know
"the caller omitted `auth_type`", but `AIV1McpServerUpdate` would have
nothing to encode that distinction into the RabbitMQ JSON body except
`""`/zero-value again, and `bin-ai-manager`'s `processV1McpServersIDPut`
would be back to unable to tell "omitted" from "set to empty" at its own
`json.Unmarshal` boundary. The pointer semantics must be threaded through
literally every hop between the OpenAPI schema and the SQL field map, or
the fix is incomplete. This is precisely why `secret`'s existing design
(§10 MN2) already made this exact argument for one field -- §4.1-4.3 above
just apply that same argument to the other five.

## 5. OpenAPI spec changes

Schema field types do not change (`name`/`detail`/`url`/`status`/
`auth_type`/`api_key_header`/`secret` were already all optional in the PUT
request body schema -- `required` is not set on any of them in
`id.yaml`). Only the `description` block changes, to make the
"omitted = unchanged" contract explicit for every field instead of the
current wording, which only calls this out for `secret`:

```yaml
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          description: >
            Every field is optional. Omitting a field leaves its current
            value unchanged; a full resend of every field is NOT required
            to update a single field (e.g. PUT { "name": "new name" } only
            renames the server and leaves url/status/auth_type/
            api_key_header/secret exactly as they were).
          properties:
            name:
              type: string
              description: "Omit to leave the current name unchanged."
            detail:
              type: string
              description: "Omit to leave the current detail unchanged."
            url:
              type: string
              format: uri
              description: "Streamable-HTTP MCP endpoint. Must be https. Omit to leave the current URL unchanged."
            status:
              type: string
              enum: ["active", "disabled"]
              description: "Set to disabled to exclude this server from ListTools/CallTool without deleting it. Omit to leave the current status unchanged."
              example: "active"
            auth_type:
              type: string
              enum: ["", "bearer", "api_key"]
              description: "Omit to leave the current auth_type unchanged. NOTE: an explicit empty string (\"\") is a valid value meaning no-auth, distinct from omitting the field."
            api_key_header:
              type: string
              description: "Omit to leave the current api_key_header unchanged."
            secret:
              type: string
              description: "Omit this field to leave the existing secret unchanged. Send an empty string to clear it."
```

The `auth_type` description's explicit "empty string is a valid value,
distinct from omitting the field" callout matters more here than for the
other fields, because `""` is `AuthTypeNone` (a real, meaningful,
non-error value) rather than an obviously-invalid placeholder the way an
empty `url` or empty `status` is -- a reader could otherwise assume
`auth_type: ""` and omitting `auth_type` are the same request.

### 5.1 RST docs update (`ai_struct_mcpserver.rst`)

Extend the existing "MCP Server Implementation Hint" note (currently
`secret`-only, lines 41-43 of the current file) into two notes -- keep the
`secret`-specific one as-is (it is still accurate and still the one field
with true three-way clear/set/leave semantics), and add a new note
covering the other six fields:

```rst
.. note:: **MCP Server Implementation Hint**

   The secret (bearer token or API key) is write-only: it is accepted on
   ``POST /mcp_servers`` and ``PUT /mcp_servers/{id}`` but is **never
   returned** in any ``GET`` response. On ``PUT``, omitting the secret
   field leaves the currently stored secret unchanged; sending an explicit
   value (including an empty string) replaces it. This distinguishes "I'm
   not touching the secret" from "clear the secret."

.. note:: **MCP Server Implementation Hint**

   ``PUT /mcp_servers/{id}`` is a true partial update: every field
   (``name``, ``detail``, ``url``, ``status``, ``auth_type``,
   ``api_key_header``, in addition to ``secret`` above) is optional, and
   omitting a field leaves its current value unchanged. A full resend of
   every field is never required -- for example, ``PUT {"name": "new
   name"}`` renames the server and leaves everything else (including
   ``url``, ``status``, and ``auth_type``) exactly as it was. One
   exception worth calling out: ``auth_type: ""`` is a valid, meaningful
   value (no authentication) and is distinct from omitting the
   ``auth_type`` field entirely -- sending the empty string explicitly
   sets no-auth, while omitting the field preserves whatever ``auth_type``
   the server already had.
```

Also update `id.yaml`'s `description` field per §5 above in the same
commit (both files describe the same contract; they must not drift).
Per this monorepo's standing RST workflow (`CLAUDE.md`), after editing the
`.rst` source: clean rebuild (`cd bin-api-manager/docsdev && rm -rf build
&& python3 -m sphinx -M html source build`), then `git add -f
bin-api-manager/docsdev/build/` (the root `.gitignore` excludes `build/`
by default) so the rebuilt HTML lands in the same commit as the RST
source change.

## 6. Interaction with existing behavior and PR #1290

### 6.1 `mcpserverhandler.Get`'s `ErrNotFound` mapping is reused, not duplicated

The `len(fields) == 0` no-op path (§4.1) calls `h.Get(ctx, id)`, which
already correctly maps `dbhandler.ErrNotFound` to `cerrors.NotFound` (400
was PR #1290's target for a DIFFERENT function,
`aihandler.ValidateMcpServerIDs`; `mcpserverhandler.Get`'s existing
`ErrNotFound` -> `cerrors.NotFound` -> HTTP 404 mapping is unrelated and
already correct -- confirmed in `handler.go:93-107`). A PUT with an
all-omitted body against a nonexistent id therefore correctly 404s via the
same path a GET would, rather than needing its own not-found check.

### 6.2 No interaction with `ValidateMcpServerIDs` (PR #1290)

PR #1290's fix lives entirely in `bin-ai-manager/pkg/aihandler` and is only
reachable from `PUT /ais/:id`'s `mcp_server_ids` field -- a completely
different resource and field. This design does not touch that code path;
noted here only to confirm no overlap/conflict for a reviewer scanning both
PRs' diffs against the same `bin-ai-manager` service.

## 7. Testing strategy

Table-driven tests at the two layers with real branching logic:

**`mcpserverhandler.Update` (`handler_test.go`):** extend the existing
Update test(s) with cases covering every field independently:
- all fields nil except `name` -> only `FieldName` present in the
  `McpServerUpdate` mock's expected `fields` map (assert exact map
  contents, not just "no error", so a regression that accidentally
  re-includes an omitted field is caught).
- `status: nil` -> no `status.IsValid()` call, no error, `FieldStatus`
  absent from the fields map (this is the direct regression test for the
  400-on-omission bug).
- `auth_type: nil` -> `FieldAuthType` absent from the fields map (this is
  the regression test for the silent-deauth bug -- the case that matters
  most, given §1's severity ranking).
- `auth_type: &AuthTypeNone` (pointer to the zero value, explicitly set,
  not omitted) -> `FieldAuthType` present with value `AuthTypeNone`,
  passes validation (confirms the "pointer to zero value is a real,
  validated value" semantics from §4.1's doc comment, distinct from nil).
- every field nil (all omitted) -> `db.McpServerUpdate` is NOT called
  (assert via gomock's `Times(0)` or simply not setting up an `EXPECT()`
  for it, so an unexpected call fails the test), `db.McpServerGet`/`Get`'s
  own mock IS called once, returns the current row unchanged.
- `url: nil` alongside `secret: &"newsecret"` -> confirms a secret-only
  rotation (the original bug report's literal repro case) succeeds without
  needing to resend `url`.
- existing `secret` pointer-semantics tests (rotate/clear/leave-untouched)
  continue to pass unchanged -- confirms this refactor did not alter
  `secret`'s existing, already-correct behavior.

**`v1_mcpservers_test.go` (new file, listenhandler layer, mirrors the
pattern PR #1290 added for `v1_ais_test.go`):** one end-to-end test posting
a PUT body with only `{"name": "new name"}` through
`processV1McpServersIDPut` and asserting `mcpServerHandler.Update` is
called with `name` non-nil and every other pointer nil, pinning the full
`json.Unmarshal` -> `Update` call chain, not just the business-handler unit
in isolation.

**Revert-and-rerun requirement (per this monorepo's standing review-loop
practice):** for at least the `auth_type: nil` and `status: nil` cases
above, confirm each new test genuinely fails against the pre-fix code
(`git stash` the `handler.go` change, rerun, confirm FAIL, restore) before
considering the test suite complete -- do not rely on "the test passes
against my fix" alone as evidence it catches the regression it claims to.

**`servicehandler`/`requesthandler` layers:** existing mock-based tests at
these layers need their `McpServerUpdate`/`AIV1McpServerUpdate` call-site
mocks updated for the new pointer parameter types (mechanical, no new
test cases required at these pure-passthrough layers beyond confirming the
build/existing assertions still compile and pass).

**`bin-api-manager/server/mcpservers_test.go` (existing file, HTTP layer --
closes the gap Round 1 review flagged):** this is where the
dereference-to-zero-value pattern originally lived (§4.2) and where a
regression could silently reintroduce it without tripping any test below
this layer, since every layer below `bin-api-manager` only ever sees
whatever `PutMcpserversId` decided to pass down. Add at minimum:
- a PUT request with a JSON body containing only `{"name": "new name"}`
  -> asserts `serviceHandler.McpServerUpdate` (via its gomock) is called
  with `name` non-nil and every other pointer argument nil -- pins the
  full `c.BindJSON` -> pointer-passthrough -> `McpServerUpdate` call chain
  at this layer specifically, independent of the deeper layers' own tests.
- a PUT request with `{"status": "disabled"}` alone -> asserts `statusPtr`
  is correctly rebuilt from the OpenAPI-generated
  `PutMcpserversIdJSONBodyStatus` local type into `*ammcpserver.Status`
  (the one place in this layer with a real type-conversion step, per §4.2)
  and every other pointer stays nil.

## 8. Non-goals

- No change to `POST /mcpservers` (see §3).
- No change to `square-admin` frontend (see §2 -- it already always sends
  every field; this fix is invisible to it, not a breaking change).
- No new PATCH-vs-PUT semantic debate -- this fix keeps the endpoint as PUT
  (already established, `/mcpservers/:id` is not being redesigned as
  PATCH); "PUT with optional fields defaulting to unchanged" is an
  accepted, existing pattern in this codebase (`mcp_server_ids` on
  `PUT /ais/:id` already works this way) and is being extended here, not
  invented.
- No retroactive fix/backfill for any MCP server rows that may have already
  had `name`/`detail`/`auth_type`/`api_key_header` silently wiped by the
  pre-fix bug in production -- if any occurred, that is a separate
  data-remediation question (checking `bin-ai-manager`'s change/audit
  trail if one exists, or asking affected customers to re-verify their MCP
  server configs) to raise with 대표님 separately once this fix ships,
  not something this design/PR addresses.

## 9. Round 1 review disposition

Independent review (`deleg_7c6523ae`) verdict: REQUEST CHANGES. Findings
and fixes:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | §2's frontend function name cited as `handleSave`; actual name is `handleUpdate` | MINOR | §2 "Out of scope" bullet corrected |
| 2 | §3 contained a leftover mid-draft sentence fragment ("wait, POST's DTO... is NOT a pointer today") | MINOR | §3 rewritten cleanly with the POST/Create non-bug argument fully spelled out and grounded in `handler.go:23-90` |
| 3 | §2's "square-admin needs zero changes" claim missed a real (if inert) behavior change: switching `auth_type` away from `api_key` now leaves a stale `api_key_header` in the DB instead of wiping it, because the frontend only sends `api_key_header` when `authType === 'api_key'` | MAJOR (claim was factually incomplete, though the underlying behavior change itself is benign) | New §2.1 added: documents the exact mechanism (`handleAuthTypeChange` vs `handleUpdate`'s conditional), confirms via `mcptoolhandler/client.go`'s `case mcpserver.AuthTypeAPIKey` that a stale header is functionally inert when `auth_type != api_key`, and explicitly marks this an accepted trade-off, not a defect to fix in this PR |
| 4 | RST docs (`ai_struct_mcpserver.rst`) update missing from scope, violating this monorepo's mandatory `CLAUDE.md` "CRITICAL: RST docs sync" rule | MAJOR (process violation, not a logic defect) | §2 point 8 added (RST file now listed as an in-scope layer); new §5.1 added with the exact RST note text and the required clean-rebuild-and-commit-build-output steps |
| 5 | Test strategy (§7) had no coverage for `bin-api-manager/server/mcpservers.go`'s own dereference/type-rebuild logic -- a regression there wouldn't be caught by any lower-layer test | MAJOR | §7 extended with a new `bin-api-manager/server/mcpservers_test.go` subsection specifying two concrete test cases (name-only PUT pins the pointer passthrough; status-only PUT pins the enum type-rebuild) |

Not changed (reviewer confirmed correct as originally written): code
snippet accuracy against current `handler.go`/`gen.go`/`dbhandler` (all
verified byte-accurate except finding #1's function name), OpenAPI
`required`-field-absence claim (directly verified against `id.yaml`),
`len(fields) == 0` -> `Get()` / `ErrNotFound` mapping reuse, POST-out-of-scope
conclusion (only the prose needed cleanup, not the conclusion), and overall
scope boundaries (no missing/extra services beyond the RST gap above).
