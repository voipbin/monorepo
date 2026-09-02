# Implementation Plan: VOIP-1443 — campaignHandler.Delete() silent failures

Full technical analysis, all traced source citations, and defect rationale:
[2026-09-02-voip-1443-campaign-delete-silent-failures-design.md](2026-09-02-voip-1443-campaign-delete-silent-failures-design.md)
(the issue-analysis document that went through 11 independent review rounds before this plan was
written; all file:line citations below are already verified there — this plan translates that
analysis into an ordered task list, it does not re-derive it).

## Scope

Two defects in `bin-campaign-manager/pkg/campaignhandler/campaign.go`'s `Delete()`:
1. `FlowV1FlowDelete` failure is silently swallowed (log-only, no metric).
2. Deleting a non-stopped campaign returns `(nil, nil)`, surfacing as a misleading HTTP 200.

Explicitly out of scope: the existing 10,000-flow production backlog for the shared test
customer (data cleanup, separate authorization), and any alerting-rule/reconciliation follow-up
(filed as VOIP-1444). Admin-console delete-UX follow-up filed as VOIP-1445.

## Tasks

1. **`bin-campaign-manager/pkg/campaignhandler/campaign.go`**
   - Defect 1: on the `FlowV1FlowDelete` error branch, classify and increment
     `promCampaignFlowDeleteFailedTotal.WithLabelValues(reason).Inc()`:
     `reason = "not_found"` if `stderrors.Is(err, requesthandler.ErrNotFound)`, else
     `reason = "error"`. Import `monorepo/bin-common-handler/pkg/requesthandler` if not already
     imported in this file (already imported at package level via `main.go`, and
     `h.reqHandler` is typed `requesthandler.RequestHandler` — confirm import path in this
     specific file during implementation). Also add `flow_id` (`res.FlowID`) and `customer_id`
     (`res.CustomerID`) to the existing `log.Errorf("Could not delete the flow. err: %v", err)`
     call's logger fields, so the log line is actionable per-flow once the counter has flagged
     that a leak occurred (design-review addition).
   - Defect 2: replace `return nil, err` (the `c.Status != campaign.StatusStop` branch) with
     `return nil, cerrors.FailedPrecondition(commonoutline.ServiceNameCampaignManager, "CAMPAIGN_NOT_STOPPED", "The campaign must be stopped before it can be deleted.")`,
     matching the existing `Get()` pattern at lines 174-178 in the same file.
2. **`bin-campaign-manager/pkg/campaignhandler/main.go`**: declare and
   `prometheus.MustRegister` a new `CounterVec` named `promCampaignFlowDeleteFailedTotal`
   (Prometheus metric name: `campaign_manager_campaign_flow_delete_failed_total`), labels
   `[]string{"reason"}`, following the existing `promCampaignCreateTotal` `init()` pattern
   verbatim (same file, same init block style). Immediately after `MustRegister`, pre-initialize
   both label series so they read `0` (not "No data") from process start:
   `promCampaignFlowDeleteFailedTotal.WithLabelValues("not_found")` and
   `.WithLabelValues("error")` (design-review round 2 requirement — otherwise the dashboard panel
   in item 9 can't distinguish "healthy, zero failures" from "metric never registered/scraped").
3. **`bin-campaign-manager/pkg/campaignhandler/campaign_test.go`**: extend `Test_Delete`
   (currently happy-path only) with:
   - a case where `FlowV1FlowDelete` mock returns a `requesthandler.ErrNotFound`-wrapped error —
     assert `Delete()` still returns success, and assert the counter's `reason="not_found"`
     series incremented (via `testutil.ToFloat64`, before/after delta — pattern:
     `bin-ai-manager/pkg/messagehandler/event_test.go`). **Note**: since both label series are
     now pre-initialized (task 2), `ToFloat64` must be called on the specific
     `promCampaignFlowDeleteFailedTotal.WithLabelValues("not_found")` child `Counter`, not on the
     `CounterVec` itself — passing a multi-series `CounterVec` to `ToFloat64` panics.
   - a case where `FlowV1FlowDelete` mock returns a generic error — assert the
     `reason="error"` series incremented instead (same per-label-child `ToFloat64` requirement).
   - a case where the campaign's `Status != StatusStop` — assert a non-nil
     `*cerrors.VoipbinError` with reason `CAMPAIGN_NOT_STOPPED` is returned (not `(nil, nil)`).
4. **`bin-openapi-manager/openapi/paths/campaigns/id.yaml`**: add
   `'409': $ref: '#/components/responses/Conflict'` to the `delete:` operation's `responses:`
   block (currently 200/400/401/403/404/500).
5. **Codegen**: `cd bin-openapi-manager && go generate ./...`, then
   `cd ../bin-api-manager && go generate ./...` (OpenAPI-First order, per
   `bin-api-manager/CLAUDE.md`). Commit the regenerated `gens/` artifacts in both services.
6. **`bin-api-manager/docsdev/source/restful_api_errors.rst`**: add a `CAMPAIGN_NOT_STOPPED`
   (409) row to the "Campaign Reasons" table (currently 3 rows, all 404). No change needed to
   the section's existing "state-transition operations are idempotent" intro sentence (`DELETE`
   is not a state-transition endpoint).
