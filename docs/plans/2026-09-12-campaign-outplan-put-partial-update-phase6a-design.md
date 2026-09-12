# Phase 6a design: PUT partial-update for campaigns and outplans/dial_info

Status: DRAFT (Round 0)
Author: CPO, per roadmap `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md` §4 Phase 6.
Reference pattern: `docs/plans/2026-09-12-mcp-server-put-partial-update-design.md` (PR #1291,
merged `81a9a6b98`); same technical pattern applied verbatim by every prior
phase (1-5), not re-derived here — see roadmap §3 for the canonical 11-step
pattern.

## 1. Scope

Two endpoints, both owned by `bin-campaign-manager`, landing in a single PR
(same repo, no one-PR-per-repo-boundary split needed):

- `PUT /campaigns/{id}` — `name`, `detail`, `type`, `service_level`,
  `end_handle` (5 fields, all currently `required`).
- `PUT /outplans/{id}/dial_info` — `source`, `dial_timeout`, `try_interval`,
  `max_try_count_0..4` (8 fields, all currently `required`).

Per roadmap §2 rows 11/12, both are MEDIUM risk: `end_handle` (campaign)
controls live campaign completion behavior; the dial-timing fields
(outplan) control live autodialer pacing. Neither carries a
credential/webhook field (that class was Phase 1/2), but both can silently
alter in-flight dialing behavior if a client omits a field expecting it to
stay unchanged and the current full-replace endpoint zeroes it instead.

Out of scope (per roadmap, not re-litigated here): `PUT /outplans/{id}`
(`name`/`detail` only, Phase 8, deferred, cosmetic); every other
`campaigns`/`outplans` single-field PUT (`status`, `service_level`,
`actions`, `resource_info`, `next_campaign_id`) — all structurally exempt
per roadmap §3.1/§3.2, not touched by this phase.

## 2. Call chain (verified against current source, not assumed)

### 2.1 `PUT /campaigns/{id}`

1. `bin-openapi-manager/openapi/paths/campaigns/id.yaml` — `put.requestBody`
   schema has `required: [name, detail, type, service_level, end_handle]`
   (confirmed, lines 62-67).
2. `bin-api-manager/server/campaigns.go:173` `PutCampaignsId` — binds
   `openapi_server.PutCampaignsIdJSONBody`, calls
   `h.serviceHandler.CampaignUpdateBasicInfo(ctx, a, target, req.Name,
   req.Detail, cmcampaign.Type(req.Type), req.ServiceLevel,
   cmcampaign.EndHandle(req.EndHandle))` (line 203) — passes plain values,
   not pointers, today; no `nil`-dereference-to-zero-value bug exists yet
   only because `req.Name` etc. are already plain (non-pointer) generated
   types under the current `required` schema. Once the schema goes
   optional, `oapi-codegen` regenerates these fields as `*string`/`*int`/
   `*cacampaign.Type`/`*cacampaign.EndHandle` and this call site becomes the
   fix site's caller — it must pass pointers through unchanged.
3. `bin-api-manager/pkg/servicehandler/campaigns.go:202`
   `CampaignUpdateBasicInfo(ctx, a, id, name, detail, campaignType,
   serviceLevel, endHandle)` — `a.IsDirect()` check, then `campaignGet`
   (for `CustomerID`), then `hasPermission`, THEN calls
   `h.reqHandler.CampaignV1CampaignUpdateBasicInfo(...)` (line 240) with the
   same 5 values passed straight through. No transformation/validation of
   the 5 fields happens at this layer today (unlike team's
   `validateTeam`) — confirmed by reading the full function body, lines
   200-248.
4. `bin-common-handler/pkg/requesthandler/campaign_campaigns.go:153`
   `CampaignV1CampaignUpdateBasicInfo` — marshals into
   `carequest.V1DataCampaignsIDPut{Name, Detail, Type, ServiceLevel,
   EndHandle}` (all plain fields, `omitempty` absent on `Type` field
   specifically — confirmed in
   `bin-campaign-manager/pkg/listenhandler/models/request/campaigns.go:40-46`,
   `Type campaign.Type \`json:"type"\`` has no `omitempty` tag, the other 4
   fields do) and RPCs to campaign-manager.
