# VoIPbin API Error Response Codes — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Each code task follows TDD per @superpowers:test-driven-development — write the failing test first, watch it fail for the right reason, then implement.

**Goal:** Ship the foundation for VoIPbin's structured API error responses (status + reason + domain + message + request_id) as a series of small, independently-verifiable PRs. This plan details PR 0a to execution-ready granularity; later PRs are scoped at a higher level.

**Architecture:** `VoipbinError` type in `bin-common-handler/models/errors` → consumed by `bin-api-manager` server layer and (eventually) internal managers via the existing `sock.Response` RPC envelope. See the companion design doc: `docs/plans/2026-04-24-api-error-response-codes-design.md`.

**Tech Stack:** Go, standard `testing` (table-driven, no testify), `github.com/gin-gonic/gin`, `github.com/oklog/ulid/v2`, `github.com/sirupsen/logrus`, `github.com/deepmap/oapi-codegen` (OpenAPI code generation).

---

## Preconditions

- Worktree exists at `~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-codes` on branch `NOJIRA-api-error-response-codes`.
- Branch is based on latest `origin/main`.
- Design doc committed at `docs/plans/2026-04-24-api-error-response-codes-design.md`.
- `ServiceNameAPIManager` constant already exists in `bin-common-handler/models/outline/servicename.go` — no need to add.

## Known constraint flag for review

**`bin-common-handler` admission rule challenge.** The monorepo rule says a package must be used by 3+ services to live in `bin-common-handler`. The new `models/errors` package will have only `bin-api-manager` as a day-1 consumer. Defensible exception: the type becomes part of the shared RPC contract (`sock.Response.Data`) and `requesthandler` RPC wrappers will use it for typed unmarshal. If reviewers push back, fallback is to place `VoipbinError` in `bin-api-manager/pkg/errors/` initially and promote to `bin-common-handler` when the first internal manager adopts it. Call this out in the PR description so reviewers can decide.

---

## PR 0a — `bin-common-handler` foundation (execution-ready)

**Scope:** introduce the `VoipbinError` type, canonical statuses, marshal/unmarshal helpers, and the `sock.Request.RequestID` field. Zero api-manager changes. Tests only — no production integration yet.

**Target files:**
- Create: `bin-common-handler/models/errors/status.go`
- Create: `bin-common-handler/models/errors/status_test.go`
- Create: `bin-common-handler/models/errors/voipbin_error.go`
- Create: `bin-common-handler/models/errors/voipbin_error_test.go`
- Create: `bin-common-handler/models/errors/constructors.go`
- Create: `bin-common-handler/models/errors/constructors_test.go`
- Create: `bin-common-handler/models/errors/datatype.go`
- Create: `bin-common-handler/models/errors/rpc.go`
- Create: `bin-common-handler/models/errors/rpc_test.go`
- Modify: `bin-common-handler/models/sock/message.go` (add `RequestID` field)
- Modify: `bin-common-handler/models/sock/message_test.go` (extend existing tests)

**Verification cadence:** run `go test ./...` after each task. Run the full 5-step verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`) from `bin-common-handler/` before the final PR push.

---

### Task 1: Define the `Status` type and the 10 canonical constants

**Files:**
- Create: `bin-common-handler/models/errors/status.go`
- Create: `bin-common-handler/models/errors/status_test.go`

**Step 1: Write the failing test**

```go
// bin-common-handler/models/errors/status_test.go
package errors