7. **RST rebuild**: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html
   source build`, then `git add -f bin-api-manager/docsdev/build/`.
8. **`bin-campaign-manager/docs/operations.md`**: add `campaign_manager_campaign_flow_delete_failed_total`
   to the existing "Prometheus Metrics" table, with its `reason` label values documented.
9. **`monitoring/grafana/dashboards/campaign-manager.json`**: add a panel for the new counter
   (split by `reason`), alongside the existing `campaign_manager_*` panels.
10. **Verification** (per root `bin-campaign-manager` and `bin-openapi-manager`/`bin-api-manager`
    CLAUDE.md, run in this order):
    - `bin-openapi-manager`: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
    - `bin-api-manager`: same five-step workflow (picks up the regenerated `gens/openapi_server`)
    - `bin-campaign-manager`: same five-step workflow (this is where the actual logic + new
      tests live — `go test ./pkg/campaignhandler/... -run Test_Delete -v` should be run
      explicitly in addition to the full suite to confirm the new cases pass)
11. Commit format per root CLAUDE.md: title `VOIP-1443-campaign-delete-silent-failures`
    (matches branch name), body with `bin-<service>:`-prefixed bullets per affected service,
    plus `bin-openapi-manager:`/`bin-api-manager:` doc-generation bullets.
12. **Post-deploy verification** (after this PR merges and deploys, not part of the PR itself —
    design-review round 2 requirement): (a) confirm both
    `campaign_manager_campaign_flow_delete_failed_total{reason="not_found"|"error"}` series read
    `0` in Prometheus, proving the metric registered and is being scraped; (b) check the next
    scheduled `api-validator` run's campaign-delete-on-non-stopped-campaign assertions (if any) —
    expect a shift from the old bogus 200 to 409, which is the fix working as intended, not a
    regression. Rollback trigger: revert (not forward-fix) only if a first-party client is found
    depending on the old 200 response.

## Acceptance criteria

- `DELETE /v1.0/campaigns/{id}` on a non-stopped campaign returns 409 with a proper
  `CAMPAIGN_NOT_STOPPED` error envelope instead of HTTP 200 with an all-zero-value body.
- `DELETE /v1.0/campaigns/{id}` on a stopped campaign still returns 200 and still deletes the
  campaign, unchanged, whether or not the backing flow's deletion succeeds (no regression to
  existing success-path behavior).
- A flow-delete failure during campaign delete increments
  `campaign_manager_campaign_flow_delete_failed_total{reason=...}`, correctly split between
  `not_found` and `error`.
- New and existing `Test_Delete` cases pass; full `go test ./...` and `golangci-lint run` pass
  for all three touched services.
- OpenAPI spec, generated code, RST docs (source + rebuilt HTML), `docs/operations.md`, and the
  Grafana dashboard are all updated consistently — no stale cross-reference introduced.

## Verify tasks (explicit, per CLAUDE.md template)

- [ ] `go test ./...` passes in `bin-campaign-manager`, `bin-api-manager`, `bin-openapi-manager`
- [ ] `golangci-lint run -v --timeout 5m` passes in all three
- [ ] Manual/table-driven test confirms both new `Test_Delete` cases and the metric-delta
      assertions
- [ ] `git diff` review confirms `gens/` regeneration only touched the expected campaign
      delete-response shape, no unrelated diff
- [ ] Sphinx rebuild is a clean `rm -rf build` rebuild, not incremental
