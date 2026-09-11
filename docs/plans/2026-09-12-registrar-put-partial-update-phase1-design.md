# Phase 1: PUT /extensions/{id} and PUT /trunks/{id} partial-update fix

Status: DRAFT (Round 0)
Author: CPO, per 대표님 approval to proceed directly to Phase 1 (no gating on
the roadmap PR's merge)
Related: `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md`
(§4 Phase 1 definition, §3 the reusable technical pattern this doc applies),
`docs/plans/2026-09-12-mcp-server-put-partial-update-design.md` (the shipped
reference implementation, PR #1291, merged as `81a9a6b98`).

## 1. Problem statement

`PUT /extensions/{id}` and `PUT /trunks/{id}` (both owned by
`bin-registrar-manager`) require every body field on every request:

- `extensions/id.yaml`: `name`, `detail`, `password` all `required`.
- `trunks/id.yaml`: `name`, `detail`, `auth_types`, `username`, `password`,
  `allowed_ips` all `required`.

Both are rated **HIGH** risk in the roadmap (§2 rows 3-4): `password` is a
SIP credential field. A client that omits it (intending to rename the
extension/trunk only) sends `password: ""`, which is written straight to
`ps_auths`/trunk auth via `AstAuthUpdate`/`SIPAuthUpdate` -- silently
breaking SIP registration/trunk authentication with a `200 OK` response and
no error. `trunks/{id}`'s `allowed_ips` carries an additional risk: an
omitted/empty array clears the trunk's IP allowlist, which (depending on
how the SIP proxy layer treats an empty allowlist) could either lock out
the legitimate carrier or, worse, remove an access restriction.

This is the exact bug class already fixed for `mcpservers` (PR #1291):
non-pointer wire types collapse "the caller didn't send this field" and
"the caller explicitly cleared it" into the same signal.

### 1.1 The DB layer already does true partial update -- the bug is entirely above it

Unlike a from-scratch fix, `bin-registrar-manager`'s dbhandler layer is
already correct and requires **no changes**:

- `dbhandler.ExtensionUpdate(ctx, id, fields map[extension.Field]any)` /
  `dbhandler.TrunkUpdate(ctx, id, fields map[trunk.Field]any)` both build
  the SQL `UPDATE` via `squirrel.Update(...).SetMap(fields)` -- only keys
  present in the map are written.
- Both have a `len(fields) == 0` early-return guard already.

The entire bug lives in the four layers between the OpenAPI schema and
these dbhandler calls, all of which currently use **plain (non-pointer)
value types**, collapsing "omitted" and "explicit empty string" into the
same value before it ever reaches the field-map construction:

1. `bin-openapi-manager/openapi/paths/extensions/id.yaml` /
   `trunks/id.yaml`: `required:` lists every field.
2. `bin-api-manager/server/extensions.go` (`PutExtensionsId`) /
   `trunks.go` (`PutTrunksId`): pass the generated (currently non-pointer)
   `req.Name`/`req.Password`/etc. straight into
   `serviceHandler.ExtensionUpdate`/`TrunkUpdateBasicInfo` as plain
   `string`/`[]string` values.
3. `bin-api-manager/pkg/servicehandler/extension.go`
   (`ExtensionUpdate(ctx, a, id, name, detail, password string)`) /
   `trunk.go` (`TrunkUpdateBasicInfo(ctx, a, id, name, detail string,
   authTypes []rmsipauth.AuthType, username, password string, allowedIPs
   []string)`): plain value parameters, passed straight to
   `reqHandler.RegistrarV1ExtensionUpdate`/`RegistrarV1TrunkUpdateBasicInfo`.
4. `bin-common-handler/pkg/requesthandler/registrar_extensions.go`
   (`RegistrarV1ExtensionUpdate`) / `registrar_trunks.go`
   (`RegistrarV1TrunkUpdateBasicInfo`): builds
   `rmrequest.V1DataExtensionsIDPut{Name: name, Detail: detail, Password:
   password}` / `V1DataTrunksIDPut{...}` -- these wire-DTO fields are
   plain `string`/`[]sipauth.AuthType`/`[]string`, marshaled to JSON and
   sent over the RabbitMQ hop.
5. `bin-registrar-manager/pkg/listenhandler/models/request/extensions.go`
   (`V1DataExtensionsIDPut`) / `trunks.go` (`V1DataTrunksIDPut`): the
   receiving wire DTO, also plain value types.
6. `bin-registrar-manager/pkg/listenhandler/v1_extensions.go`
   (`processV1ExtensionsIDPut`) / `v1_trunks.go` (`processV1TrunksIDPut`):
   this is where the actual field-map is built --
   `fields := map[extension.Field]any{extension.FieldName: req.Name,
   extension.FieldDetail: req.Detail, extension.FieldPassword:
   req.Password}` unconditionally includes all three/six keys regardless
   of whether the caller sent them, because `req.Name` etc. are plain
   strings with no "was this set" signal left after JSON unmarshaling an
   omitted field to its zero value.

So this fix must thread pointer semantics through all six layers above,
mirroring PR #1291 §4.4's argument for why every hop matters (fixing only
`bin-api-manager`'s HTTP layer would just move the "omitted flattens to
empty string" collapse one hop later, at the RabbitMQ wire DTO).

## 2. Scope

**In scope, `PUT /extensions/{id}`:**
- `bin-openapi-manager/openapi/paths/extensions/id.yaml`: remove
  `required:` from `name`/`detail`/`password`; per-field description
  update stating the omitted-means-unchanged contract.
- `bin-api-manager/server/extensions.go` (`PutExtensionsId`): pass
  generated pointers straight through.
- `bin-api-manager/pkg/servicehandler/extension.go` (`ExtensionUpdate`):
  signature becomes `name, detail, password *string`.
- `bin-common-handler/pkg/requesthandler/registrar_extensions.go`
  (`RegistrarV1ExtensionUpdate`): signature becomes pointers, wire DTO
  built with pointers.
- `bin-registrar-manager/pkg/listenhandler/models/request/extensions.go`
  (`V1DataExtensionsIDPut`): fields become `*string` with `omitempty`.
- `bin-registrar-manager/pkg/listenhandler/v1_extensions.go`
  (`processV1ExtensionsIDPut`): field-map construction becomes
  conditional (`if req.Name != nil { fields[...] = *req.Name }`, etc.).
- `bin-registrar-manager/pkg/extensionhandler/extension.go` (`Update`):
  no change needed to its own signature (it already takes
  `fields map[extension.Field]any` built by the listenhandler) --
  **except** the existing `if password, ok := fields[extension.FieldPassword];
  ok { ... AstAuthUpdate ... }` branch already correctly checks map
  presence, so once the listenhandler only inserts `FieldPassword` when
  the caller actually sent it, this branch's existing logic becomes
  correct automatically. No business-handler code change required here,
  only its caller's behavior changes -- confirm with a test, not a code
  edit.
- `bin-api-manager/docsdev/source/` -- find and update the extension
  struct/overview RST doc with an Implementation Hint note (see §5).
- Tests at every layer per §6.

**In scope, `PUT /trunks/{id}`:** identical set of layers, substituting
`trunk`/`Trunk` for `extension`/`Extension`:
- `bin-openapi-manager/openapi/paths/trunks/id.yaml`.
- `bin-api-manager/server/trunks.go` (`PutTrunksId`).
- `bin-api-manager/pkg/servicehandler/trunk.go` (`TrunkUpdateBasicInfo`):
  signature becomes `name, detail, username, password *string`,
  `authTypes *[]rmsipauth.AuthType`, `allowedIPs *[]string`.
- `bin-common-handler/pkg/requesthandler/registrar_trunks.go`
  (`RegistrarV1TrunkUpdateBasicInfo`).
- `bin-registrar-manager/pkg/listenhandler/models/request/trunks.go`
  (`V1DataTrunksIDPut`).
- `bin-registrar-manager/pkg/listenhandler/v1_trunks.go`
  (`processV1TrunksIDPut`).
- `bin-registrar-manager/pkg/trunkhandler/trunk.go` (`Update`): same as
  extensions -- no signature change needed, its existing
  `map[trunk.Field]any` parameter and `SetMap`-based dbhandler call
  already do the right thing once the listenhandler stops always
  populating every key. One thing to verify explicitly (not assume): the
  unconditional `sip := res.GenerateSIPAuth(); ... h.db.SIPAuthUpdate(...)`
  block at the end of `Update` regenerates SIP auth from `res` -- the
  **freshly re-fetched, current** trunk row after the partial `SetMap`
  update, not from the caller's raw input. This means an omitted
  `auth_types`/`username`/`password`/`allowed_ips` correctly leaves
  `sipauth` derived from the unchanged stored values, not from a
  zero-valued caller input -- confirm this with a dedicated test (§6),
  since it is the one place in this Phase where correctness depends on
  re-fetch-after-partial-update ordering, not just "did the field map
  get the right keys."
- RST doc for trunks.
- Tests at every layer per §6.

**Explicitly out of scope:**
- `POST /extensions`, `POST /trunks` (create) -- same non-goal argument as
  PR #1291 §3: no prior value to preserve, every field is either supplied
  or defaults from scratch. Not touched.
- `square-admin` frontend -- per the roadmap's Phase-level default (no
  frontend changes bundled unless a phase's own design doc identifies a
  concrete behavior change). A grep of `square-admin`'s extension/trunk
  edit forms should be done during implementation to confirm whether they
  already send all fields unconditionally (mirroring PR #1291 §2's
  pattern) or already omit some -- if the latter, document the resulting
  behavior change the same way PR #1291 §2.1 did, but do not change
  frontend code in this PR regardless.
- `ps_endpoints`/`ps_aors` (Asterisk tables for extension identity/contact
  binding) -- extension `Update` only ever touches `ps_auths` (via
  `AstAuthUpdate`) for password rotation; `name`/`detail` are bin-manager-
  DB-only fields with no Asterisk-table mirror to worry about.
- Any change to `Extension`/`Trunk` `Create` validation rules (e.g.
  minimum password strength) -- this fix only changes whether a field is
  required to be *present* on PUT, not what values are valid once present.

## 3. Design: apply the established pattern (PR #1291 §4) to both endpoints

No new design is being invented here; this section states the
endpoint-specific instantiation of the 11-step pattern already defined in
the roadmap's §3, plus the two endpoint-specific verification items noted
in §2 above (the pre-existing correctness of `extensionHandler.Update`'s
password-presence check, and `trunkHandler.Update`'s
regenerate-from-refetched-row SIP auth behavior).

### 3.1 `V1DataExtensionsIDPut` / `V1DataTrunksIDPut` (wire DTOs)

```go
// V1DataExtensionsIDPut is v1 data type request struct for
// /v1/extensions/{id} PUT
//
// Every field is a pointer: nil means "leave the existing value
// unchanged", matching the OpenAPI-layer contract. See
// docs/plans/2026-09-12-registrar-put-partial-update-phase1-design.md.
type V1DataExtensionsIDPut struct {
	Name     *string `json:"name,omitempty"`
	Detail   *string `json:"detail,omitempty"`
	Password *string `json:"password,omitempty"`
}
```

```go
// V1DataTrunksIDPut is v1 data type request struct for
// /v1/trunks/{id} PUT
type V1DataTrunksIDPut struct {
	Name       *string             `json:"name,omitempty"`
	Detail     *string             `json:"detail,omitempty"`
	AuthTypes  *[]sipauth.AuthType `json:"auth_types,omitempty"`
	Username   *string             `json:"username,omitempty"`
	Password   *string             `json:"password,omitempty"`
	AllowedIPs *[]string           `json:"allowed_ips,omitempty"`
}
```

Note: the existing field is spelled `Authtypes` (lowercase `t`), not
`AuthTypes`. This design intentionally does NOT rename it while also
changing its type -- keep the existing casing (`Authtypes *[]sipauth.AuthType`)
to avoid bundling an unrelated rename into this fix. If a future cleanup
wants to fix the casing, do it as its own separate change so this PR's
diff stays focused on the pointer-semantics fix; a casing rename ripples
into every struct-literal reference (`bin-common-handler/pkg/requesthandler/
registrar_trunks.go`'s `Authtypes: authTypes` field, `V1DataTrunksPost`'s
same-named field in `trunks.go`, and any test fixture using the field
name) and is out of scope here.

`AuthTypes`/`AllowedIPs` are pointers-to-slices (`*[]T`), not bare slices,
so `nil` (omitted) is distinguishable from an explicit empty array `[]`
(which is a legitimate "clear the allowlist" / "clear auth types" value a
caller might send on purpose -- same reasoning as `mcpservers.secret`'s
"empty string is an explicit clear, not omission").

### 3.2 `processV1ExtensionsIDPut` / `processV1TrunksIDPut` (listenhandler, actual fix site)

```go
fields := map[extension.Field]any{}
if reqData.Name != nil {
	fields[extension.FieldName] = *reqData.Name
}
if reqData.Detail != nil {
	fields[extension.FieldDetail] = *reqData.Detail
}
if reqData.Password != nil {
	fields[extension.FieldPassword] = *reqData.Password
}

if len(fields) == 0 {
	// No-op PUT: return current state, reusing Get's existing
	// ErrNotFound mapping, per PR #1291 §4.1's rationale.
	tmp, err := h.extensionHandler.Get(ctx, id)
	...
}

tmp, err := h.extensionHandler.Update(ctx, id, fields)
```

Trunks mirror this exactly, with `AuthTypes`/`AllowedIPs` dereferenced
from `*[]T` only when non-nil.

### 3.3 `bin-api-manager` HTTP layer and `servicehandler` layer

Follow PR #1291 §4.2/§4.3 verbatim: `PutExtensionsId`/`PutTrunksId` stop
dereferencing OpenAPI-generated pointers to zero-value defaults and pass
them straight through; `ExtensionUpdate`/`TrunkUpdateBasicInfo` accept
pointers and pass them straight to the `requesthandler` layer, which
builds the wire DTO with pointers (§3.1).

### 3.4 OpenAPI spec changes

Schema field types do not change structurally (pointers are generated
automatically for non-required object properties by `oapi-codegen`) --
only `required:` is removed and `description` is updated per field,
mirroring PR #1291 §5's exact wording pattern:

```yaml
# extensions/id.yaml put.requestBody
schema:
  type: object
  description: >
    Every field is optional. Omitting a field leaves its current value
    unchanged; a full resend of every field is NOT required to update a
    single field (e.g. PUT { "name": "new name" } only renames the
    extension and leaves detail/password exactly as they were).
  properties:
    name:
      type: string
      description: "Omit to leave the current name unchanged."
    detail:
      type: string
      description: "Omit to leave the current detail unchanged."
    password:
      type: string
      description: "SIP registration password. Omit to leave the current password unchanged. Sending an explicit value (including empty string) replaces it -- see the Implementation Hint in the extension struct docs for the security implication of accidentally clearing this."
  # required: removed entirely
```

`trunks/id.yaml` follows the same shape for its six fields;
`allowed_ips`'s description gets the same "explicit empty array clears
the allowlist, distinct from omitting the field" callout `mcpservers`'
`auth_type` description used for its own zero-value-is-meaningful case.

### 3.5 RST docs

Locate the extension and trunk struct/overview RST files under
`bin-api-manager/docsdev/source/` (implementer should `grep -l
"RegistrarManagerExtension\|RegistrarManagerTrunk"
bin-api-manager/docsdev/source/*.rst` to find the exact filenames -- not
pre-guessed here to avoid stating a possibly-stale filename). Add an
Implementation Hint note per PR #1291 §5.1's pattern to each, stating the
omitted-means-unchanged contract and specifically flagging that
`password` (extensions) / `password`+`allowed_ips` (trunks) are
credential/security-relevant fields where accidental omission would
previously have caused silent breakage -- this fix is what prevents that.
Clean Sphinx rebuild (`cd bin-api-manager/docsdev && rm -rf build &&
python3 -m sphinx -M html source build`), commit `build/` with `git add
-f` per the mandatory RST-sync rule.

## 4. Why `Create` (`POST`) needs no change

Same argument as PR #1291 §3, restated for this domain:
`extensionHandler.Create`/`trunkHandler.Create` (confirmed via
`bin-registrar-manager/pkg/extensionhandler/extension.go` /
`pkg/trunkhandler/trunk.go` `Create` bodies) take plain
`name, detail, extension, password string` /
`name, detail, domainName string, authTypes []sipauth.AuthType, username,
password string, allowedIPs []string` parameters with no
"omitted vs explicitly empty" ambiguity to preserve -- there is no prior
row. Do not touch `POST` in this phase.

## 5. Interaction with existing behavior

### 5.1 `extensionHandler.Update`'s password/AstAuth branch becomes correct automatically

`extensionHandler.Update` (`pkg/extensionhandler/extension.go:234-245`,
unchanged by this fix) already does:

```go
if password, ok := fields[extension.FieldPassword]; ok {
	...AstAuthUpdate(ctx, auth)...
}
```

Today this branch always fires (the pre-fix listenhandler always inserts
`FieldPassword`, even as `""`), silently zeroing the Asterisk-side
password on every PUT that doesn't explicitly resend the current one.
After this fix, the branch only fires when the caller actually sent
`password` -- no code change to this function, but its observed behavior
changes because its caller (the listenhandler) now only populates the map
conditionally. This is the single highest-value fix in this phase and
must have a dedicated test (§6) proving the branch does NOT fire when
`password` is omitted.

### 5.2 `trunkHandler.Update`'s unconditional SIP-auth regeneration is safe, verify don't assume

`trunkHandler.Update` (`pkg/trunkhandler/trunk.go:168-204`, unchanged)
always regenerates and writes `sipauth` fields via
`res.GenerateSIPAuth()` where `res` is the row **re-fetched after** the
partial `TrunkUpdate` SQL write. Because the SQL write only touched the
keys present in `fields`, `res` reflects a mix of "just-changed" and
"still as before" columns correctly -- `GenerateSIPAuth()` operating on
that merged row is therefore already correct for partial updates with no
code change needed. State this explicitly as a design decision (not an
oversight) so a reviewer doesn't flag "why isn't sipauth regen also made
conditional" -- it doesn't need to be, because it always derives from the
authoritative current row, never from the caller's raw (possibly nil)
input.

## 6. Testing strategy

Mirrors PR #1291 §7's structure, instantiated per field for both
endpoints.

**`bin-registrar-manager/pkg/listenhandler/v1_extensions_test.go`
(existing file) and `v1_trunks_test.go` (existing file):**
- PUT body with only `{"name": "new name"}` -> `extensionHandler.Update`
  called with `fields` containing exactly `{FieldName: "new name"}`
  (assert exact map contents, not just no-error).
- PUT body with only `{"password": "newpass"}` -> `fields` contains
  exactly `{FieldPassword: "newpass"}` -- this is the direct regression
  test for §5.1's silent-AstAuth-wipe bug.
- PUT body with every field omitted -> `extensionHandler.Update` is NOT
  called (`Times(0)`); `extensionHandler.Get` IS called, returns current
  row.
- Trunks: same three shapes, plus a fourth case -- PUT body with only
  `{"allowed_ips": []}` (explicit empty array) -> `fields` contains
  `{FieldAllowedIPs: []string{}}` (explicit clear, distinguishable from
  omission, which must NOT populate this key at all).

**`bin-registrar-manager/pkg/extensionhandler/extension_test.go` /
`pkg/trunkhandler/trunk_test.go`:** extend existing `Test_Update`/
equivalent with a case asserting: when `fields` does not contain
`FieldPassword`, `dbAst.AstAuthUpdate` is NOT called (gomock `Times(0)`);
when it does, `AstAuthUpdate` IS called with the new password. For
trunks, add a case confirming `SIPAuthUpdate` is called with fields
derived from the *updated* row even when the field map only contained
`{FieldName: ...}` -- pins §5.2's "safe by construction" claim with an
actual test, not just documentation.

**`bin-api-manager/server/extensions_test.go` / `trunks_test.go`
(existing files, HTTP layer -- closes the gap PR #1291's own Round 1
review flagged for `mcpservers`):**
- Update both files' existing PUT tests (breaks compilation once the
  signature changes to pointers) to pass `&value` instead of `value`.
