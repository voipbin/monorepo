# Fix cross-customer email leak in `GET /emails` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `GET /v1.0/emails` (list) return only the authenticated customer's emails by having `bin-email-manager` parse the `customer_id` filter from the RPC request body instead of (non-existent) URL query params.

**Architecture:** Adopt the established repo filter pattern (ai-manager / call-manager). Add a typed `email.FieldStruct` whose `filter:` tags drive `utilhandler.ConvertFilters`, then rewrite `v1EmailsGet` to read filters from `m.Data` (request body) via `ParseFiltersFromRequestBody` + `ConvertFilters`. `ConvertFilters` re-types the JSON `customer_id` string into `uuid.UUID`, so `ApplyFields` emits the correct **binary** `WHERE customer_id = ?` clause. No api-manager or DB-schema changes.

**Tech Stack:** Go, `go.uber.org/mock/gomock`, `github.com/gofrs/uuid`, `github.com/sirupsen/logrus`, squirrel (via `bin-common-handler/pkg/databasehandler.ApplyFields`), RabbitMQ RPC (`sock.Request`).

**Spec:** `docs/superpowers/specs/2026-06-02-email-list-customer-filter-design.md`
**Branch:** `NOJIRA-Fix-email-list-customer-filter` (worktree already created)
**Working dir for all commands:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-email-list-customer-filter/bin-email-manager`

---

## File Structure

- **Create:** `bin-email-manager/models/email/filters.go` — typed `FieldStruct` declaring the filterable email columns with `filter:` tags. Single responsibility: define the allowed list-filter surface. Mirrors `bin-ai-manager/models/ai/filters.go`.
- **Modify:** `bin-email-manager/pkg/listenhandler/v1_emails.go` (`v1EmailsGet`, lines ~17-52) — read filters from the request body instead of URL query params; add a logrus logger and the `utilhandler` package import.
- **Modify:** `bin-email-manager/pkg/listenhandler/v1_emails_test.go` (`Test_v1EmailsGet`, lines ~19-160) — drive filters through `m.Data`, drop the `URLParseFilters` mock expectation, add a malformed-body 400 case.

No other endpoints are affected: `URLParseFilters` has exactly one production call site in email-manager (`v1_emails.go:30`).

---

## Task 1: Add `email.FieldStruct`

**Files:**
- Create: `bin-email-manager/models/email/filters.go`

This is a declarative struct with no behavior of its own (exactly like `ai/filters.go`, which has no dedicated test). It is exercised end-to-end by the handler test in Task 2. We still verify it compiles and that the package builds.

- [ ] **Step 1: Create the FieldStruct file**

Create `bin-email-manager/models/email/filters.go` with this exact content:

```go
package email

import "github.com/gofrs/uuid"

// FieldStruct defines the allowed filters for Email list queries.
// Each field corresponds to a filterable database column. The `filter:` tag is
// both the wire key sent by callers (api-manager) and the DB column name applied
// by databasehandler.ApplyFields.
//
// JSON columns (source, destinations, attachments) and large free-text columns
// (subject, content) are intentionally excluded — they are not equality-filter
// targets.
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

Notes for the implementer:
- `ProviderType` and `Status` are named string types already declared in `models/email/main.go`. They are used here for intent/documentation; `ConvertFilters` down-converts them to a plain `string` at runtime, which is correct because `provider_type`/`status` are plain (non-binary) columns.
- `uuid` is referenced by three fields, so the import is used — no unused-import lint error.

- [ ] **Step 2: Verify the package compiles**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-email-list-customer-filter/bin-email-manager && go build ./models/email/`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-email-list-customer-filter
git add bin-email-manager/models/email/filters.go
git commit -m "NOJIRA-Fix-email-list-customer-filter

- bin-email-manager: Add email.FieldStruct for typed list filters"
```

---

## Task 2: Rewrite `v1EmailsGet` to read filters from the request body (TDD)