import "testing"

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"invalid_argument", StatusInvalidArgument, "INVALID_ARGUMENT"},
		{"unauthenticated", StatusUnauthenticated, "UNAUTHENTICATED"},
		{"payment_required", StatusPaymentRequired, "PAYMENT_REQUIRED"},
		{"permission_denied", StatusPermissionDenied, "PERMISSION_DENIED"},
		{"not_found", StatusNotFound, "NOT_FOUND"},
		{"already_exists", StatusAlreadyExists, "ALREADY_EXISTS"},
		{"failed_precondition", StatusFailedPrecondition, "FAILED_PRECONDITION"},
		{"resource_exhausted", StatusResourceExhausted, "RESOURCE_EXHAUSTED"},
		{"unavailable", StatusUnavailable, "UNAVAILABLE"},
		{"internal", StatusInternal, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("got %q want %q", string(tt.constant), tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestStatusConstants -v
```

Expected: compile error — `Status` and constants undefined.

**Step 3: Write minimal implementation**

```go
// bin-common-handler/models/errors/status.go
// Package errors defines the shared VoipbinError type used across the
// VoIPbin monorepo for external API error responses.
package errors

// Status is the canonical error status. It maps 1:1 to an HTTP status
// code (see bin-api-manager for the mapping). The set is intentionally
// closed — extending it requires a coordinated schema update.
type Status string

const (
	StatusInvalidArgument    Status = "INVALID_ARGUMENT"
	StatusUnauthenticated    Status = "UNAUTHENTICATED"
	StatusPaymentRequired    Status = "PAYMENT_REQUIRED"
	StatusPermissionDenied   Status = "PERMISSION_DENIED"
	StatusNotFound           Status = "NOT_FOUND"
	StatusAlreadyExists      Status = "ALREADY_EXISTS"
	StatusFailedPrecondition Status = "FAILED_PRECONDITION"
	StatusResourceExhausted  Status = "RESOURCE_EXHAUSTED"
	StatusUnavailable        Status = "UNAVAILABLE"
	StatusInternal           Status = "INTERNAL"
)
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestStatusConstants -v
```

Expected: PASS.

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-codes
git add bin-common-handler/models/errors/status.go bin-common-handler/models/errors/status_test.go
git commit -m "NOJIRA-api-error-response-codes

Add canonical Status type and constants for the shared VoipbinError vocabulary.

- bin-common-handler: Add models/errors package with Status type and 10 canonical status constants"
```

---

### Task 2: Define the `VoipbinError` struct with JSON tags

**Files:**
- Create: `bin-common-handler/models/errors/voipbin_error.go`
- Create: `bin-common-handler/models/errors/voipbin_error_test.go`

**Step 1: Write the failing test**

```go
// bin-common-handler/models/errors/voipbin_error_test.go
package errors

import (
	stderrors "errors"
	"encoding/json"
	"testing"
)

func TestVoipbinErrorFields(t *testing.T) {
	e := &VoipbinError{
		Status:  StatusNotFound,
		Reason:  "CALL_NOT_FOUND",
		Domain:  "call-manager",
		Message: "The call was not found.",
		Cause:   stderrors.New("underlying"),
	}
	if e.Status != StatusNotFound {
		t.Errorf("wrong Status: %v", e.Status)
	}
	if e.Reason != "CALL_NOT_FOUND" {
		t.Errorf("wrong Reason: %v", e.Reason)
	}
	if e.Domain != "call-manager" {
		t.Errorf("wrong Domain: %v", e.Domain)
	}
	if e.Message != "The call was not found." {
		t.Errorf("wrong Message: %v", e.Message)
	}
	if e.Cause == nil || e.Cause.Error() != "underlying" {
		t.Errorf("wrong Cause: %v", e.Cause)
	}
}

func TestVoipbinErrorJSONExcludesCause(t *testing.T) {
	e := &VoipbinError{
		Status:  StatusNotFound,
		Reason:  "CALL_NOT_FOUND",
		Domain:  "call-manager",
		Message: "The call was not found.",
		Cause:   stderrors.New("DB driver error: connection refused"),
	}

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	s := string(b)
	for _, want := range []string{
		`"status":"NOT_FOUND"`,
		`"reason":"CALL_NOT_FOUND"`,
		`"domain":"call-manager"`,
		`"message":"The call was not found."`,
	} {
		if !contains(s, want) {
			t.Errorf("expected %q in JSON %q", want, s)
		}
	}
	for _, forbidden := range []string{"cause", "connection refused", "DB driver"} {
		if contains(s, forbidden) {
			t.Errorf("forbidden token %q leaked into JSON %q", forbidden, s)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestVoipbinError -v
```

Expected: compile error — `VoipbinError` undefined.

**Step 3: Write minimal implementation**

```go
// bin-common-handler/models/errors/voipbin_error.go
package errors

// VoipbinError is the canonical error shape returned from the external
// VoIPbin API and (eventually) over RPC between internal managers.
// The Cause field is for server-side logging only and is never
// serialized to clients.
type VoipbinError struct {
	Status  Status `json:"status"`
	Reason  string `json:"reason"`
	Domain  string `json:"domain"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestVoipbinError -v
```

Expected: PASS (both tests).

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/voipbin_error.go bin-common-handler/models/errors/voipbin_error_test.go
git commit -m "NOJIRA-api-error-response-codes

Introduce VoipbinError struct with fields Status/Reason/Domain/Message/Cause.
Cause is excluded from JSON to avoid leaking server-side detail to clients.

- bin-common-handler: Add VoipbinError struct and JSON serialization contract"
```

---

### Task 3: Implement `Error()` method

**Files:**
- Modify: `bin-common-handler/models/errors/voipbin_error.go`
- Modify: `bin-common-handler/models/errors/voipbin_error_test.go`

**Step 1: Write the failing test**

```go
// Append to voipbin_error_test.go
func TestVoipbinErrorErrorMethod(t *testing.T) {
	e := &VoipbinError{
		Status:  StatusPermissionDenied,
		Reason:  "BILLING_ACCESS_DENIED",
		Domain:  "billing-manager",
		Message: "Not allowed.",
	}
	got := e.Error()
	want := "billing-manager: BILLING_ACCESS_DENIED: Not allowed."
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestVoipbinErrorErrorMethodWithCause(t *testing.T) {
	e := &VoipbinError{
		Status:  StatusInternal,
		Reason:  "INTERNAL",
		Domain:  "api-manager",
		Message: "Something went wrong.",
		Cause:   stderrors.New("pq: connection refused"),
	}
	got := e.Error()
	want := "api-manager: INTERNAL: Something went wrong.: pq: connection refused"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestVoipbinErrorError -v
```

Expected: compile error — `(*VoipbinError).Error` undefined.

**Step 3: Write minimal implementation**

```go
// Append to voipbin_error.go
import "fmt"

func (e *VoipbinError) Error() string {
	if e == nil {
		return "<nil VoipbinError>"
	}
	base := fmt.Sprintf("%s: %s: %s", e.Domain, e.Reason, e.Message)
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", base, e.Cause.Error())
	}
	return base
}
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestVoipbinErrorError -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/voipbin_error.go bin-common-handler/models/errors/voipbin_error_test.go
git commit -m "NOJIRA-api-error-response-codes

Implement the error interface for VoipbinError so it composes with fmt.Errorf / errors.Is / errors.As.

- bin-common-handler: Implement Error() on VoipbinError"
```

---

### Task 4: Implement `Unwrap()` method for `errors.Is` / `errors.As`

**Files:**
- Modify: `bin-common-handler/models/errors/voipbin_error.go`
- Modify: `bin-common-handler/models/errors/voipbin_error_test.go`

**Step 1: Write the failing test**

```go
// Append to voipbin_error_test.go
func TestVoipbinErrorUnwrap(t *testing.T) {
	inner := stderrors.New("inner failure")
	e := &VoipbinError{
		Status:  StatusInternal,
		Reason:  "INTERNAL",
		Domain:  "api-manager",
		Message: "wrap test",
		Cause:   inner,
	}

	// errors.Is walks the chain via Unwrap.
	if !stderrors.Is(e, inner) {
		t.Fatalf("errors.Is did not find wrapped cause")
	}

	// errors.As on a wrapped VoipbinError finds it.
	wrapped := fmt.Errorf("context: %w", e)
	var target *VoipbinError
	if !stderrors.As(wrapped, &target) {
		t.Fatalf("errors.As did not recover VoipbinError from wrapped chain")
	}
	if target.Reason != "INTERNAL" {
		t.Errorf("errors.As returned wrong VoipbinError: %v", target)
	}
}
```

(You will need to add `"fmt"` to the test file imports.)

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestVoipbinErrorUnwrap -v
```

Expected: FAIL — `errors.Is` returns false because `Unwrap` isn't defined.

**Step 3: Write minimal implementation**

```go
// Append to voipbin_error.go
func (e *VoipbinError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestVoipbinErrorUnwrap -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/voipbin_error.go bin-common-handler/models/errors/voipbin_error_test.go
git commit -m "NOJIRA-api-error-response-codes

Support errors.Is and errors.As traversal so callers can walk wrapped VoipbinError chains.

- bin-common-handler: Add Unwrap() to VoipbinError"
```

---

### Task 5: Implement `Wrap(cause)` fluent method

**Files:**
- Modify: `bin-common-handler/models/errors/voipbin_error.go`
- Modify: `bin-common-handler/models/errors/voipbin_error_test.go`

**Step 1: Write the failing test**

```go
// Append to voipbin_error_test.go
func TestVoipbinErrorWrap(t *testing.T) {
	e := &VoipbinError{Status: StatusInternal, Reason: "INTERNAL", Domain: "api-manager", Message: "x"}
	inner := stderrors.New("boom")
	out := e.Wrap(inner)

	if out != e {
		t.Errorf("Wrap should return the receiver for chaining")
	}
	if e.Cause != inner {
		t.Errorf("Wrap did not set Cause: %v", e.Cause)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestVoipbinErrorWrap -v
```

Expected: compile error — `Wrap` undefined.

**Step 3: Write minimal implementation**

```go
// Append to voipbin_error.go
func (e *VoipbinError) Wrap(cause error) *VoipbinError {
	if e == nil {
		return nil
	}
	e.Cause = cause
	return e
}
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestVoipbinErrorWrap -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/voipbin_error.go bin-common-handler/models/errors/voipbin_error_test.go
git commit -m "NOJIRA-api-error-response-codes

Add a fluent Wrap(cause) method so constructors can chain underlying errors without an extra statement.

- bin-common-handler: Add Wrap(cause) to VoipbinError"
```

---

### Task 6: Add the 10 constructor functions

**Files:**
- Create: `bin-common-handler/models/errors/constructors.go`
- Create: `bin-common-handler/models/errors/constructors_test.go`

**Step 1: Write the failing test**

```go
// bin-common-handler/models/errors/constructors_test.go
package errors

import "testing"

func TestConstructors(t *testing.T) {
	tests := []struct {
		name       string
		build      func(string, string, string) *VoipbinError
		wantStatus Status
	}{
		{"invalid_argument", InvalidArgument, StatusInvalidArgument},
		{"unauthenticated", Unauthenticated, StatusUnauthenticated},
		{"payment_required", PaymentRequired, StatusPaymentRequired},
		{"permission_denied", PermissionDenied, StatusPermissionDenied},
		{"not_found", NotFound, StatusNotFound},
		{"already_exists", AlreadyExists, StatusAlreadyExists},
		{"failed_precondition", FailedPrecondition, StatusFailedPrecondition},
		{"resource_exhausted", ResourceExhausted, StatusResourceExhausted},
		{"unavailable", Unavailable, StatusUnavailable},
		{"internal", Internal, StatusInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.build("d", "R", "m")
			if e == nil {
				t.Fatal("nil VoipbinError")
			}
			if e.Status != tt.wantStatus {
				t.Errorf("wrong Status: got %q want %q", e.Status, tt.wantStatus)
			}
			if e.Domain != "d" || e.Reason != "R" || e.Message != "m" {
				t.Errorf("wrong fields: %+v", e)
			}
			if e.Cause != nil {
				t.Errorf("Cause should be nil by default")
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestConstructors -v
```

Expected: compile error — constructors undefined.

**Step 3: Write minimal implementation**

```go
// bin-common-handler/models/errors/constructors.go
package errors

func newVoipbinError(s Status, domain, reason, message string) *VoipbinError {
	return &VoipbinError{Status: s, Domain: domain, Reason: reason, Message: message}
}

func InvalidArgument(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusInvalidArgument, domain, reason, message)
}
func Unauthenticated(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusUnauthenticated, domain, reason, message)
}
func PaymentRequired(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusPaymentRequired, domain, reason, message)
}
func PermissionDenied(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusPermissionDenied, domain, reason, message)
}
func NotFound(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusNotFound, domain, reason, message)
}
func AlreadyExists(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusAlreadyExists, domain, reason, message)
}
func FailedPrecondition(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusFailedPrecondition, domain, reason, message)
}
func ResourceExhausted(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusResourceExhausted, domain, reason, message)
}
func Unavailable(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusUnavailable, domain, reason, message)
}
func Internal(domain, reason, message string) *VoipbinError {
	return newVoipbinError(StatusInternal, domain, reason, message)
}
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestConstructors -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/constructors.go bin-common-handler/models/errors/constructors_test.go
git commit -m "NOJIRA-api-error-response-codes