5. `bin-campaign-manager/pkg/listenhandler/campaigns.go:224` — reads
   `request.V1DataCampaignsIDPut`, calls
   `h.campaignHandler.UpdateBasicInfo(ctx, id, req.Name, req.Detail,
   req.Type, req.ServiceLevel, req.EndHandle)` (pass-through, to be
   confirmed exact line at implementation time — the struct read is
   confirmed, the exact handler call line is a mechanical one-line read
   away and will be verified again before patching, not assumed here).
6. `bin-campaign-manager/pkg/campaignhandler/campaign.go:246`
   `UpdateBasicInfo(ctx, id, name, detail, campaignType, serviceLevel,
   endHandle)` — **this is the actual fix site**. Currently builds an
   unconditional 5-key `fields := map[campaign.Field]any{...}` and always
   calls `h.db.CampaignUpdateBasicInfo(...)` then re-fetches via
   `h.db.CampaignGet` and unconditionally calls
   `h.notifyHandler.PublishWebhookEvent(...)` (lines 266-277) — no no-op
   short-circuit exists today.
7. `bin-campaign-manager/pkg/dbhandler/campaign.go:358`
   `CampaignUpdateBasicInfo(ctx, id, name, detail, campaignType,
   serviceLevel, endHandle) error` — thin wrapper that always builds the
   same 5-key map and calls `h.CampaignUpdate(ctx, id, fields)` (line 375).
   `CampaignUpdate` (line 327) already has a `len(fields) == 0` guard
   (returns nil without touching the DB) — this guard is currently
   unreachable via this call path (the wrapper always populates all 5
   keys) but will become load-bearing once step 6's fix site builds the
   map conditionally and can produce an empty map on an all-omitted
   request routed straight through to `CampaignUpdate`. **Design decision**
   (see §3 item 7): keep `dbhandler.CampaignUpdateBasicInfo`'s own
   signature unchanged (still takes 5 plain values, still builds an
   unconditional 5-key map) and do NOT push pointer-awareness down to the
   dbhandler layer — instead the business-handler fix site (step 6,
   `campaignhandler.UpdateBasicInfo`) takes 5 pointers, builds its own
   conditional map, and calls `h.db.CampaignUpdate(ctx, id, fields)`
   directly (bypassing `dbhandler.CampaignUpdateBasicInfo`'s
   always-5-keys wrapper entirely) when any field is set, or short-circuits
   to `h.db.CampaignGet` when all 5 are nil. This mirrors the
   conferences/Phase 5a pattern exactly (business handler owns the
   conditional map, dbhandler-level convenience wrappers that assume full
   replacement are bypassed, not made pointer-aware) and avoids
   duplicating conditional-map logic at two layers.
8. CLI: `bin-campaign-manager/cmd/campaign-control/main.go:300`
   `runUpdateBasicInfo` — unconditionally reads all 5 `viper` flags
   (`name` required via manual `if name == ""` check, others default via
   flag defaults, e.g. `service-level` defaults to `100`) and calls
   `handler.UpdateBasicInfo(...)` with all 5 values every time. **This CLI
   command has no flag-conditional construction today and, like Phase 5a's
   conference-control, must gain one** — see §3 item 8.

### 2.2 `PUT /outplans/{id}/dial_info`

1. `bin-openapi-manager/openapi/paths/outplans/id_dial_info.yaml` —
   `required: [source, dial_timeout, try_interval, max_try_count_0..4]`
   (8 fields, confirmed lines 44-52).
