# Fix Email Verify Expired Token Display - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the email verification page so expired token errors are actually visible to users.

**Architecture:** Remove an inline `style.display = 'none'` that overrides CSS class-based `display: block`, and improve the error UX by permanently disabling the button on token errors.

**Tech Stack:** Go (HTML template string), JavaScript, CSS

---

### Task 1: Add test for HTML error display behavior

**Files:**
- Modify: `bin-api-manager/lib/service/signup_test.go`

**Step 1: Write a test that verifies the HTML template does NOT contain the inline style bug**

Add to the existing `TestGetCustomerEmailVerify` function's valid token case, or add a new focused test:

```go
func TestGetCustomerEmailVerify_HTMLContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/auth/email-verify", GetCustomerEmailVerify)

	req, _ := http.NewRequest("GET", "/auth/email-verify?token=a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6abcd", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got: %d", w.Code)
	}

	body := w.Body.String()

	// Verify the inline style bug is NOT present.
	// The old code had `msgEl.style.display = 'none'` which overrides CSS class-based display.
	if strings.Contains(body, "msgEl.style.display") {
		t.Error("HTML should not set inline style.display - it overrides CSS class-based visibility")
	}

	// Verify error case disables button permanently (not re-enabled)
	if strings.Contains(body, `btn.disabled = false`) {
		// The error (non-network) case should NOT re-enable the button.
		// Only the catch (network error) case should re-enable.
		// Count occurrences: should be exactly 1 (in catch block only)
		count := strings.Count(body, `btn.disabled = false`)
		if count > 1 {
			t.Errorf("Expected btn.disabled = false only in catch block (1 occurrence), found %d", count)
		}
	}

	// Verify "Verification Failed" text appears for error case
	if !strings.Contains(body, "Verification Failed") {
		t.Error("Expected 'Verification Failed' button text in error handler")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-api-manager && go test -v ./lib/service/ -run TestGetCustomerEmailVerify_HTMLContent`

Expected: FAIL — current HTML contains `msgEl.style.display`, has 2 occurrences of `btn.disabled = false`, and lacks "Verification Failed".

---

### Task 2: Fix the HTML template

**Files:**
- Modify: `bin-api-manager/lib/service/signup.go:183-214` (the `<script>` section of `emailVerifyHTML`)

**Step 1: Apply three changes to the JavaScript inside `emailVerifyHTML`**

Change the script block (lines 183-214) from:

```javascript
<script>
  var token = "%s";
  function verify() {
    var btn = document.getElementById('verifyBtn');
    var msgEl = document.getElementById('message');
    btn.disabled = true;
    btn.textContent = 'Verifying...';
    msgEl.className = 'message';
    msgEl.style.display = 'none';

    fetch('/auth/email-verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token })
    }).then(function(resp) {
      if (resp.ok) {
        msgEl.textContent = 'Email verified successfully! Check your inbox for a welcome email with instructions to set your password.';
        msgEl.className = 'message success';
        btn.style.display = 'none';
      } else {
        msgEl.textContent = 'Invalid or expired verification link. Please sign up again.';
        msgEl.className = 'message error';
        btn.disabled = false;
        btn.textContent = 'Verify Email';
      }
    }).catch(function() {
      msgEl.textContent = 'An error occurred. Please try again.';
      msgEl.className = 'message error';
      btn.disabled = false;
      btn.textContent = 'Verify Email';
    });
  }
</script>
```

To:

```javascript
<script>
  var token = "%s";
  function verify() {
    var btn = document.getElementById('verifyBtn');
    var msgEl = document.getElementById('message');
    btn.disabled = true;
    btn.textContent = 'Verifying...';
    msgEl.className = 'message';

    fetch('/auth/email-verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token })
    }).then(function(resp) {
      if (resp.ok) {
        msgEl.textContent = 'Email verified successfully! Check your inbox for a welcome email with instructions to set your password.';
        msgEl.className = 'message success';
        btn.style.display = 'none';
      } else {
        msgEl.textContent = 'Your verification link has expired or is invalid. Please sign up again.';
        msgEl.className = 'message error';
        btn.textContent = 'Verification Failed';
      }
    }).catch(function() {
      msgEl.textContent = 'An error occurred. Please try again.';
      msgEl.className = 'message error';
      btn.disabled = false;
      btn.textContent = 'Verify Email';
    });
  }
</script>
```

Three changes:
1. **Remove line `msgEl.style.display = 'none';`** — `msgEl.className = 'message'` already hides via CSS. The inline style was overriding `.message.error { display: block }`.
2. **In the `else` block (token error):** Remove `btn.disabled = false;` (keep button disabled) and change `btn.textContent` from `'Verify Email'` to `'Verification Failed'`. Updated error message text.
3. **In the `catch` block (network error):** No change — button stays re-enabled since network errors are transient.

**Step 2: Run the test from Task 1 to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-api-manager && go test -v ./lib/service/ -run TestGetCustomerEmailVerify_HTMLContent`

Expected: PASS

**Step 3: Run ALL existing tests to verify no regressions**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-api-manager && go test -v ./lib/service/`

Expected: All tests PASS

---

### Task 3: Run full verification workflow and commit

**Step 1: Run the full verification workflow**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass with zero errors.

**Step 2: Commit**

```bash
git add bin-api-manager/lib/service/signup.go bin-api-manager/lib/service/signup_test.go docs/plans/2026-02-20-fix-email-verify-expired-token-display-design.md docs/plans/2026-02-20-fix-email-verify-expired-token-display-plan.md
git commit -m "NOJIRA-fix-email-verify-expired-token-display

Fix email verification page not showing error when token is expired.
The inline style.display='none' was overriding the CSS class-based
display:block, making the error message invisible to users.

- bin-api-manager: Remove inline style.display that overrides CSS error visibility
- bin-api-manager: Disable button permanently on expired token error
- bin-api-manager: Add test verifying HTML error display behavior
- docs: Add design and plan documents"
```
