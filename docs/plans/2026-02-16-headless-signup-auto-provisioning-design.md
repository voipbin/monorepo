# Headless Signup & Auto-Provisioning Design

## Problem Statement

AI agents and CLI tools cannot currently sign up for VoIPbin and obtain an API key without human intervention. The existing flow requires clicking an email link in a browser, then setting a password via another email link, then logging in to create an AccessKey manually. This blocks programmatic onboarding.

## Design Decisions (from brainstorming)

- **AccessKey creation**: Inline in bin-customer-manager during verification (it already owns both Customer and AccessKey models). Agent creation stays async via event.
- **Token design**: Separate temp_token + 6-digit OTP (two Redis keys per signup session).
- **Web path**: Also auto-provisions AccessKey (both paths get auto-provisioning).
- **Email format**: Plain text with both OTP code and verification link.
- **Password hash**: Keep NOT NULL with random unusable hash (no schema change).
- **Rate limiting**: Redis counter per temp_token, max 5 attempts.
- **Architecture**: All logic in bin-customer-manager (Approach 1).

## Architecture

### Services Touched

| Service | Scope of change |
|---------|----------------|
| bin-customer-manager | New `CompleteSignup()`, modify `Signup()` and `EmailVerify()`, new Redis keys, OTP generation |
| bin-api-manager | New `POST /auth/complete-signup` handler, modify signup + email-verify responses |
| bin-agent-manager | Suppress welcome email when `customer_created` event includes headless flag |
| bin-openapi-manager | New endpoint spec, updated schemas |
| bin-dbscheme-manager | No changes needed |

### Modified Signup Flow

```
POST /auth/signup
  ├── Creates customer (email_verified=false) [unchanged]
  ├── Generates 64-hex verification token + stores in Redis [unchanged]
  ├── Generates 6-digit OTP code (crypto/rand, range 100000-999999)
  ├── Generates temp_token (16 random bytes → 32 hex chars)
  ├── Stores in Redis: signup_session:<temp_token> → {customer_id, otp_code, verify_token} (1hr TTL)
  ├── Sends email with BOTH code + link [modified template]
  └── Returns {temp_token} in response
```

### Path A: Headless (New)

```
POST /auth/complete-signup {temp_token, code}
  ├── Rate limit check: signup_attempts:<temp_token> (max 5)
  ├── Lookup Redis: signup_session:<temp_token>
  ├── Validate OTP code matches
  ├── Mark customer email_verified=true
  ├── Delete Redis keys (session + attempts + email_verify token)
  ├── Create AccessKey via accesskeyhandler.Create()
  ├── Publish customer_created event with headless=true flag
  └── Return {customer_id, accesskey: {id, token}}
```

### Path B: Web (Enhanced)

```
POST /auth/email-verify {token}
  ├── Lookup Redis: email_verify:<token> → customer_id [unchanged]
  ├── Mark customer email_verified=true [unchanged]
  ├── Delete token from Redis [unchanged]
  ├── Create AccessKey via accesskeyhandler.Create()
  ├── Publish customer_created event with headless=false
  └── Return {customer_id, email_verified, accesskey: {id, token}}
```

## API Specification

### POST /v1/auth/signup (modified response)

Response (200 OK):
```json
{
  "message": "Verification code sent to email.",
  "temp_token": "a1b2c3d4e5f6..."
}
```

Always returns 200 to prevent email enumeration (existing behavior preserved). On failure, returns 200 without a valid temp_token.

### POST /v1/auth/complete-signup (new)

Request:
```json
{
  "temp_token": "a1b2c3d4e5f6...",
  "code": "123456"
}
```

Response (200 OK):
```json
{
  "customer_id": "uuid-...",
  "accesskey": {
    "id": "uuid-...",
    "token": "vb_live_abcdef123456..."
  }
}
```

Error responses:
- 400: Invalid/expired temp_token, wrong OTP code
- 429: Too many attempts (> 5 per temp_token)

No authentication required (public endpoint).

### POST /v1/auth/email-verify (modified response)

Response (200 OK):
```json
{
  "customer_id": "uuid-...",
  "email_verified": true,
  "accesskey": {
    "id": "uuid-...",
    "token": "vb_live_abcdef123456..."
  }
}
```

### GET /v1/auth/email-verify (modified HTML)

The HTML page that auto-POSTs and displays success now also displays the AccessKey token in plain text.

## Redis Key Design

| Key pattern | Value | TTL | Purpose |
|-------------|-------|-----|---------|
| `email_verify:<64-hex-token>` | customer_id (string) | 1hr | Existing — web verification link |
| `signup_session:<32-hex-temp-token>` | JSON: `{customer_id, otp_code, verify_token}` | 1hr | New — headless session |
| `signup_attempts:<32-hex-temp-token>` | integer counter | 1hr | New — rate limiting |

All keys created during `Signup()`. Deleted on successful verification (either path). Expire naturally after 1hr if unused.

## Email Template

Subject: `VoIPBin - Verify Your Email (Code: 123456)`

Body (plain text):
```
Welcome to VoIPBin!

Your verification code is: 123456

API Users: POST this code with your temp_token to /v1/auth/complete-signup

Or click the link below to verify via browser (expires in 1 hour):
https://api.voipbin.net/v1/auth/email-verify?token=<64-hex-token>

If you did not create this account, you can safely ignore this email.
```

## Agent Manager: Welcome Email Suppression

The `customer_created` event payload gets a new boolean field `headless`.

- `headless: true` (from complete-signup) → Agent created with random password, welcome email skipped
- `headless: false` (from email-verify or legacy) → Agent created + welcome email sent (existing behavior)

## Security

| Concern | Mitigation |
|---------|-----------|
| OTP brute force (1M combinations) | 5 attempts per temp_token, then locked out |
| temp_token guessing | 32 hex chars = 128 bits of entropy |
| Token reuse | All tokens deleted on successful verification |
| Enumeration | Signup always returns 200 |
| AccessKey exposure | Token shown only once in response |
| TTL expiry | All Redis keys expire in 1 hour |

## What Does NOT Change

- Database schema (no migrations)
- `password_hash` stays NOT NULL (agents get random unusable hash)
- Agent creation flow (still async via event)
- Login flow (JWT, cookies)
- AccessKey authentication mechanism
- Existing AccessKey CRUD endpoints
- Password forgot/reset flow
