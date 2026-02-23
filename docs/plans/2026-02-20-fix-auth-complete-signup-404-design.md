# Fix /auth/complete-signup 404 and Remove /auth/* from OpenAPI

## Problem

`POST /auth/complete-signup` returns 404 for all requests, breaking signup E2E tests
(monorepo-monitoring issue #26). The endpoint exists in the codebase but is only registered
under `/v1.0/auth/complete-signup` behind authentication middleware. Since users have no
credentials during signup, unauthenticated requests get 401 at the v1.0 path and 404 at
the root path (where tests expect it).

Root cause: the headless signup feature added `PostAuthCompleteSignup` as an OpenAPI-generated
handler but never added a corresponding public route in `main.go`.

## Background

`bin-api-manager` has two route registration systems:

1. **Public routes** (no auth) — manually registered at `/auth/*` in `main.go` using handlers
   from `lib/service/`. These serve `/auth/login`, `/auth/signup`, `/auth/email-verify`, etc.

2. **OpenAPI routes** (auth required) — auto-registered at `/v1.0/*` via
   `openapi_server.RegisterHandlers(v1, appServer)` behind `middleware.Authenticate()`.

All 8 `/auth/*` routes in the OpenAPI spec are duplicates of the legacy public handlers. They
get registered at `/v1.0/auth/*` behind auth middleware, making them unusable for
unauthenticated auth flows. The OpenAPI spec itself notes this: "available at `/auth/XXX`
(not under `/v1.0` prefix)".

## Changes

### 1. Add public handler for /auth/complete-signup

**File: `bin-api-manager/lib/service/signup.go`**

Add `PostCustomerCompleteSignup()` function following the existing pattern:

```go
func PostCustomerCompleteSignup(c *gin.Context) {
    // Bind JSON: { temp_token, code }
    // Call serviceHandler.CustomerCompleteSignup()
    // Return 200/400/429
}
```

**File: `bin-api-manager/cmd/api-manager/main.go`**

Register the new public route:

```go
auth.POST("/complete-signup", service.PostCustomerCompleteSignup)
```

### 2. Remove /auth/* paths from OpenAPI spec

**File: `bin-openapi-manager/openapi/openapi.yaml`**

Remove these 6 path entries (lines 5819-5830):
- `/auth/login`
- `/auth/password-forgot`
- `/auth/password-reset`
- `/auth/signup`
- `/auth/email-verify`
- `/auth/complete-signup`

**Delete path files:**
- `bin-openapi-manager/openapi/paths/auth/complete-signup.yaml`
- `bin-openapi-manager/openapi/paths/auth/email-verify.yaml`
- `bin-openapi-manager/openapi/paths/auth/login.yaml`
- `bin-openapi-manager/openapi/paths/auth/password-forgot.yaml`
- `bin-openapi-manager/openapi/paths/auth/password-reset.yaml`
- `bin-openapi-manager/openapi/paths/auth/signup.yaml`

### 3. Remove OpenAPI auth handlers from server package

**Delete these files from `bin-api-manager/server/`:**
- `auth.go` (PostAuthLogin, PostAuthPasswordForgot, GetAuthPasswordReset, PostAuthPasswordReset)
- `auth_test.go`
- `auth_password_test.go`
- `auth_signup.go` (PostAuthSignup, GetAuthEmailVerify, PostAuthEmailVerify)
- `auth_signup_test.go`
- `auth_complete_signup.go` (PostAuthCompleteSignup)
- `auth_complete_signup_test.go`

### 4. Regenerate code

- `cd bin-openapi-manager && go generate ./...`
- `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./...`

### 5. Handle orphaned OpenAPI schemas

After removing auth paths, check if any schemas in `openapi.yaml` (under `components/schemas`)
are now unreferenced. Key schemas to check:
- `AuthLoginResponse`
- `PostAuthCompleteSignupJSONBody`
- `PostAuthSignupJSONBody`

Remove any schemas that are no longer referenced.

## What stays unchanged

- All legacy public handlers in `lib/service/` (auth.go, signup.go, unregister.go)
- All public route registrations in `main.go` (lines 213-227)
- The authenticated `/auth/unregister` routes (POST/DELETE, not in OpenAPI)
- The `servicehandler.CustomerCompleteSignup()` business logic
- The `requesthandler.CustomerV1CustomerCompleteSignup()` RPC call

## Verification

1. Run verification workflow for `bin-openapi-manager` and `bin-api-manager`
2. Run E2E test: `cd monorepo-monitoring/api-validator && .venv/bin/python -m pytest tests/scenarios/test_signup_e2e.py::test_complete_signup_invalid_token -v`
