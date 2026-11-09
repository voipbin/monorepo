# Restore route priority ordering (bin-route-manager)

## 1. Issue analysis

Verified against origin/main:

- `dbhandler.RouteList` orders by `tm_create DESC, id DESC`. The earlier `order by priority asc` was removed in VOIP-1190 (commit 1bb2e344d). No priority sort exists anywhere in the service.
- `routeHandler.ListByTarget` fetches target-specific routes and `target=all` routes separately (limit 1000 each), keeps the target-specific ones first and appends the `all` routes whose provider is not already present. It does not sort.
- `routeHandler.DialrouteList` returns customer routes followed by default-customer routes (providers already present are skipped). The `route_provider_ids` override builds synthetic routes in the given order and returns early.
- `RouteList` backs `routeHandler.ListByCustomerID` (list API, `GET /routes`, page token is `tm_create` based) and `ListByTarget`. The list API ordering must not change. `ListByTarget` callers: `DialrouteList` and the `route-control` CLI (`route list-by-target`, `route dialroute-list`). `providercallhandler` uses synthetic routes and is not affected.
- bin-call-manager calls only `RouteV1DialrouteList`, stores the returned array in the call and advances by array index (`getNextDialroute`), so the returned order is the attempt order. Calls already in progress keep their stored list.
- Public docs (`route_overview.rst`, `route_struct_route.rst`, `route_tutorial.rst`) say routes are tried in priority order; the code does not do that.
- `route_routes.priority` is nullable. `Route.Priority` is a Go `int` and `ScanRow` reads NULL as 0, so NULL cannot be told apart after scanning. The API always stores an integer.
- Known existing behaviours kept as is: an error while reading the `all` routes in `ListByTarget` is only logged and the target-specific list is returned; each query is limited to 1000 rows.
- Internal docs: `bin-route-manager/docs/domain.md` (lines 37, 53, 60), `docs/operations.md` (manual `ORDER BY priority` SQL puts NULL first, to be aligned), `docs/architecture.md`, `bin-route-manager/CLAUDE.md`. List API consumers: api-manager `servicehandler` `RouteGets` and `RouteGetsByCustomerID`. Public rst: `route_overview.rst`, `route_struct_route.rst`, `route_tutorial.rst`, `provider_overview.rst` (route priority mention).

## 2. Design

Rule: attempt order is ascending `priority` (lower = tried first), NULL last. Ties keep the current order (`tm_create DESC, id DESC`). The customer-before-default merge in `DialrouteList` is unchanged.

NULL handling: a new dbhandler read method `RouteListForDial(ctx, limit, filters)` selects `COALESCE(priority, 2147483647) AS priority` in place of the plain column and orders by `COALESCE(route_routes.priority, 2147483647) ASC, tm_create DESC, id DESC` (table-qualified so it does not depend on alias resolution). It keeps the `tm_delete IS NULL` condition and drops the `tm_create < token` condition (no pagination). The select list is built from `GetDBFields` by replacing the entry equal to `priority` at the same index; if it is not found the method returns an error, so a model change cannot silently shift columns. NULL therefore becomes the largest priority value and sorts last, without changing the model type (`*int` was rejected: wide blast radius). `RouteList` (list API) is untouched, so pagination and the API output are unchanged. A NULL row is returned with `Priority` 2147483647 instead of 0; consumers (bin-call-manager advances by array index and never reads `.Priority`; `route-control` prints it) are not affected. A stored value of exactly 2147483647 ties with NULL, which is acceptable. Because the SQL order applies before `LIMIT 1000`, truncation drops the lowest-priority routes, not the highest.

Placement: `ListByTarget` switches to `RouteListForDial` for both queries, then merges (target-specific first, `all` routes appended when the provider is not present) and applies `sort.SliceStable` by `Priority` on a new slice, so target and `all` are ordered jointly and tie order stays "target-specific first, then all, each in SQL order". De-duplication runs before sorting, so the winner on provider overlap does not change. `DialrouteList` needs no sort: both of its sources come from `ListByTarget`, and customer routes stay ahead of default routes by design. The ignored `all` query error keeps its behaviour: the sorted target list is returned.

Not changed: `RouteList` / list API, synthetic routes, schema (no migration, no flag).

Duplicate handling note: duplicates inside the `all` list were previously resolved by SQL order (newest wins); now the lower priority number wins. Duplicates inside the target-specific list are still not removed (unchanged).

Docs: ascending priority; equal priority = most recently created first; customer routes before default routes; a route without a stored priority is tried last. Update the internal docs and the rst files listed above, rebuild the HTML (clean) and force-add `build/`.

## 3. Verification plan

- dbhandler (sqlite): `RouteListForDial` orders ascending, NULL last, ties by `tm_create DESC, id DESC`, filters work, NULL row returns the max value.
- routehandler `ListByTarget`: ascending sort; tie order with 30+ mixed items (catches `Slice` instead of `SliceStable`); negative and zero priorities; target vs all tie keeps target first; overlap dedup (target provider X priority 9 vs all provider X priority 1 keeps the target one); `all` query error returns the sorted target list; input slice not required to be reused.
- `DialrouteList`: default route with lower priority still after customer routes; provider overlap keeps the customer route; synthetic route path unchanged.
- `ListByCustomerID` still passes through the SQL order (regression).
- Existing `route_merge_test.go` expectations move from `RouteList` to `RouteListForDial`; mocks regenerated. NULL rows in tests are made with raw INSERT / `UPDATE ... SET priority=NULL`. dbhandler sqlite cases include equal priority + equal `tm_create` (id DESC) and a 1001-row case proving the lowest priority values survive `LIMIT`. `ListByCustomerID` test expects only `RouteList` (never `RouteListForDial`). `target=all` with identical results from both queries is de-duplicated.
- Mutation checks (also: drop the tm_create/id tie keys, drop only the ORDER BY COALESCE, make `ListByTarget` call `RouteList`): remove sort, reverse comparison, `SliceStable` -> `Slice`, drop COALESCE must each fail a test.
- Full verification workflow, `scripts/check-*.sh`, docs HTML rebuild.
