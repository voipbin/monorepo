# Auth Login OpenAPI Specification

**Date:** 2026-01-24
**Status:** Implemented

## Overview

Add `/auth/login` endpoint to OpenAPI specification in openapi-manager and implement the corresponding handler in api-manager's generated server pattern.

## Requirements

1. Keep route at `/auth/login` (not `/v1.0/auth/login`)
2. Comprehensive error responses (400, 401, 429, 500)
3. Define in OpenAPI for documentation and code generation
4. Follow existing server implementation patterns

## Design Decisions

### OpenAPI Specification

- **Location:** `bin-openapi-manager/openapi/paths/auth/login.yaml`
- **Tag:** New `Auth` tag added to openapi.yaml
- **Security:** `security: []` to indicate no authentication required
- **Note:** Documentation mentions endpoint is at `/auth/login` not under `/v1.0`

### API-Manager Implementation

- **Handler Location:** `bin-api-manager/server/auth.go`
- **Pattern:** Uses generated types from `openapi_server.PostAuthLoginJSONBody`
- **Route Registration:** Keep manual wiring in `cmd/api-manager/main.go` for `/auth/login` path (the generated `RegisterHandlers` puts routes under `/v1.0`)

## Files Changed

### bin-openapi-manager

1. **Created:** `openapi/paths/auth/login.yaml`
   - POST endpoint definition
   - Request body: username, password
   - Responses: 200, 400, 401, 429, 500

2. **Modified:** `openapi/openapi.yaml`
   - Added `Auth` tag in tags section
   - Added path reference: `/auth/login: $ref: './paths/auth/login.yaml'`

### bin-api-manager

1. **Created:** `server/auth.go`
   - `PostAuthLogin` method implementing `openapi_server.ServerInterface`
   - Uses generated types from openapi_server
   - Follows existing handler patterns (logging, error handling)

2. **Created:** `server/auth_test.go`
   - Tests for successful login
   - Tests for invalid credentials
   - Tests for malformed JSON

3. **Regenerated:** `gens/openapi_server/gen.go`
   - Auto-generated from updated openapi spec
   - Contains `PostAuthLogin` interface method and types

## Verification

```bash
# openapi-manager
cd bin-openapi-manager
go test ./...       # PASS
golangci-lint run   # 0 issues

# api-manager
cd bin-api-manager
go test ./...       # PASS
golangci-lint run   # 0 issues
```

## Notes

- The `/auth/login` endpoint is registered manually in main.go because `openapi_server.RegisterHandlers()` puts all routes under the `/v1.0` prefix
- The existing manual handler at `lib/service/auth.go` can be deprecated once this implementation is verified in production
- The OpenAPI spec server URL is `https://api.voipbin.net/v1.0`, so clients using the spec directly will see `/v1.0/auth/login` in documentation. The spec description clarifies the actual path is `/auth/login`
