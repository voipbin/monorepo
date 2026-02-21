# Design: Add Terms of Service Agreement Fields to Customer

## Problem Statement

We need to record when customers agree to the Terms of Service during signup for compliance and audit purposes. This requires:
1. Two new fields on the customer model: `terms_agreed_version` (datetime string, to evolve into version identifier later) and `terms_agreed_ip` (IP address of the agreeing client).
2. A required `accepted_tos` boolean in the signup request — if `false` or absent, reject the request.
3. These fields are internal only — not exposed in API responses (WebhookMessage/OpenAPI).

## Approach

Add fields to the existing customer model and populate them during the initial `Signup()` call. The client IP flows from `gin.Context.ClientIP()` in api-manager through the RPC chain to customer-manager.

## Changes

### 1. Customer Model (`bin-customer-manager/models/customer/customer.go`)
- Add `TermsAgreedVersion string` with `db:"terms_agreed_version"` tag
- Add `TermsAgreedIP string` with `db:"terms_agreed_ip"` tag

### 2. Customer Field Constants (`bin-customer-manager/models/customer/field.go`)
- Add `FieldTermsAgreedVersion Field = "terms_agreed_version"`
- Add `FieldTermsAgreedIP Field = "terms_agreed_ip"`

### 3. Database Migration (`bin-dbscheme-manager`)
- Add Alembic migration: `ALTER TABLE customer_customers ADD COLUMN terms_agreed_version VARCHAR(64) NOT NULL DEFAULT '' AFTER status` and `terms_agreed_ip VARCHAR(45) NOT NULL DEFAULT '' AFTER terms_agreed_version`

### 4. Signup Request Changes

**API-Manager (`bin-api-manager/lib/service/signup.go`):**
- Add `AcceptedTOS bool` to `RequestBodySignupPOST` with `json:"accepted_tos" binding:"required"`
- Validate `accepted_tos == true` before proceeding; return 400 if false
- Extract `c.ClientIP()` and pass it through the call chain

**API-Manager ServiceHandler (`bin-api-manager/pkg/servicehandler/customer.go`):**
- Add `clientIP string` parameter to `CustomerSignup()`

**RequestHandler (`bin-common-handler/pkg/requesthandler/`):**
- Add `clientIP string` parameter to `CustomerV1CustomerSignup()`
- Include `client_ip` in the RPC request data

**Customer-Manager ListenHandler (`bin-customer-manager/pkg/listenhandler/`):**
- Parse `client_ip` from the signup request data
- Pass to `customerHandler.Signup()`

**Customer-Manager Request Model (`bin-customer-manager/pkg/listenhandler/models/request/customers.go`):**
- Add `ClientIP string` and `AcceptedTOS bool` to `V1DataCustomersSignupPost`

**Customer-Manager CustomerHandler (`bin-customer-manager/pkg/customerhandler/signup.go`):**
- Add `clientIP string` parameter to `Signup()`
- Set `TermsAgreedVersion` to current datetime string (RFC3339)
- Set `TermsAgreedIP` to the provided client IP
- These fields are set on the customer struct before `CustomerCreate()`

### 5. Additional Changes (added during implementation)
- `WebhookMessage` — Added `EmailVerifyResultWebhookMessage` and `CompleteSignupResultWebhookMessage` to prevent `TermsAgreedIP` leaking through `EmailVerifyResult` and `CompleteSignupResult` API responses
- OpenAPI schema — Added definitions for `/auth/signup`, `/auth/email-verify`, `/auth/complete-signup` endpoints with request/response schemas

### 6. No Changes To
- `FilterStruct` — not filterable
- `dbhandler` — existing `CustomerCreate()` with `PrepareFields()` will pick up new fields automatically via db tags

## Trade-offs

- **DateTime string vs dedicated timestamp field**: Using a string allows future evolution to a version identifier without a migration. For now it stores RFC3339 datetime.
- **accepted_tos validation in api-manager vs customer-manager**: Validating in api-manager is simpler and consistent with the auth pattern (api-manager handles request validation, customer-manager handles business logic). The `accepted_tos` field is not stored — it's only used for request-time validation.
