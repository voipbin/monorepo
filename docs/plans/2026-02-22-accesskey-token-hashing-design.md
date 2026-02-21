# Access Key Token Hashing Design

**Date:** 2026-02-22
**Branch:** NOJIRA-hash-accesskey-tokens

## Problem

Access key tokens are stored as plain text in the `customer_accesskeys` table (`token VARCHAR(1023)`). A database breach (SQL injection, backup leak, insider threat) would expose all valid access keys, allowing an attacker to impersonate any customer. Additionally, tokens appear in API responses for GET/List operations, increasing exposure surface.

## Decision

Hash tokens with SHA-256 before storage. Tokens are shown once at creation time and cannot be retrieved again. Add a `vb_` prefix to tokens for identification in logs and secret scanning tools.

**Why SHA-256 over bcrypt:** Access keys are 192 bits of `crypto/rand` entropy. Brute-forcing SHA-256 of a 192-bit random input is computationally infeasible. bcrypt's intentional slowness is designed for low-entropy passwords, not high-entropy API keys. SHA-256 also allows direct indexed DB lookups without a two-step process.

## Token Format

```
vb_<32-char base64url random>
```

- Prefix: `vb_` (identifiable in logs, secret scanners)
- Random part: 32 characters of base64url-encoded `crypto/rand` bytes (~192 bits entropy)
- Total length: 35 characters
- Example: `vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xWe`

## Database Changes

### New Columns

| Column | Type | Purpose |
|--------|------|---------|
| `token_hash` | `CHAR(64)` | `hex(SHA256(full_token))`, indexed for auth lookups |
| `token_prefix` | `VARCHAR(16)` | First 11 chars of token (e.g., `vb_a3Bf9xKm`), display only |

### Removed Columns

| Column | Action |
|--------|--------|
| `token` | Dropped after migration completes |

### Migration (Alembic)

Single migration file with:

**upgrade():**
1. Add `token_hash CHAR(64)` column (nullable initially)
2. Add `token_prefix VARCHAR(16)` column (nullable initially)
3. Backfill: `UPDATE customer_accesskeys SET token_hash = SHA2(token, 256), token_prefix = LEFT(token, 11) WHERE token IS NOT NULL`
4. Add index on `token_hash`
5. Drop index on `token`
6. Drop `token` column

**downgrade():**
1. Add `token VARCHAR(1023)` column back
2. Add index on `token`
3. Drop index on `token_hash`
4. Drop `token_hash` and `token_prefix` columns

Note: downgrade loses the original plain-text tokens since hashing is one-way. This is acceptable — downgrade is for schema rollback, not data recovery.

## Code Changes

### bin-customer-manager