2. `bin-api-manager/server/outplans.go:194` `PutOutplansIdDialInfo` — binds
   `PutOutplansIdDialInfoJSONBody`, converts `source` via
   `ConvertCommonAddress(req.Source)` (line 222, **note**: this returns a
   plain `address.Address`, not a pointer, even though the call site then
   takes its address with `&source` — confirmed line 224 — meaning
   `ConvertCommonAddress` must be checked at implementation time for
   nil-safety once `req.Source` becomes an optional pointer; today
   `req.Source` is a required plain-or-pointer generated struct and
   `ConvertCommonAddress` presumably handles it unconditionally). Calls
   `h.serviceHandler.OutplanUpdateDialInfo(ctx, a, target, &source,
   req.DialTimeout, req.TryInterval, req.MaxTryCount0..4)` (line 224).
3. `bin-api-manager/pkg/servicehandler/outplans.go:229`
   `OutplanUpdateDialInfo` — `a.IsDirect()` check, `outplanGet` (for
   `CustomerID`), `hasPermission`, then
   `h.reqHandler.CampaignV1OutplanUpdateDialInfo(...)` (line 266), pure
   pass-through, no field-specific validation.
4. `bin-common-handler/pkg/requesthandler/campaign_outplans.go:162`
   `CampaignV1OutplanUpdateDialInfo` — marshals into
   `carequest.V1DataOutplansIDDialsPut{Source, DialTimeout, TryInterval,
   MaxTryCount0..4}` — **note**: `Source` field is already `*address.Address`
   in the RPC signature (line 165, `source *address.Address`) — it is
   already nilable at this hop even though the OpenAPI schema requires it;
   this is a pre-existing asymmetry, not introduced by this phase, and
   just needs to keep flowing through unchanged (a genuinely-nil pointer at
   this hop after the phase's fix would mean "field omitted", which is the
   desired new semantics for free at this one field).
5. `bin-campaign-manager/pkg/listenhandler/outplans.go:266` — reads
   `request.V1DataOutplansIDDialsPut`, calls into
   `outplanHandler.UpdateDialInfo` (exact pass-through to be confirmed at
   implementation time, mechanical).
6. `bin-campaign-manager/pkg/outplanhandler/outplan.go:186`
   `UpdateDialInfo` — **fix site**. Currently builds an unconditional
   8-key `fields := map[outplan.Field]any{...}` (verified against
   `dbhandler.OutplanUpdateDialInfo`, line 303-312, since
   `outplanhandler.UpdateDialInfo` calls `h.db.OutplanUpdateDialInfo(ctx,
   id, source, dialTimeout, ...)` at line 212 with all 8 plain values,
   which itself re-wraps into the same 8-key map at the dbhandler layer)
   and always calls the DB write. Same design decision as campaigns
   (§2.1 step 7): the business-handler fix site takes 8 pointers, builds
   its own conditional map, and calls `h.db.OutplanUpdate(ctx, id,
   fields)` directly — bypassing `dbhandler.OutplanUpdateDialInfo`'s
   always-8-keys wrapper — rather than pushing pointer-awareness into the
   dbhandler convenience method.
7. `bin-campaign-manager/pkg/dbhandler/outplan.go:250` `OutplanUpdate`
   already has the `len(fields) == 0` guard (line 251-253), same shape as
   `CampaignUpdate` — becomes load-bearing once step 6 is fixed.
8. CLI: no `outplan-control`/dial-info CLI subcommand exists in
   `bin-campaign-manager/cmd` (confirmed: `search_files` for
   `outplan`/`dial` inside `bin-campaign-manager/cmd` returned zero
   matches) — **no CLI changes needed for this endpoint**, unlike
   campaigns.

## 3. Fixes to apply (per roadmap §3's 11-step pattern, endpoint-specific notes only)

1. **OpenAPI**: remove `required:` block on both
   `paths/campaigns/id.yaml` (put) and
   `paths/outplans/id_dial_info.yaml` (put); update each field's
   `description` to state "Omit to leave unchanged."
2. **`bin-api-manager/server/campaigns.go` `PutCampaignsId`**: pass
   `req.Name`/`req.Detail`/`req.Type`/`req.ServiceLevel`/`req.EndHandle`
   straight through as the regenerated pointer types (no
   dereference-to-zero-value).