**Files:**
- Modify: `bin-email-manager/pkg/listenhandler/v1_emails.go`
- Test: `bin-email-manager/pkg/listenhandler/v1_emails_test.go`

The current handler calls `h.utilHandler.URLParseFilters(u)` and builds filters from URL `filter_*` query params. We change it to parse the request body. We update the test FIRST (red), then change the handler (green).

- [ ] **Step 1: Rewrite the test to drive filters from the request body**

Replace the entire `Test_v1EmailsGet` function in `bin-email-manager/pkg/listenhandler/v1_emails_test.go` with the following. Key changes vs. the original: filters now live in `request.Data` (JSON body); the URI keeps its `?...page_token=...&page_size=...` query string (required for routing to match `regV1EmailsGet = /v1/emails\?`); the `responseFilters`/`expectFilters`/`URLParseFilters` machinery is removed; a malformed-body case asserts a `400`.

```go
func Test_v1EmailsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken string
		pageSize  uint64

		responseEmails []*email.Email

		expectRes *sock.Response
	}{
		{
			name: "1 item",
			request: &sock.Request{
				URI:      "/v1/emails?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"16d3fcf0-7f4c-11ec-a4c3-7bf43125108d","deleted":false}`),
			},

			pageToken: "2020-10-10T03:30:17.000000Z",
			pageSize:  10,

			responseEmails: []*email.Email{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("2a046c74-00c7-11f0-b07e-a385bcd60724"),
						CustomerID: uuid.FromStringOrNil("16d3fcf0-7f4c-11ec-a4c3-7bf43125108d"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2a046c74-00c7-11f0-b07e-a385bcd60724","customer_id":"16d3fcf0-7f4c-11ec-a4c3-7bf43125108d","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			name: "2 items",
			request: &sock.Request{
				URI:      "/v1/emails?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"2457d824-7f4c-11ec-9489-b3552a7c9d63","deleted":false}`),
			},

			pageToken: "2020-10-10T03:30:17.000000Z",
			pageSize:  10,

			responseEmails: []*email.Email{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("2a242316-00c7-11f0-96db-93d500b33431"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("2c2a494c-00c7-11f0-be49-8f6777e928d8"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2a242316-00c7-11f0-96db-93d500b33431","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null},{"id":"2c2a494c-00c7-11f0-be49-8f6777e928d8","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			name: "empty",
			request: &sock.Request{
				URI:      "/v1/emails?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"3ee14bee-7f4c-11ec-a1d8-a3a488ed5885","deleted":false}`),
			},

			pageToken: "2020-10-10T03:30:17.000000Z",
			pageSize:  10,

			responseEmails: []*email.Email{},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			h := &listenHandler{
				utilHandler: mockUtil,
				sockHandler: mockSock,

				emailHandler: mockEmail,
			}

			mockEmail.EXPECT().List(gomock.Any(), tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.responseEmails, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1EmailsGet_malformedBody(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockEmail := emailhandler.NewMockEmailHandler(mc)

	h := &listenHandler{
		utilHandler:  mockUtil,
		sockHandler:  mockSock,
		emailHandler: mockEmail,
	}

	request := &sock.Request{
		URI:      "/v1/emails?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
		Data:     []byte(`{not-json`),
	}

	// emailHandler.List must NOT be called when the body cannot be parsed.
	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
	}
}
```

Note: `commonaddress` may become an unused import after this rewrite. If `goimports`/the compiler flags it, remove the `commonaddress "monorepo/bin-common-handler/models/address"` line from the test's import block. (Leave all other imports — `identity`, `sock`, `sockhandler`, `utilhandler`, `email`, `emailhandler`, `reflect`, `testing`, `uuid`, `gomock` — in place; they are all still used.)

- [ ] **Step 2: Run the test to verify it fails (RED)**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-email-list-customer-filter/bin-email-manager && go test ./pkg/listenhandler/ -run Test_v1EmailsGet -v`
Expected: FAIL. The current handler still calls `h.utilHandler.URLParseFilters(...)`, which is no longer mocked → gomock reports an unexpected call to `URLParseFilters` (and/or the malformed-body test fails because the current handler returns 200, not 400).

- [ ] **Step 3: Rewrite the handler to parse filters from the body (GREEN)**

In `bin-email-manager/pkg/listenhandler/v1_emails.go`, replace the import block and the `v1EmailsGet` function body.

First, update the import block (top of file) to this:

```go
import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)
```

(Removed `strings` — only used by the deleted URL-filter loop. Added `utilhandler` and `logrus`. Keep `uuid`, `errors`, `json`, `request` — they are used by other handlers in this file such as `v1EmailsPost`. If after the full edit the compiler reports `strings` or any other import as unused, remove that line; if it reports `uuid`/`errors`/`request` as unused, that means another handler in the file was changed unexpectedly — do not remove them.)

Then replace the `v1EmailsGet` function (currently lines ~16-52) with:

```go
// v1EmailsGet handles /v1/emails GET request
func (h *listenHandler) v1EmailsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1EmailsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

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

	tmp, err := h.emailHandler.List(ctx, pageToken, pageSize, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get emails")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the res")
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
```

- [ ] **Step 4: Run the test to verify it passes (GREEN)**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-email-list-customer-filter/bin-email-manager && go test ./pkg/listenhandler/ -run Test_v1EmailsGet -v`
Expected: PASS for `Test_v1EmailsGet` (all three subtests) and `Test_v1EmailsGet_malformedBody`.

- [ ] **Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-email-list-customer-filter
git add bin-email-manager/pkg/listenhandler/v1_emails.go bin-email-manager/pkg/listenhandler/v1_emails_test.go
git commit -m "NOJIRA-Fix-email-list-customer-filter

- bin-email-manager: Parse GET /emails list filters from request body so customer_id is enforced (fixes cross-customer leak, issue #956)"
```

---

## Task 3: Full verification workflow

**Files:** none (verification only)

- [ ] **Step 1: Run the mandatory verification workflow**

Run, from the service directory:

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-email-list-customer-filter/bin-email-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all five steps succeed; `go test ./...` passes; `golangci-lint` reports no issues. `go generate ./...` regenerates mocks — `URLParseFilters` stays on the `UtilHandler` interface (other handlers/services still use it), so no mock churn is expected for this change.

- [ ] **Step 2: Commit any go.mod/go.sum/generated changes (if produced)**

If `go mod tidy` or `go generate` changed tracked files (e.g. `go.mod`, `go.sum`, regenerated mocks — NOT `vendor/`, which is gitignored):

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-email-list-customer-filter
git add -A -- ':!bin-email-manager/vendor'
git status --short
git commit -m "NOJIRA-Fix-email-list-customer-filter

- bin-email-manager: Sync go.mod/go.sum and generated files after filter change"
```

If nothing changed, skip this step.

---

## Post-implementation (handled outside this plan)

- **api-validator regression:** `test_get_email_if_exists` should pass once deployed — every listed ID is now owned by the caller, so the follow-up single GET returns 200. Add/confirm api-validator coverage per repo workflow.
- **PR:** Per repo rules, fetch latest `main`, check for conflicts, then open a PR titled `NOJIRA-Fix-email-list-customer-filter`. Do NOT merge without explicit user authorization.

---

## Acceptance Criteria (from spec)

1. `GET /v1.0/emails` returns only records whose `customer_id` matches the authenticated customer.
2. Every ID returned by the list endpoint returns `200` (not `403`) on `GET /v1.0/emails/{id}` for the same credential.
3. `test_get_email_if_exists` passes in api-validator.
4. Full `bin-email-manager` verification workflow passes (tidy, vendor, generate, test, lint).
