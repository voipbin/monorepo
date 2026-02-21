# Fix Verification Email for AI-Native Signup — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the verification email template so AI agents can parse the OTP code and construct the complete-signup API call autonomously.

**Architecture:** Single string template change in `sendVerificationEmail`. The base URL comes from the existing `cfg.EmailVerifyBaseURL` config. The `otpCode` is injected twice via `fmt.Sprintf` (display line + JSON body).

**Tech Stack:** Go, fmt.Sprintf

---

### Task 1: Update the verification email template

**Files:**
- Modify: `bin-customer-manager/pkg/customerhandler/signup.go:393-402`

**Step 1: Update the email template**

Replace the `content` assignment (lines 393-402) with:

```go
	content := fmt.Sprintf(
		"Welcome to VoIPBin!\n\n"+
			"Your verification code is: %s\n\n"+
			"To complete signup via API:\n\n"+
			"POST %s/auth/complete-signup\n"+
			"Content-Type: application/json\n"+
			"{\"temp_token\": \"<from signup response>\", \"code\": \"%s\"}\n\n"+
			"Or click the link below to verify via browser (expires in 1 hour):\n\n"+
			"%s\n\n"+
			"If you did not create this account, you can safely ignore this email.",
		otpCode,
		cfg.EmailVerifyBaseURL,
		otpCode,
		verifyLink,
	)
```

Note: `cfg` is already available (line 389). The `otpCode` arg appears twice — once for the display line, once for the JSON body.

**Step 2: Run tests**

```bash
cd bin-customer-manager && go test -v ./pkg/customerhandler/... -run Test_sendVerificationEmail
```

Expected: Both `Test_sendVerificationEmail` and `Test_sendVerificationEmail_error` PASS (tests use `gomock.Any()` for content).

**Step 3: Run full verification workflow**

```bash
cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass, no lint errors.

**Step 4: Commit**

```bash
git add bin-customer-manager/pkg/customerhandler/signup.go
git commit -m "NOJIRA-fix-verification-email-ai-native

- bin-customer-manager: Fix wrong API path in verification email (/v1/auth/ -> /auth/)
- bin-customer-manager: Add full API base URL to verification email
- bin-customer-manager: Add request format (method, content-type, JSON body) to email"
```

### Task 2: Commit design doc and plan

**Files:**
- Add: `docs/plans/2026-02-21-fix-verification-email-ai-native-design.md`
- Add: `docs/plans/2026-02-21-fix-verification-email-ai-native-plan.md`

**Step 1: Stage and commit docs**

```bash
git add docs/plans/2026-02-21-fix-verification-email-ai-native-design.md docs/plans/2026-02-21-fix-verification-email-ai-native-plan.md
git commit -m "NOJIRA-fix-verification-email-ai-native

- docs: Add design doc and implementation plan for AI-native verification email fix"
```

### Task 3: Push and create PR

**Step 1: Fetch main and check for conflicts**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

Expected: No conflicts (changes are isolated to one file + new docs).

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-fix-verification-email-ai-native
```
