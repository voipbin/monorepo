# Add OpenAPI Definitions for Auth Signup Endpoints - Design

## Goal

Document the 3 existing auth signup endpoints in the OpenAPI spec, add WebhookMessage variants for `EmailVerifyResult` and `CompleteSignupResult` to prevent internal field leaks, and fix related bugs found during code review.

## Endpoints

1. **POST /auth/signup** — Public. Request: name, detail, email, phone_number, address, webhook_method, webhook_uri, accepted_tos. Response: `SignupResultWebhookMessage` (customer + temp_token) or empty `{}` on failure.
2. **POST /auth/email-verify** — Public. Request: token (64-char hex). Response: `EmailVerifyResultWebhookMessage` (customer + accesskey).
3. **POST /auth/complete-signup** — Public. Request: temp_token, code. Response: `CompleteSignupResultWebhookMessage` (customer_id + accesskey). Also returns 429.
4. **GET /auth/email-verify** — HTML page, excluded from OpenAPI.

## Changes

### bin-customer-manager (model layer)
- Add `EmailVerifyResultWebhookMessage` + `ConvertWebhookMessage()` in `models/customer/signup.go`
- Add `CompleteSignupResultWebhookMessage` + `ConvertWebhookMessage()` in `models/customer/signup.go`
- Add tests for both new WebhookMessage types

### bin-api-manager (servicehandler layer)
- Update `CustomerEmailVerify` return type to `*cscustomer.EmailVerifyResultWebhookMessage`, add `.ConvertWebhookMessage()` call
- Update `CustomerCompleteSignup` return type to `*cscustomer.CompleteSignupResultWebhookMessage`, add `.ConvertWebhookMessage()` call
- Fix `c.ClientIP` -> `c.ClientIP()` in signup.go log fields (pre-existing bug)

### bin-openapi-manager (OpenAPI spec)
- Create path files: `signup.yaml`, `email-verify.yaml`, `complete-signup.yaml`
- Add schemas: `RequestBodyAuthSignupPOST`, `RequestBodyAuthEmailVerifyPOST`, `RequestBodyAuthCompleteSignupPOST`, `CustomerManagerSignupResult`, `CustomerManagerEmailVerifyResult`, `CustomerManagerCompleteSignupResult`
- Add path references in `openapi.yaml`

### Bug fixes
- Fix test SQL column ordering to match migration (`AFTER status`)
- Fix `c.ClientIP` method reference -> `c.ClientIP()` call in signup.go
- Update design doc migration spec to match implementation
