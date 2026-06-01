# Fix cross-customer email leak in `GET /emails`

- **Date:** 2026-06-02
- **Issue:** [#956](https://github.com/voipbin/monorepo/issues/956)
- **Branch:** `NOJIRA-Fix-email-list-customer-filter`
- **Status:** Approved design

## Problem

`GET /v1.0/emails` (list) returns email records belonging to customers other than
the authenticated account. When one of those leaked IDs is then used in
`GET /v1.0/emails/{id}`, the single-item endpoint correctly returns
`403 PERMISSION_DENIED`. This is a data-isolation violation and causes the
api-validator CI test `test_get_email_if_exists` to fail on every run.

## Root cause

The bug is isolated to a single handler:
`bin-email-manager/pkg/listenhandler/v1_emails.go` → `v1EmailsGet` (line 30).

End-to-end flow today:

1. **api-manager** (`bin-api-manager/pkg/servicehandler/email.go:77`) correctly builds
   filters `{customer_id: <auth customer>, deleted: false}`, type-converts them via
   `convertEmailFilters` → `ConvertMapToTypedMap`, and passes them to the request handler.
2. **request handler** (`bin-common-handler/pkg/requesthandler/email_emails.go:20`)
   marshals the filters to JSON and sends them in the **request body** of the RPC.
   The URI carries only `page_token` and `page_size`.
3. **email-manager** (`v1EmailsGet`) ignores the request body and instead reads filters
   from **URL query params** via `utilHandler.URLParseFilters(u)`. Because the filters
   were never in the URL, the resulting filter map is empty.
4. **dbhandler** (`bin-email-manager/pkg/dbhandler/emails.go:154` `EmailList`) calls
   `ApplyFields` with the empty map → no `WHERE customer_id` clause → the query returns
   **every customer's emails**.

The single-item `GET /emails/{id}` is unaffected because api-manager performs an explicit
ownership check (`hasPermission`) on the returned record, which is the source of the 403.

### Secondary subtlety: binary `customer_id`

`customer_id` (and `id`) are stored as **binary** in the database (see
`emails.go` using `id.Bytes()` in WHERE clauses). A naive "parse the body as
`map[string]any`" fix would deliver `customer_id` as a plain string, and
`ApplyFields`' string branch would emit `WHERE customer_id = '<uuid-string>'`,
which matches **no** binary rows — turning a data leak into a silently-empty list.

The established repo pattern (ai-manager, call-manager, etc.) solves both parsing
and typing at once with a typed `FieldStruct` whose fields carry `filter:` tags,
fed through `utilhandler.ConvertFilters`. `ConvertFilters` re-types the raw JSON
value (string → `uuid.UUID`), so `ApplyFields` emits the correct **binary** WHERE.
The `email` model never adopted this pattern because it has only
`type Field string` and **no `FieldStruct`**.

## Convention check

Inspected the api-manager list handlers for `call`, `message`, `conference`, and
`agent`. **None** of them re-filter the RPC result by `customer_id`; they all rely
solely on the manager-level `customer_id` filter as the single enforcement boundary.
`email.go`'s `EmailList` already follows this exact pattern correctly — the only
break is the email-manager side discarding the filter. Therefore the fix stays
entirely within email-manager, and **no api-manager defense-in-depth re-filter** is
added (it would be inconsistent with every other resource).

## Design

All changes are in `bin-email-manager`.

### 1. New file: `models/email/filters.go`

Add a typed `FieldStruct` covering the sensibly-filterable columns, mirroring
`bin-ai-manager/models/ai/filters.go`. JSON columns (`source`, `destinations`,
`attachments`) and large text columns (`subject`, `content`) are intentionally
excluded — they are not equality-filter targets. The model's named types
(`ProviderType`, `Status`) are used directly, exactly as `ai/filters.go` uses its
own named enum types.

```go
package email

import "github.com/gofrs/uuid"

// FieldStruct defines the filterable fields for Email list queries.
// The `filter:` tag is the wire key sent by callers (api-manager) and the
// DB column name applied by ApplyFields.
type FieldStruct struct {
	ID                  uuid.UUID    `filter:"id"`
	CustomerID          uuid.UUID    `filter:"customer_id"`
	ActiveflowID        uuid.UUID    `filter:"activeflow_id"`
	ProviderType        ProviderType `filter:"provider_type"`
	ProviderReferenceID string       `filter:"provider_reference_id"`
	Status              Status       `filter:"status"`
	Deleted             bool         `filter:"deleted"`
}
```

### 2. `pkg/listenhandler/v1_emails.go` → `v1EmailsGet`

Replace the URL-filter block with the body-parse + type-convert pattern used by
call-manager (`bin-call-manager/pkg/listenhandler/v1_calls.go:44`):

```go
// parse the filters from the request body
tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
if err != nil {
	log.Errorf("Could not parse filters. err: %v", err)
	return simpleResponse(400), nil
}

filters, err := utilhandler.ConvertFilters[email.FieldStruct, email.Field](email.FieldStruct{}, tmpFilters)
if err != nil {
	log.Errorf("Could not convert filters. err: %v", err)
	return simpleResponse(400), nil
}
```

Remove the now-dead `URLParseFilters(u)` block and the manual
`map[email.Field]any` construction loop.

**Implementation notes (verified against current code):**

- **Logger must be declared — it is NOT currently in scope.** Unlike call-manager's
  `processV1CallsGet`, the current `v1EmailsGet` has no `log` variable. The snippet
  above assumes one. Add a logger at the top of the handler, matching call-manager:
  ```go
  log := logrus.WithFields(logrus.Fields{"func": "v1EmailsGet", "request": m})
  ```
  (Add the `github.com/sirupsen/logrus` import.) If a different error/logging
  convention is already established elsewhere in email-manager's listenhandler, match
  that instead — but the handler currently logs nothing, so introducing the
  call-manager-style logger is the expected outcome.
- **`ParseFiltersFromRequestBody` and `ConvertFilters` are package-level functions**
  on `bin-common-handler/pkg/utilhandler`, not methods on the `h.utilHandler` interface
  field. Add `import "monorepo/bin-common-handler/pkg/utilhandler"` to `v1_emails.go`
  (it is currently imported only in the package's `main.go`). Do not look for an
  interface method.
- **`simpleResponse` already exists** in the package (`listenhandler/main.go:77`) — no
  new helper needed.
- **`url` import stays** (pagination still parses the URI); drop `strings` only if it
  becomes unreferenced after removing the URL-filter loop.
- **Named-type precision:** `ProviderType` and `Status` are declared in `FieldStruct`
  for intent/documentation, but `ConvertFilters` down-converts named string types to a
  plain Go `string` at runtime. That is harmless — `ApplyFields` handles the string via
  its `case string:` branch and `status`/`provider_type` are plain (non-binary) columns.
  Do not expect a typed `email.Status` value to reach `ApplyFields`.

### Data flow after the fix

```
api-manager sets {customer_id, deleted}
  → marshalled into RPC request body
  → email-manager ParseFiltersFromRequestBody  (map[string]any)
  → ConvertFilters[FieldStruct, Field]          (customer_id string → uuid.UUID, deleted → bool)
  → emailHandler.List(filters)
  → dbhandler.EmailList → ApplyFields
  → WHERE customer_id = <binary> AND tm_delete IS NULL
```

## Error handling

- Malformed JSON body → `400` (matches call-manager).
- Empty body → `ParseFiltersFromRequestBody` returns an empty map → no filters
  applied. In practice api-manager always sends `customer_id`, so an unfiltered list
  is only reachable by direct/internal RPC callers — identical to the behavior of
  every other manager and therefore acceptable.
- Unknown filter keys in the body are ignored by `ConvertFilters` (it iterates the
  `FieldStruct` fields, not the incoming map), so unexpected keys cannot inject
  arbitrary WHERE clauses.

## Testing

### Unit — `pkg/listenhandler/v1_emails_test.go`
- Remove the `mockUtil.EXPECT().URLParseFilters(...)` expectation (no longer called).
- **Keep the `?`-query URI form on each test request.** Routing matches
  `regV1EmailsGet = /v1/emails\?...` (`main.go:46`), so the request URI must still
  carry `?page_token=...&page_size=...` for `processRequest` to reach `v1EmailsGet`.
  Only the *filters* move to `m.Data`; pagination stays in the URI.
- Drive filters through `m.Data` (request body) instead. Add/adjust cases:
  - body `{"customer_id":"<uuid>"}` → `emailHandler.List` receives a map whose
    `customer_id` value is a `uuid.UUID` (not a string).
  - body `{"customer_id":"<uuid>","deleted":false}` → both filters present and typed.
  - malformed body (e.g. `[`) → handler returns `400`, `List` not called.
  - empty body → `List` called with an empty filter map.

### Unit — `pkg/dbhandler/emails_test.go`
- Confirm (or add) a case asserting `EmailList` with a `customer_id` filter emits a
  binary-bound `WHERE customer_id = ?` (the `.Bytes()` form). Likely already covered
  by `ApplyFields`; verify rather than duplicate.

### Regression — api-validator
- `test_get_email_if_exists` (the failing test in the issue) goes green: every listed
  ID is now owned by the caller, so the follow-up single GET returns `200`.

### Verification workflow (before commit, in `bin-email-manager`)
```
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Out of scope

- No api-manager changes (its filter logic is already correct).
- No DB migration (all referenced columns already exist).
- No other endpoints — only `v1EmailsGet` exhibited the bug (`URLParseFilters` has a
  single production call site in email-manager).

## Acceptance criteria

1. `GET /v1.0/emails` returns only records whose `customer_id` matches the
   authenticated customer.
2. Every ID returned by the list endpoint returns `200` (not `403`) on
   `GET /v1.0/emails/{id}` for the same credential.
3. `test_get_email_if_exists` passes in api-validator.
4. Full `bin-email-manager` verification workflow passes (tidy, vendor, generate,
   test, lint).
