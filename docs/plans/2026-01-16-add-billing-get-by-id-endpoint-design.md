# Design: Add GET /v1/billings/{billing-id} Endpoint

**Date:** 2026-01-16
**Status:** Approved

## Overview

Add a new API endpoint to retrieve a single billing record by its UUID. This enables customers to view charge details, admins to investigate billing issues, and integrations to fetch full billing data from webhook events.

## API Specification

### New Endpoint

**Request:**
- Method: `GET`
- Path: `/v1/billings/{billing-id}`
- Path Parameter: `billing-id` (UUID string)
- Authentication: JWT
- Permission: `PermissionCustomerAdmin` only

**Response (Success 200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "6a93f71e-f542-11ee-9a48-7f8011d36229",
  "account_id": "7b94f82f-f643-22ff-9b59-8f9022e47330",
  "status": "end",
  "reference_type": "call",
  "reference_id": "8c95f93g-g754-33gg-0c60-9g0133f58441",
  "cost_per_unit": 0.020,
  "cost_total": 1.40,
  "billing_unit_count": 70.0,
  "tm_billing_start": "2024-01-15 10:30:00.000000",
  "tm_billing_end": "2024-01-15 10:31:10.000000",
  "tm_create": "2024-01-15 10:30:00.000000",
  "tm_update": "2024-01-15 10:31:10.000000"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid UUID format or other errors
- `401 Unauthorized` - Missing/invalid JWT token

## Design Principles

### 1. Atomic API Responses

API responses return ONLY the requested resource type without combining data from other services.

**Exceptions:**
1. Pagination metadata (`next_page_token`) in list responses
2. Atomic operation responses (e.g., POST /v1/calls creating call + groupcall simultaneously)

### 2. Authentication & Authorization

**Architecture:**
```
HTTP Request → bin-api-manager → bin-billing-manager
               (Auth + Permission)  (Data Access)
```

- **bin-api-manager**: JWT validation, permission checks
- **Internal services**: Pure data access, NO auth logic

### 3. Permission Model

**Billing resources require `PermissionCustomerAdmin` ONLY:**
- More restrictive than other resources
- Billing data is sensitive financial information
- Consistent across all billing endpoints

## Implementation

### 1. bin-openapi-manager (OpenAPI Spec)

**Create:** `openapi/paths/billings/id.yaml`

```yaml
get:
  summary: Get a billing record by ID
  description: Retrieves a single billing record by its unique identifier. Returns only the billing record without related account or reference resource data.
  tags:
    - Billing
  parameters:
    - name: billing-id
      in: path
      required: true
      description: The unique identifier of the billing record
      schema:
        type: string
        format: uuid
  responses:
    200:
      description: Successfully retrieved billing record
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/BillingManagerBilling'
    400:
      description: Invalid billing ID format or not found
    401:
      description: Unauthorized - missing or invalid JWT token
```

**Update:** `openapi/openapi.yaml` (add to paths section)

```yaml
/billings/{billing-id}:
  $ref: './paths/billings/id.yaml'
```

**Regenerate:**
```bash
cd bin-openapi-manager
go generate ./...
```

### 2. bin-billing-manager (Backend Service)

**Database layer already exists:**
- `pkg/dbhandler/billing.go:BillingGet(ctx, id)` - No changes needed
- No soft-delete filter (consistent with other single-item GETs)

**Create:** `pkg/listenhandler/v1_billing.go`

```go
package listenhandler

import (
	"context"
	"net/url"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

// processV1BillingGet handles GET /v1/billings/{billing-id} request
func (h *listenHandler) processV1BillingGet(ctx context.Context, m sock.Request) (sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse URI. err: %v", err)
		return simpleResponse(400), nil
	}

	// Extract billing ID from path: /v1/billings/{billing-id}
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) != 3 {
		log.Errorf("Invalid path format. path: %s", u.Path)
		return simpleResponse(400), nil
	}

	billingID := uuid.FromStringOrNil(pathParts[2])
	if billingID == uuid.Nil {
		log.Errorf("Invalid billing ID format. billing_id: %s", pathParts[2])
		return simpleResponse(400), nil
	}

	// Fetch billing from database (no authorization check)
	res, err := h.dbHandler.BillingGet(ctx, billingID)
	if err != nil {
		log.Errorf("Could not get billing. billing_id: %s, err: %v", billingID, err)
		return simpleResponse(404), nil
	}

	return marshalResponse(res)
}
```

**Update:** `pkg/listenhandler/main.go`

Add regex pattern:
```go
regV1BillingGet = regexp.MustCompile(`^/v1/billings/[a-f0-9-]+$`)
```

Add route in `ProcessRequest()`:
```go
case sock.RequestMethodGet:
	if regV1BillingsGet.MatchString(m.URI) {
		requestType = "/v1/billings"
		res, err = h.processV1BillingsGet(ctx, m)
	} else if regV1BillingGet.MatchString(m.URI) {
		requestType = "/v1/billing"
		res, err = h.processV1BillingGet(ctx, m)
	}
```

### 3. bin-common-handler (Request Handler)

**Update:** `pkg/requesthandler/billing_billings.go`

```go
// BillingV1BillingGet returns a single billing record by ID.
func (r *requestHandler) BillingV1BillingGet(ctx context.Context, billingID uuid.UUID) (*bmbilling.Billing, error) {
	uri := fmt.Sprintf("/v1/billings/%s", billingID.String())

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/billing", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res bmbilling.Billing
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Update:** `pkg/requesthandler/main.go` (interface)

```go
BillingV1BillingGet(ctx context.Context, billingID uuid.UUID) (*bmbilling.Billing, error)
```

**Regenerate mocks:**
```bash
cd bin-common-handler
go generate ./pkg/requesthandler
```

### 4. bin-api-manager (API Gateway)

**Update:** `pkg/servicehandler/billings.go`

Fix permission in `BillingGets` (line 31):
```go
if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
	return nil, fmt.Errorf("user has no permission")
}
```

Add new methods:
```go
// billingGet validates the billing's ownership and returns the billing info.
func (h *serviceHandler) billingGet(ctx context.Context, billingID uuid.UUID) (*bmbilling.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "billingGet",
		"billing_id": billingID,
	})

	res, err := h.reqHandler.BillingV1BillingGet(ctx, billingID)
	if err != nil {
		log.Errorf("Could not get the billing info. err: %v", err)
		return nil, err
	}
	log.WithField("billing", res).Debug("Received result.")

	return res, nil
}

