# Design: Expose RTPDebug to Normal Customers

**Date:** 2026-03-11
**Branch:** NOJIRA-Expose-rtp-debug-to-customer

## Problem

The `rtp_debug` flag is only visible and editable by ProjectSuperAdmin via `PUT /customers/{id}/metadata`. Normal customers cannot see or control RTP packet capture for their own calls, even though it's a debugging tool useful for customers diagnosing audio issues.

## Goal

Allow CustomerAdmin to view and update the `rtp_debug` flag for their own customer account through the self-service `/customer` API.

## Approach

- **Read:** Add `metadata` field to the `WebhookMessage` struct so `GET /customer` responses include it.
- **Write:** Create a new `PUT /customer/metadata` endpoint (following the existing `PUT /customer/billing_account_id` pattern) with CustomerAdmin permission.
- The existing admin endpoint `PUT /customers/{id}/metadata` stays unchanged (ProjectSuperAdmin-only).

## Changes Required

### 1. Customer WebhookMessage (`bin-customer-manager/models/customer/webhook.go`)

- Add `Metadata` field to the `WebhookMessage` struct.
- Update `ConvertWebhookMessage()` to copy the `Metadata` field from the internal `Customer` struct.

### 2. Customer Metadata Comment (`bin-customer-manager/models/customer/metadata.go`)

- Update the comment that says "Not exposed in WebhookMessage" — it will now be exposed.

### 3. OpenAPI Spec (`bin-openapi-manager`)

- Add `metadata` property (referencing `CustomerManagerMetadata`) to the `CustomerManagerCustomer` schema in `openapi/openapi.yaml`.
- Create new path file `openapi/paths/customer/metadata.yaml` defining `PUT /customer/metadata`:
  - Request body: `{ "rtp_debug": boolean }` (optional fields, same as `CustomerManagerMetadata`)
  - Response: `CustomerManagerCustomer`
  - Follow the `billing_account_id.yaml` pattern.
- Register the new path in the main OpenAPI spec.

### 4. API Server Handler (`bin-api-manager/server/customer.go`)

- Add `PutCustomerMetadata()` handler:
  - Parse `PutCustomerMetadataJSONRequestBody` from request.
  - Build `cucustomer.Metadata` from the request.
  - Call `h.serviceHandler.CustomerSelfUpdateMetadata()`.
  - Return the `WebhookMessage` response.

### 5. Service Handler (`bin-api-manager/pkg/servicehandler/customer.go`)

- Add `CustomerSelfUpdateMetadata()` method:
  - Check `PermissionCustomerAdmin` permission.
  - Call `h.reqHandler.CustomerV1CustomerUpdateMetadata(ctx, a.CustomerID, metadata)`.
  - Convert the result to `WebhookMessage` and return.
  - Follow the `CustomerSelfUpdateBillingAccountID` pattern.

### 6. Regenerate

- `go generate ./...` in `bin-openapi-manager`.
- `go generate ./...` in `bin-api-manager`.

### 7. Tests

- Add server handler test for `PutCustomerMetadata` in `bin-api-manager/server/customer_test.go`.
- Add service handler test for `CustomerSelfUpdateMetadata`.

## What Stays the Same

- `PUT /customers/{id}/metadata` — ProjectSuperAdmin-only, unchanged.
- `/customers/{id}` endpoints — admin-only, unchanged.
- `bin-customer-manager` backend — reuses existing `UpdateMetadata` / `processV1CustomersIDMetadataPut` logic. No changes needed.
- Call-level RTP debug logic in `bin-call-manager` — unchanged.

## Permission Model

| Endpoint | Read | Write |
|----------|------|-------|
| `GET /customer` | CustomerAdmin / CustomerManager | — |
| `PUT /customer/metadata` | — | CustomerAdmin |
| `PUT /customers/{id}/metadata` | — | ProjectSuperAdmin |

## Side Effects

Adding `Metadata` to `WebhookMessage` means all endpoints returning `WebhookMessage` will now include the metadata field. This affects:
- `GET /customer`
- `PUT /customer`
- `PUT /customer/billing_account_id`
- `PUT /customer/metadata` (new)

This is acceptable since `rtp_debug` is not sensitive — the customer controls it.

## Files to Modify

| File | Change |
|------|--------|
| `bin-customer-manager/models/customer/webhook.go` | Add Metadata to WebhookMessage + ConvertWebhookMessage() |
| `bin-customer-manager/models/customer/metadata.go` | Update comment |
| `bin-openapi-manager/openapi/openapi.yaml` | Add metadata to CustomerManagerCustomer schema |
| `bin-openapi-manager/openapi/paths/customer/metadata.yaml` | New file: PUT /customer/metadata path |
| `bin-api-manager/server/customer.go` | Add PutCustomerMetadata handler |
| `bin-api-manager/pkg/servicehandler/customer.go` | Add CustomerSelfUpdateMetadata method |
| `bin-api-manager/server/customer_test.go` | Add test for new handler |