**`models/accesskey/accesskey.go`:**
```go
type Accesskey struct {
    ID         uuid.UUID  `json:"id" db:"id,uuid"`
    CustomerID uuid.UUID  `json:"customer_id" db:"customer_id,uuid"`
    Name       string     `json:"name,omitempty" db:"name"`
    Detail     string     `json:"detail,omitempty" db:"detail"`
    TokenHash  string     `json:"-" db:"token_hash"`
    TokenPrefix string    `json:"token_prefix" db:"token_prefix"`
    TMExpire   *time.Time `json:"tm_expire" db:"tm_expire"`
    TMCreate   *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate   *time.Time `json:"tm_update" db:"tm_update"`
    TMDelete   *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

- Remove `Token` field (was `db:"token"`)
- Add `TokenHash` with `json:"-"` (never exposed in API)
- Add `TokenPrefix` with `json:"token_prefix"`

**`models/accesskey/webhook.go`:**
```go
type WebhookMessage struct {
    // ... existing fields ...
    Token       string `json:"token,omitempty"`       // only set on creation
    TokenPrefix string `json:"token_prefix,omitempty"` // always set
    // Remove old Token field that was always populated
}
```

**`pkg/accesskeyhandler/db.go` (Create):**
1. Generate token: `"vb_" + utilHandler.StringGenerateRandom(32)`
2. Compute hash: `hex(sha256(token))`
3. Extract prefix: `token[:11]`
4. Store `token_hash` and `token_prefix` in DB
5. Set `Token` on the returned WebhookMessage (one-time only)

**`pkg/dbhandler/accesskey.go`:**
- Update all queries to use `token_hash` instead of `token`
- Add `AccesskeyGetByTokenHash(ctx, tokenHash)` method
- Remove `AccesskeyGetByToken` (plain text lookup)

### bin-api-manager

**`pkg/servicehandler/auth.go` (AuthAccesskeyParse):**
1. Receive raw token from request
2. Compute `hex(sha256(token))`
3. Call `CustomerV1AccesskeyGetByTokenHash(ctx, hash)`
4. Validate expiration and deletion as before

**`pkg/servicehandler/accesskeys.go`:**
- `AccesskeyGet` / `AccesskeyList`: Return `token_prefix` only, no `token`
- `AccesskeyCreate`: Return full `token` in response (one-time)

### bin-common-handler

- Add `HashSHA256Hex(input string) string` utility function in `pkg/utilhandler/`

### bin-openapi-manager

- Update `CustomerManagerAccesskey` schema:
  - Remove `token` from required/listed properties
  - Add `token_prefix` (string)
  - Add `token` as optional (only present on creation)

## API Response Examples

**POST /v1.0/accesskeys (Create):**
```json
{
    "id": "2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc",
    "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
    "name": "My API Key",
    "detail": "For accessing reporting APIs",
    "token": "vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xWe",
    "token_prefix": "vb_a3Bf9xKm",
    "tm_expire": "2027-02-22T01:41:40.503790Z",
    "tm_create": "2026-02-22T01:41:40.503790Z",
    "tm_update": "2026-02-22T01:41:40.503790Z",
    "tm_delete": "9999-01-01T00:00:00.000000Z"
}
```

**GET /v1.0/accesskeys (List) and GET /v1.0/accesskeys/{id} (Get):**
```json
{
    "id": "2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc",
    "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
    "name": "My API Key",
    "detail": "For accessing reporting APIs",
    "token_prefix": "vb_a3Bf9xKm",
    "tm_expire": "2027-02-22T01:41:40.503790Z",
    "tm_create": "2026-02-22T01:41:40.503790Z",
    "tm_update": "2026-02-22T01:41:40.503790Z",
    "tm_delete": "9999-01-01T00:00:00.000000Z"
}
```

## RST Documentation Updates

### accesskey_struct.rst
- Replace `token` field with `token_prefix` in example JSON
- Update field descriptions: `token` shown only on creation, `token_prefix` always visible
- Add AI Implementation Hint about one-time token visibility

### accesskey_tutorial.rst
- Update Create response example: show `vb_` prefixed token and `token_prefix`
- Update List/Get response examples: show `token_prefix` only, no `token`
- Add AI Implementation Hint: "The token is shown only once during creation. Store it securely and immediately. If lost, delete the key and create a new one."

### accesskey_overview.rst
- Update example `accesskey` query parameter to use `vb_` prefix format
- Add security note about server-side hashing
- Update Authentication section about one-time visibility

### Rebuild
After RST changes: `cd docsdev && python3 -m sphinx -M html source build`

## Backward Compatibility

- Existing clients using old-format tokens: **tokens stop working** after migration because the `token` column is dropped and old tokens weren't `vb_` prefixed. This is a **breaking change**.
- Alternative: keep old tokens working by hashing them during migration and supporting lookup of both old-hash and new-hash formats. Since old tokens don't have the `vb_` prefix, the auth code can detect format and hash accordingly.
- Decision: Accept the breaking change. Access keys have expiration dates, and customers can create new keys. Communicate the change via release notes.

## Files Changed

| File | Change |
|------|--------|
| `bin-dbscheme-manager/` | New Alembic migration |
| `bin-customer-manager/models/accesskey/accesskey.go` | Update struct |
| `bin-customer-manager/models/accesskey/field.go` | Update Field constants |
| `bin-customer-manager/models/accesskey/webhook.go` | Update WebhookMessage |
| `bin-customer-manager/pkg/accesskeyhandler/db.go` | Hash on create |
| `bin-customer-manager/pkg/dbhandler/accesskey.go` | Query by token_hash |
| `bin-api-manager/pkg/servicehandler/auth.go` | Hash before lookup |
| `bin-api-manager/pkg/servicehandler/accesskeys.go` | Token visibility logic |
| `bin-api-manager/docsdev/source/accesskey_struct.rst` | Update struct docs |
| `bin-api-manager/docsdev/source/accesskey_tutorial.rst` | Update examples |
| `bin-api-manager/docsdev/source/accesskey_overview.rst` | Update overview |
| `bin-common-handler/pkg/utilhandler/` | Add SHA-256 hex helper |
| `bin-openapi-manager/openapi/openapi.yaml` | Update schema |