Add one typed constructor per canonical Status so callers construct VoipbinError without exposing the raw struct literal.

- bin-common-handler: Add 10 VoipbinError constructors"
```

---

### Task 7: Add `DataTypeVoipbinError` constant

**Files:**
- Create: `bin-common-handler/models/errors/datatype.go`
- Modify: `bin-common-handler/models/errors/constructors_test.go` (append to existing file)

**Step 1: Write the failing test**

```go
// Append to constructors_test.go
func TestDataTypeVoipbinError(t *testing.T) {
	if DataTypeVoipbinError != "voipbin_error" {
		t.Errorf("DataTypeVoipbinError = %q, want %q", DataTypeVoipbinError, "voipbin_error")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestDataTypeVoipbinError -v
```

Expected: compile error — `DataTypeVoipbinError` undefined.

**Step 3: Write minimal implementation**

```go
// bin-common-handler/models/errors/datatype.go
package errors

// DataTypeVoipbinError is the sock.Response.DataType value used by
// internal managers to signal that Data contains a JSON-serialized
// VoipbinError. bin-api-manager's requesthandler inspects this to
// decide whether to unmarshal the payload as a typed error.
const DataTypeVoipbinError = "voipbin_error"
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestDataTypeVoipbinError -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/datatype.go bin-common-handler/models/errors/constructors_test.go
git commit -m "NOJIRA-api-error-response-codes

Reserve the sock.Response.DataType value used to carry typed errors across the RPC boundary.

- bin-common-handler: Add DataTypeVoipbinError constant"
```

---

### Task 8: Add `FromResponse` helper (unmarshal `VoipbinError` from `sock.Response`)

**Files:**
- Create: `bin-common-handler/models/errors/rpc.go`
- Create: `bin-common-handler/models/errors/rpc_test.go`

**Step 1: Write the failing test**

```go
// bin-common-handler/models/errors/rpc_test.go
package errors

import (
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

func TestFromResponseTypedError(t *testing.T) {
	payload := &VoipbinError{
		Status:  StatusNotFound,
		Reason:  "CALL_NOT_FOUND",
		Domain:  "call-manager",
		Message: "The call was not found.",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal setup: %v", err)
	}
	resp := &sock.Response{
		StatusCode: 404,
		DataType:   DataTypeVoipbinError,
		Data:       data,
	}

	got := FromResponse(resp)
	if got == nil {
		t.Fatal("FromResponse returned nil for a typed error response")
	}
	if got.Status != StatusNotFound || got.Reason != "CALL_NOT_FOUND" {
		t.Errorf("wrong VoipbinError: %+v", got)
	}
}

func TestFromResponseSuccess(t *testing.T) {
	resp := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       json.RawMessage(`{"ok":true}`),
	}
	if got := FromResponse(resp); got != nil {
		t.Errorf("FromResponse on success should return nil, got %+v", got)
	}
}

func TestFromResponseErrorWithoutTypedDataType(t *testing.T) {
	// Legacy manager — error code but no typed body.
	resp := &sock.Response{
		StatusCode: 500,
		DataType:   "application/json",
		Data:       json.RawMessage(`{"message":"legacy"}`),
	}
	if got := FromResponse(resp); got != nil {
		t.Errorf("FromResponse without DataTypeVoipbinError must return nil, got %+v", got)
	}
}

func TestFromResponseMalformedData(t *testing.T) {
	resp := &sock.Response{
		StatusCode: 500,
		DataType:   DataTypeVoipbinError,
		Data:       json.RawMessage(`{not json`),
	}
	// Must not panic and must fall back to nil so the caller uses its own fallback path.
	if got := FromResponse(resp); got != nil {
		t.Errorf("malformed Data must return nil, got %+v", got)
	}
}

func TestFromResponseNil(t *testing.T) {
	if got := FromResponse(nil); got != nil {
		t.Errorf("nil response must return nil, got %+v", got)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestFromResponse -v
```

Expected: compile error — `FromResponse` undefined.

**Step 3: Write minimal implementation**

```go
// bin-common-handler/models/errors/rpc.go
package errors

import (
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
)

// FromResponse extracts a typed *VoipbinError from a sock.Response if
// the response signals one (StatusCode >= 400 AND DataType ==
// DataTypeVoipbinError AND Data unmarshals cleanly). Returns nil
// otherwise — callers should apply their own fallback.
func FromResponse(resp *sock.Response) *VoipbinError {
	if resp == nil || resp.StatusCode < 400 || resp.DataType != DataTypeVoipbinError || len(resp.Data) == 0 {
		return nil
	}
	out := &VoipbinError{}
	if err := json.Unmarshal(resp.Data, out); err != nil {
		return nil
	}
	return out
}
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestFromResponse -v
```

Expected: PASS (all five tests).

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/rpc.go bin-common-handler/models/errors/rpc_test.go
git commit -m "NOJIRA-api-error-response-codes

Add FromResponse helper. api-manager and internal managers use it to detect typed errors without growing a new RPC field.

- bin-common-handler: Add FromResponse helper to extract VoipbinError from sock.Response"
```

---

### Task 9: Add `ToResponse` helper (encode `VoipbinError` into `sock.Response`)

**Files:**
- Modify: `bin-common-handler/models/errors/rpc.go`
- Modify: `bin-common-handler/models/errors/rpc_test.go`

**Step 1: Write the failing test**

```go
// Append to rpc_test.go
func TestToResponse(t *testing.T) {
	e := NotFound("call-manager", "CALL_NOT_FOUND", "The call was not found.")
	resp, err := ToResponse(e)
	if err != nil {
		t.Fatalf("ToResponse returned error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("wrong StatusCode: %d", resp.StatusCode)
	}
	if resp.DataType != DataTypeVoipbinError {
		t.Errorf("wrong DataType: %s", resp.DataType)
	}

	// Round-trip: FromResponse must recover the original.
	got := FromResponse(resp)
	if got == nil || got.Status != StatusNotFound || got.Reason != "CALL_NOT_FOUND" {
		t.Errorf("round-trip failed: %+v", got)
	}
}

func TestToResponseAllStatuses(t *testing.T) {
	tests := []struct {
		status Status
		http   int
	}{
		{StatusInvalidArgument, 400},
		{StatusUnauthenticated, 401},
		{StatusPaymentRequired, 402},
		{StatusPermissionDenied, 403},
		{StatusNotFound, 404},
		{StatusAlreadyExists, 409},
		{StatusFailedPrecondition, 409},
		{StatusResourceExhausted, 429},
		{StatusUnavailable, 503},
		{StatusInternal, 500},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			e := &VoipbinError{Status: tt.status, Reason: "X", Domain: "d", Message: "m"}
			resp, err := ToResponse(e)
			if err != nil {
				t.Fatalf("ToResponse failed: %v", err)
			}
			if resp.StatusCode != tt.http {
				t.Errorf("wrong StatusCode for %s: got %d want %d", tt.status, resp.StatusCode, tt.http)
			}
		})
	}
}

func TestToResponseNil(t *testing.T) {
	if _, err := ToResponse(nil); err == nil {
		t.Errorf("ToResponse(nil) must return an error")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestToResponse -v
```

Expected: compile error — `ToResponse` undefined.

**Step 3: Write minimal implementation**

```go
// Append to rpc.go
import "errors" as stderrors  // NOTE: adjust imports; the file already
// imports encoding/json and monorepo/bin-common-handler/models/sock.
// Actually we'll use the "fmt" package for the error below, so import fmt.

func ToResponse(e *VoipbinError) (*sock.Response, error) {
	if e == nil {
		return nil, fmt.Errorf("cannot marshal nil VoipbinError")
	}
	body, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("marshal VoipbinError: %w", err)
	}
	return &sock.Response{
		StatusCode: httpStatusFor(e.Status),
		DataType:   DataTypeVoipbinError,
		Data:       body,
	}, nil
}

// httpStatusFor maps a canonical Status to an HTTP status code.
// This mapping is the single source of truth for the RPC→HTTP layer.
func httpStatusFor(s Status) int {
	switch s {
	case StatusInvalidArgument:
		return 400
	case StatusUnauthenticated:
		return 401
	case StatusPaymentRequired:
		return 402
	case StatusPermissionDenied:
		return 403
	case StatusNotFound:
		return 404
	case StatusAlreadyExists, StatusFailedPrecondition:
		return 409
	case StatusResourceExhausted:
		return 429
	case StatusUnavailable:
		return 503
	case StatusInternal:
		return 500
	default:
		return 500
	}
}
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/errors/... -run TestToResponse -v
```

Expected: PASS (all three tests).

**Step 5: Commit**

```bash
git add bin-common-handler/models/errors/rpc.go bin-common-handler/models/errors/rpc_test.go
git commit -m "NOJIRA-api-error-response-codes

Add ToResponse helper that encodes VoipbinError into a sock.Response, plus the canonical Status-to-HTTP-status mapping that both sides of the RPC share.

- bin-common-handler: Add ToResponse helper and canonical Status-to-HTTP mapping"
```

---

### Task 10: Add `RequestID` field to `sock.Request`

**Files:**
- Modify: `bin-common-handler/models/sock/message.go`
- Modify: `bin-common-handler/models/sock/message_test.go`

**Step 1: Write the failing test**

```go
// Append to bin-common-handler/models/sock/message_test.go
func TestRequestRequestIDField(t *testing.T) {
	r := Request{
		URI:       "/v1/calls/1",
		Method:    RequestMethodGet,
		RequestID: "req_01hxyz12345",
	}
	if r.RequestID != "req_01hxyz12345" {
		t.Errorf("RequestID not set: %q", r.RequestID)
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"request_id":"req_01hxyz12345"`) {
		t.Errorf("request_id missing from JSON: %s", string(b))
	}
}

func TestRequestRequestIDOmitempty(t *testing.T) {
	r := Request{
		URI:    "/v1/calls",
		Method: RequestMethodPost,
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "request_id") {
		t.Errorf("request_id should be omitted when empty, got: %s", string(b))
	}
}
```

Add `"strings"` to the test file's imports.

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler && go test ./models/sock/... -run TestRequestRequestID -v
```

Expected: compile error — `Request` has no `RequestID` field.

**Step 3: Write minimal implementation**

```go
// Modify bin-common-handler/models/sock/message.go
type Request struct {
	URI       string          `json:"uri"`
	Method    RequestMethod   `json:"method"`
	Publisher string          `json:"publisher"`
	DataType  string          `json:"data_type"`
	Data      json.RawMessage `json:"data,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}
```

**Step 4: Run test to verify it passes**

```bash
cd bin-common-handler && go test ./models/sock/... -v
```

Expected: PASS — including all pre-existing tests (confirms the addition is backward-compatible).

**Step 5: Commit**

```bash
git add bin-common-handler/models/sock/message.go bin-common-handler/models/sock/message_test.go
git commit -m "NOJIRA-api-error-response-codes

Add optional RequestID so request correlation IDs propagate from api-manager through RPC to internal managers for log correlation.

- bin-common-handler: Add RequestID field to sock.Request"
```

---

### Task 11: Run the full verification workflow

**No new files. Confirms the package compiles, tests pass, lint is clean, and go.sum is in sync.**

**Step 1: Run the workflow**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-codes/bin-common-handler
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all five steps pass. If `go mod tidy` or `go generate` mutated tracked files, stage and commit them.

**Step 2: Commit any tidy/generate changes (if any)**

```bash
git status
# If go.mod / go.sum / generated mocks changed:
git add bin-common-handler/go.mod bin-common-handler/go.sum bin-common-handler/pkg/*/mock_*.go 2>/dev/null
git diff --cached --quiet || git commit -m "NOJIRA-api-error-response-codes

Bring go.mod/go.sum and generated mocks in sync after introducing the new errors package.

- bin-common-handler: go mod tidy and mock regeneration sync"
```

(If nothing changed, skip the commit.)

---

### Task 12: Push the branch and open PR 0a

**Step 1: Verify remote tracking and pull main for conflict check**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-api-error-response-codes
git fetch origin main
git log --oneline HEAD..origin/main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
```

Expected: "no conflicts". If conflicts exist, rebase onto latest main, resolve, and re-run Task 11's verification workflow before pushing.

**Step 2: Push**

```bash
git push -u origin NOJIRA-api-error-response-codes
```

**Step 3: Open PR**

```bash
gh pr create --title "NOJIRA-api-error-response-codes" --body "$(cat <<'EOF'
Foundation for VoIPbin API structured error responses (PR 0a of N). Introduces the shared VoipbinError vocabulary and RPC request correlation, without wiring any callers yet. This PR changes only bin-common-handler so it can land independently before api-manager work begins.

See design doc: docs/plans/2026-04-24-api-error-response-codes-design.md
See implementation plan: docs/plans/2026-04-24-api-error-response-codes-plan.md

- bin-common-handler: Add models/errors package with Status type, VoipbinError struct (Error/Unwrap/Wrap), 10 constructors, DataTypeVoipbinError, FromResponse/ToResponse helpers
- bin-common-handler: Add optional RequestID field to sock.Request for cross-service log correlation

Reviewer note: this introduces a new models/errors package with only bin-api-manager as the day-1 consumer. The admission-rule threshold is 3 services. Defensible exception: VoipbinError is part of the shared RPC contract (sock.Response.Data) and will be consumed by requesthandler typed-unmarshal in PR 0b and by each internal manager that later migrates. If the committee prefers stricter adherence, the fallback is to put VoipbinError in bin-api-manager/pkg/errors/ and promote when a second consumer appears.
EOF
)"
```

---

### PR 0a success criteria

- [ ] All 12 tasks committed.
- [ ] `go test ./...` passes inside `bin-common-handler`.
- [ ] `golangci-lint run -v --timeout 5m` is clean.
- [ ] `go.mod` / `go.sum` / vendor are consistent; no uncommitted changes after verification.
- [ ] No changes outside `bin-common-handler` (zero api-manager or other-manager modifications).
- [ ] PR description explicitly flags the admission-rule exception for reviewers.

---

## Subsequent PRs (scoped at high level)

These need their own execution-ready plans before implementation. Below are the rough shape and per-PR scope so reviewers and future authors share the same mental model.

### PR 0b — bin-api-manager infrastructure

**Scope (from design §10.2):**
- `server/error.go`: `abortWithError`, `abortWithServiceError`, `httpStatusFor`, `assertErrorResponse` test helper.
- `server/error_translate.go`: translator (typed-passthrough → sentinel → transport → substring → INTERNAL) with `defer recover()` panic safety.
- `lib/middleware/request_id.go`: 30-char ULID middleware, echoes inbound `X-Request-Id`, stores in `gin.Context`, `c.Request.Context()`, logrus fields. Registered in `cmd/api-manager/main.go`.
- `pkg/serviceerrors/sentinels.go`: initial sentinel set (`ErrPermissionDenied`, `ErrNotFound`, `ErrAuthenticationRequired`, `ErrDirectAccessNotSupported`).
- Migrate `lib/middleware/ratelimit.go` (swap flat error shape for `abortWithError`).
- Migrate `lib/middleware/authenticate.go` (bare 401 + non-envelope 403 → typed errors).
- **Spike:** wire error responses into `GET /v1/ping` only, regenerate `gens/openapi_server/gen.go`, confirm `oapi-codegen` output is compatible. If not, pivot (document in PR).
- **Audit:** survey `Response.StatusCode` usage across manager callers.
- OpenAPI components in `bin-openapi-manager/openapi/openapi.yaml`: `ErrorBody` (with reserved `details` array), `ErrorResponse`, named responses. No per-endpoint wiring yet (except the spike endpoint).
- RST: update `bin-api-manager/docsdev/source/restful_api.rst` with envelope + status table; create empty `restful_api_errors.rst` catalog with table header; clean-rebuild and commit HTML.

**Verification:** full workflow for both `bin-openapi-manager` and `bin-api-manager`. RST HTML rebuild + `git add -f docsdev/build/`.

**Blocker:** depends on PR 0a being merged.

### PR 1 — Auth & identity migration

**Scope:** handler files `auth_*`, `me`, `customer`, plus `service_agents_*` auth if separate codepath. Convert `c.AbortWithStatus(…)` and legacy `c.AbortWithStatusJSON` to `abortWithServiceError`. Add sentinels or typed constructors in `servicehandler` counterparts. Wire the group's OpenAPI paths to reference named error responses. Append any new reasons to `restful_api_errors.rst`. Update handler tests via `assertErrorResponse`. Update `*_tutorial.rst` / `*_troubleshooting.rst` per AI-native docs Rule 5. Companion PR in `monorepo-monitoring/api-validator/`.

**Pre-flight blocker:** frontend client audit complete — `admin.voipbin.net` and `talk.voipbin.net` must tolerate 401/403/404/409/429/503 before this PR rolls out.

### PR 2 — Calls group

`calls`, `groupcalls`, `recordings`, `recordingfiles`, `transfers`. Same pattern as PR 1.

### PR 3 — Flows & activeflows

`flows`, `activeflows`.

### PR 4 — Numbers & providers

`numbers`, `available_numbers`, `providers`, `providercalls`, `trunks`, `routes`.

### PR 5 — Billing (exercises 402)

`billings`, `billing_account`, `billing_accounts`. First PR to emit `PAYMENT_REQUIRED` / 402. Verify the full 402 code path end-to-end.

### PR 6 — Messaging & conversations

`messages`, `emails`, `conversations`, `conversation_accounts`.

### PR 7 — AI, transcription, speaking

`ais`, `aicalls`, `aimessages`, `aisummaries`, `transcribes`, `transcripts`, `speakings`.

### PR 8 — Agents, queues, conferences, campaigns

`agents`, `queues`, `queuecalls`, `conferences`, `conferencecalls`, `campaigns`, `campaigncalls`, `outplans`, `outdials`.

### PR 9 — Storage, extensions, remainder

`storage_*`, `extensions`, `tags`, `teams`, `timelines*`, `aggregated_events`, `contacts`, remaining `service_agents_*`, `ws`, `accesskeys`, `rags`. Final sweep.

### PR N+1 — Shrink the translator fallback

Remove the substring-fallback cases from `server/error_translate.go`. Any error not caught by typed passthrough, sentinel, or transport detection returns clean `INTERNAL`. Add a test that asserts an unknown error hits the INTERNAL default with the original error in `Cause` (server-side only) and a generic client-visible message.

---

## Per-PR checklist (applies to every PR in this plan)

For each PR from 0a onward:

- [ ] Branch name matches `NOJIRA-api-error-response-codes` (single long-lived branch across PRs — use sub-branches per PR only if the team prefers; otherwise land in-sequence).
- [ ] Commits squashed into one on merge (project rule: all merges squash-merged).
- [ ] Full verification workflow run for each touched service (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`).
- [ ] Pulled latest `origin/main` and verified no merge conflicts before pushing or merging.
- [ ] If RST source changed: clean-rebuilt Sphinx HTML (`rm -rf build && python3 -m sphinx -M html source build`) and force-added `docsdev/build/`.
- [ ] PR title matches branch name.
- [ ] PR body narrates what and lists `- project-name: change` bullets.
- [ ] No AI attribution in commit messages or PR bodies.
- [ ] No merge until the user explicitly authorizes it.

---

## Execution handoff

Plan complete and saved to `docs/plans/2026-04-24-api-error-response-codes-plan.md`. Two execution options:

**1. Subagent-driven (this session)** — I dispatch a fresh subagent per task, review between tasks, fast iteration. Uses @superpowers:subagent-driven-development.

**2. Parallel session (separate)** — Open a new session in this worktree, it uses @superpowers:executing-plans for batched execution with checkpoints.

Which approach?