3. **`bin-api-manager/server/outplans.go` `PutOutplansIdDialInfo`**: same
   pass-through for `req.Source`/`req.DialTimeout`/`req.TryInterval`/
   `req.MaxTryCount0..4`; re-verify `ConvertCommonAddress`'s nil-handling
   for a nil `req.Source` at implementation time (it may already return a
   zero-value `address.Address{}` for a nil input, in which case the
   caller must additionally check `req.Source != nil` before calling
   `ConvertCommonAddress` and taking its address, to avoid silently
   sending a non-nil `*address.Address{}` pointer to an "omitted" field —
   this is the one endpoint-specific gotcha in this phase, flagged
   explicitly so it isn't missed the way a similar per-field gotcha nearly
   was in Phase 5b's flow-ID 3-state logic).
4. **`bin-api-manager/pkg/servicehandler/campaigns.go`
   `CampaignUpdateBasicInfo`**: accept 5 pointers, pass through unchanged
   (auth/permission ordering already correct, confirmed §2.1 step 3 — no
   dereference before the permission check needed since this layer never
   dereferences at all, it's pure pass-through).
5. **`bin-api-manager/pkg/servicehandler/outplans.go`
   `OutplanUpdateDialInfo`**: accept pointers for all 8 fields (`source`
   already effectively nilable per §2.2 step 4's note; the 7 int fields
   become `*int`), pass through unchanged.
6. **`bin-common-handler/pkg/requesthandler/campaign_campaigns.go`
   `CampaignV1CampaignUpdateBasicInfo`** and
   **`campaign_outplans.go` `CampaignV1OutplanUpdateDialInfo`**: accept
   pointers, marshal into the request DTOs as pointers (mandatory per
   roadmap §3 item 4 — skipping this reintroduces the bug one hop later).
7. **`bin-campaign-manager/pkg/listenhandler/models/request/campaigns.go`
   `V1DataCampaignsIDPut`** and **`outplans.go`
   `V1DataOutplansIDDialsPut`**: fields become pointers with `omitempty`.
   Note `V1DataCampaignsIDPut.Type` currently lacks `omitempty` (§2.1 step
   4) — this must be added as part of the pointer conversion (a pointer
   field's `omitempty` tag is what makes Go's `encoding/json` skip a truly
   nil pointer on marshal; without it a nil pointer marshals as JSON
   `null`, which is fine for RabbitMQ-hop DTOs since the receiving side
   unmarshals `null` into a nil pointer too — but keeping `omitempty`
   consistent across all fields in the same struct avoids an inconsistent
   convention within one type).
8. **`bin-campaign-manager/pkg/listenhandler/campaigns.go`** and
   **`outplans.go`**: pass pointers through unchanged to the business
   handler.
9. **`bin-campaign-manager/pkg/campaignhandler/campaign.go`
   `UpdateBasicInfo`** (fix site): change signature to 5 pointers; build
   `fields := map[campaign.Field]any{}` conditionally
   (`if name != nil { fields[campaign.FieldName] = *name }`, etc.);
   `if len(fields) == 0 { return h.Get(ctx, id) }` (reusing the existing
   `Get`/`ErrNotFound` mapping per roadmap §3 item 8); else call
   `h.db.CampaignUpdate(ctx, id, fields)` directly (bypassing
   `dbhandler.CampaignUpdateBasicInfo`, per §2.1 step 7's design decision),
   then `h.db.CampaignGet` + `PublishWebhookEvent` as today.
10. **`bin-campaign-manager/pkg/outplanhandler/outplan.go`
    `UpdateDialInfo`** (fix site): same conditional-map treatment for all
    8 fields, calling `h.db.OutplanUpdate(ctx, id, fields)` directly
    (bypassing `dbhandler.OutplanUpdateDialInfo`), no-op short-circuits to
    `h.Get(ctx, id)`.
11. **`bin-campaign-manager/cmd/campaign-control/main.go`
    `runUpdateBasicInfo`**: convert to flag-conditional construction —
    `viper.IsSet("name")`/`IsSet("detail")`/`IsSet("type")`/
    `IsSet("service-level")`/`IsSet("end-handle")` gate each pointer,
    mirroring the `--data`-flag-addition precedent from Phase 5a
    (conference-control) and the pre-existing `viper.IsSet` pattern from
    number-control. The existing `if name == ""` hard-required check on
    `name` must be removed (name becomes genuinely optional, matching the
    OpenAPI schema change) — confirm no other caller relies on `name`
    being mandatory via this CLI path before removing the check.
12. OpenAPI `description` fields + RST doc update
    (`bin-api-manager/docsdev/source/campaign_struct_campaign.rst`,
    `outplan_struct_outplan.rst` or equivalent — exact filename to be
    confirmed against the existing RST tree at implementation time) with
    an Implementation Hint per the established convention.
13. Tests per roadmap §3 item 11 at every layer, plus the
    revert-and-rerun proof for at least `end_handle` (campaign, MEDIUM's
    highest-risk field) and `source`/`dial_timeout` (outplan).

## 4. Non-goals

- `PUT /outplans/{id}` (`name`/`detail` only) is NOT in this phase — it is
  Phase 8, LOW risk, deferred per 오버엔지니어링 지양 (no concrete trigger
  yet).
- No change to `campaign-control`'s other subcommands
  (`update-status`) — only `update-basic-info` is touched.
- No change to `CampaignUpdateResourceInfo`/`UpdateNextCampaignID`/
  `UpdateServiceLevel`/`UpdateActions` — these back different, already
  single-field or intentionally-atomic endpoints (roadmap §3.1/§3.2),
  untouched by this phase.
- `dbhandler.CampaignUpdateBasicInfo` and `dbhandler.OutplanUpdateDialInfo`
  (the always-full-map convenience wrappers) are left in place, not
  deleted — they remain dead code reachable only if some future caller
  needs an always-full-replace path; deleting them is out of scope
  (touches more surface than needed to fix the bug, no concrete trigger to
  remove them beyond this phase's goal). Flagged for optional follow-up
  cleanup, not scheduled.

## 5. Risk / testing notes

- Campaign's `end_handle` and `type` are enum-typed
  (`cacampaign.EndHandle`, `cacampaign.Type`) whose zero value
  (empty string) is not a valid enum member today (confirmed: `Type` has
  no `omitempty` in the wire DTO, meaning the current required-field
  contract already assumes a non-empty string is always sent) — this
  matches the same "zero-value looks invalid, fails loud not silent"
  risk shape roadmap §2 uses to justify MEDIUM rather than HIGH for this
  endpoint; the fix still closes the omission-corrupts-state gap for
  `name`/`detail`/`service_level` which have valid-looking zero values
  (empty string / 0).
- Outplan's 8 fields are all either an address struct or plain ints with
  valid-looking zero values (`dial_timeout: 0` is a real, if degenerate,
  timeout) — this is the more classic silent-wipe shape and is why this
  endpoint is rated MEDIUM on the "wrong/omitted values change live
  autodialer pacing" risk sentence in roadmap §2 row 12.

## 6. Round 1 review disposition

Independent review (`deleg_095290d1` task 1) verdict: APPROVE. Reviewer
independently re-derived every §2 call-chain claim against current
source (file paths, line numbers, function signatures for both
`PUT /campaigns/{id}` and `PUT /outplans/{id}/dial_info`), confirmed no
CLI subcommand exists for outplan dial_info, confirmed
`campaign-control`'s exact flag names and lack of `IsSet` gating,
confirmed the dbhandler-bypass design decision (§2.1 step 7 / §2.2 step
6) is safe via a full caller-graph grep (each convenience wrapper and
each business-handler method has exactly one caller), confirmed the
`V1DataCampaignsIDPut.Type` missing-`omitempty` gap, and confirmed the
`ConvertCommonAddress` nil-handling concern (§3 item 3) is a real,
correctly-identified gotcha (the function takes a plain, non-pointer
`CommonAddress` today and will need a `req.Source != nil` guard at the
call site once the field becomes optional). No findings required a
change; only trivial (1-3 line) line-number drift was noted, which does
not affect any technical conclusion and was left as-is.

