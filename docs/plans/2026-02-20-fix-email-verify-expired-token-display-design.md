# Fix Email Verify Expired Token Display Bug

## Problem

When a user clicks the "Verify Email" button on the email verification page (`GET /auth/email-verify?token=...`) after the token has expired (1-hour TTL), the browser shows no visible change. The button remains clickable and no error message appears.

The backend correctly returns HTTP 400 and logs: `Could not get verification token. err: token not found or expired`

## Root Cause

In `bin-api-manager/lib/service/signup.go`, the JavaScript `verify()` function sets an inline style before the fetch request:

```javascript
msgEl.className = 'message';
msgEl.style.display = 'none';  // inline style
```

On error, it sets the CSS class but does not clear the inline style:

```javascript
msgEl.className = 'message error';
// inline style="display: none" still set, overrides .message.error { display: block }
```

Inline styles have higher CSS specificity than class-based styles, so `style="display: none"` overrides `.message.error { display: block }` and the error message never becomes visible.

## Fix

**File:** `bin-api-manager/lib/service/signup.go`

1. Remove the `msgEl.style.display = 'none'` line. The `msgEl.className = 'message'` already hides the element via CSS (`.message { display: none }`).

2. On expired/invalid token error: disable the button permanently and change text to "Verification Failed" since retrying an expired token is pointless.

3. On network error: keep the button re-enabled since network errors are transient and retrying makes sense. The error message will now display correctly with the inline style removed.

No signup link is added because this is a generic API - we don't know which client the user signed up from.

## Scope

Single file change: `bin-api-manager/lib/service/signup.go` (the `emailVerifyHTML` template constant).
