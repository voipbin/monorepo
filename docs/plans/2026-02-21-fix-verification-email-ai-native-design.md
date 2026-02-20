# Fix Verification Email for AI-Native Signup

## Problem

The current verification email has three issues that prevent AI agents from completing signup autonomously:

1. **Wrong API path**: Email says `/v1/auth/complete-signup` but the actual route is `/auth/complete-signup` (no `/v1` prefix). See `bin-api-manager/cmd/api-manager/main.go:222`.
2. **No base URL**: Email only shows a relative path, so the AI doesn't know the full API FQDN (`https://api.voipbin.net`).
3. **No request format**: Email doesn't show the HTTP method, content type, or JSON body structure, so the AI doesn't know how to construct the API call.

## Use Case

An AI agent with email access (e.g., via MCP tools) performs the full signup flow autonomously:

1. AI calls `POST /auth/signup` with email/password -> receives `{customer, temp_token}`
2. AI reads the verification email from the user's inbox
3. AI extracts the OTP code and API instructions from the email
4. AI calls `POST /auth/complete-signup` with `{temp_token, code}`

The `temp_token` comes from the signup response (step 1), not the email. The email only needs to deliver the OTP code and clear API instructions.

## Change

Update the email template in `bin-customer-manager/pkg/customerhandler/signup.go` (lines 393-401).

**Before:**
```
Welcome to VoIPBin!

Your verification code is: 138308

API Users: POST this code with your temp_token to /v1/auth/complete-signup

Or click the link below to verify via browser (expires in 1 hour):

https://api.voipbin.net/auth/email-verify?token=...

If you did not create this account, you can safely ignore this email.
```

**After:**
```
Welcome to VoIPBin!

Your verification code is: 138308

To complete signup via API:

POST https://api.voipbin.net/auth/complete-signup
Content-Type: application/json
{"temp_token": "<from signup response>", "code": "138308"}

Or click the link below to verify via browser (expires in 1 hour):

https://api.voipbin.net/auth/email-verify?token=...

If you did not create this account, you can safely ignore this email.
```

## Files Changed

| File | Change |
|------|--------|
| `bin-customer-manager/pkg/customerhandler/signup.go` | Update `sendVerificationEmail` email template (lines 393-401) |

## Implementation Notes

- The base URL comes from `cfg.EmailVerifyBaseURL` (already available in the function at line 389-390)
- `otpCode` appears twice in `fmt.Sprintf` args (display line + JSON body)
- Existing tests use `gomock.Any()` for email subject and content, so no test updates needed
- No other files reference or depend on the email content string