- Add: PUT body `{"name": "new name"}` only -> asserts
  `serviceHandler.ExtensionUpdate`/`TrunkUpdateBasicInfo` called with
  `name` non-nil, every other pointer nil.

**`bin-api-manager/pkg/servicehandler/extension_test.go` /
`trunk_test.go` and `bin-common-handler/pkg/requesthandler/
registrar_extensions_test.go` / `registrar_trunks_test.go`:** mechanical
signature-type updates for existing tests (pointer params), no new test
cases required at these pure-passthrough layers beyond confirming
build/existing assertions still pass.

**Revert-and-rerun requirement (standing practice):** for the
`password`-omitted case on both endpoints, confirm the new test fails
against pre-fix code (`git stash` the listenhandler change, rerun,
confirm FAIL, restore) before considering the suite complete.

**Codegen and mock regeneration (explicit step, not implied):**
- `bin-openapi-manager`: after editing `extensions/id.yaml`/`trunks/id.yaml`,
  run `go generate ./...` to regenerate `gens/models/gen.go`.
- `bin-api-manager`: run `go generate ./...` to regenerate
  `gens/openapi_server/gen.go` (the `PutExtensionsIdJSONBody`/
  `PutTrunksIdJSONBody` fields become pointers automatically once
  `required:` is removed from the spec -- confirm this by diffing the
  generated struct, the same way `PutMcpserversIdJSONBody` became pointer-
  typed in PR #1291's precedent). Update the `ServiceHandler` interface
  method signatures in `pkg/servicehandler/main.go`
  (`ExtensionUpdate`/`TrunkUpdateBasicInfo`) to the new pointer
  parameters, then regenerate `mock_main.go`
  (`mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod`,
  run from `pkg/servicehandler/`).
- `bin-registrar-manager`: run `go generate ./...` to regenerate any
  mocks depending on the changed `extensionHandler`/`trunkHandler`
  interfaces if their method signatures change (they should not, per
  §2/§5.1/§5.2 -- only the listenhandler's map-construction logic
  changes -- but confirm via `go generate` + `go build ./...` that no
  interface actually needed a signature change).
- `bin-common-handler`: no interface regen needed beyond the
  `requesthandler` method's own signature edit (it has no mock; it IS the
  RPC client).