// BillingGet sends a request to billing-manager to get a billing.
func (h *serviceHandler) BillingGet(ctx context.Context, a *amagent.Agent, billingID uuid.UUID) (*bmbilling.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"billing_id":  billingID,
	})

	b, err := h.billingGet(ctx, billingID)
	if err != nil {
		log.Infof("Could not get billing info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, b.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := b.ConvertWebhookMessage()
	return res, nil
}
```

**Update:** `pkg/servicehandler/main.go` (interface)

```go
BillingGet(ctx context.Context, a *amagent.Agent, billingID uuid.UUID) (*bmbilling.WebhookMessage, error)
```

**Update:** `server/billings.go`

```go
func (h *server) GetBilling(c *gin.Context, billingId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetBilling",
		"request_address": c.ClientIP(),
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	billingID := uuid.FromStringOrNil(billingId)
	if billingID == uuid.Nil {
		log.Errorf("Invalid billing ID format.")
		c.AbortWithStatus(400)
		return
	}

	billing, err := h.serviceHandler.BillingGet(c.Request.Context(), &a, billingID)
	if err != nil {
		logrus.Errorf("Could not get billing info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, billing)
}
```

**Regenerate Swagger:**
```bash
cd bin-api-manager
swag init -g cmd/api-manager/main.go -o docsapi
```

### 5. Update All Services

```bash
cd /home/pchero/gitvoipbin/monorepo

# Update services to pull new common-handler
find . -maxdepth 2 -name "go.mod" -not -path "./bin-openapi-manager/*" \
  -not -path "./bin-billing-manager/*" -not -path "./bin-common-handler/*" \
  -not -path "./bin-api-manager/*" -execdir bash -c \
  "echo 'Updating \$(basename \$(pwd))...' && \
   go mod tidy && go mod vendor" \;
```

## Verification Workflow

```bash
# Step 1: OpenAPI
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./...
go test ./...

# Step 2: Billing Manager
cd ../bin-billing-manager
go mod tidy && go mod vendor && go generate ./...
go test ./...
golangci-lint run -v --timeout 5m

# Step 3: Common Handler
cd ../bin-common-handler
go mod tidy && go mod vendor && go generate ./...
go test ./...

# Step 4: API Manager
cd ../bin-api-manager
go mod tidy && go mod vendor && go generate ./...
swag init -g cmd/api-manager/main.go -o docsapi
go test ./...
golangci-lint run -v --timeout 5m
```

## Testing

**Manual testing:**
```bash
# Get JWT token
TOKEN=$(curl -X POST https://api.voipbin.net/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user","password":"pass"}' | jq -r .token)

# Test valid billing retrieval
curl -X GET https://api.voipbin.net/v1/billings/{billing-id} \
  -H "Authorization: Bearer $TOKEN"

# Test invalid UUID
curl -X GET https://api.voipbin.net/v1/billings/invalid-uuid \
  -H "Authorization: Bearer $TOKEN"
```

## Files Changed

**Created:**
1. `bin-openapi-manager/openapi/paths/billings/id.yaml`
2. `bin-billing-manager/pkg/listenhandler/v1_billing.go`
3. `docs/plans/2026-01-16-add-billing-get-by-id-endpoint-design.md`

**Modified:**
1. `bin-openapi-manager/openapi/openapi.yaml` - Add path reference
2. `bin-billing-manager/pkg/listenhandler/main.go` - Add regex and route
3. `bin-common-handler/pkg/requesthandler/billing_billings.go` - Add method
4. `bin-common-handler/pkg/requesthandler/main.go` - Add interface method
5. `bin-api-manager/pkg/servicehandler/billings.go` - Fix permission + add methods
6. `bin-api-manager/pkg/servicehandler/main.go` - Add interface method
7. `bin-api-manager/server/billings.go` - Add HTTP handler

**No database migrations required** - Using existing `billing_billings` table and `BillingGet()` method.