## 7. Non-goals

- No retroactive remediation for extensions/trunks whose password may
  have already been silently wiped in production by the pre-fix
  behavior -- raise as a separate question to 대표님 if a specific
  customer complaint surfaces (mirrors PR #1291 §8, roadmap §6).
- No frontend changes bundled into this PR (see §2).
- No change to `Create`/`POST` validation or behavior (see §4).
- No new PATCH method -- stays PUT with optional-field semantics,
  consistent with the platform-wide convention this phase extends.

## 8. Round 1 review disposition

Independent review (`deleg_bdff1bb5`) verdict: APPROVE, with two LOW-severity
documentation-completeness notes folded in before Round 2:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | §3.1's proposed trunk DTO used `AuthTypes` (capital T), silently renaming the existing `Authtypes` field without flagging it as a rename or noting the ripple to `registrar_trunks.go`'s struct literal | LOW | §3.1: reverted to the existing `Authtypes` casing, added a note explaining why the casing is intentionally NOT changed in this fix (keeps the diff focused; a casing cleanup is a separate future change) |
| 2 | §6 implied but never explicitly stated that `ServiceHandler` interface signatures and `mock_main.go` need regeneration, and didn't call out the openapi/oapi-codegen regen step the way PR #1291 did | LOW | §6: added an explicit "Codegen and mock regeneration" subsection listing every `go generate` step across all four touched services |

All core technical claims (dbhandler partial-update behavior, the 6-layer
call chain, the password/AstAuth branch, the SIP-auth re-fetch ordering,
OpenAPI required-field lists, DTO struct definitions, Create non-goal
reasoning, section cross-references) were independently verified against
the repository and found accurate; no changes needed to those sections.
