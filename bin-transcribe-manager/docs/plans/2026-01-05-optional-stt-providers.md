# Optional STT Provider Configuration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make GCP and AWS STT provider credentials optional (at least one required) instead of both being mandatory.

**Architecture:** Conditional client initialization at startup with dynamic handler list at runtime. GCP uses ADC for credentials, AWS uses explicit keys. Service fails if neither provider is available.

**Tech Stack:** Go 1.x, Google Cloud Speech-to-Text API, AWS Transcribe Streaming, go.uber.org/mock for testing

---

## Task 1: Add nil-check defense to gcpRun

**Files:**
- Modify: `pkg/streaminghandler/gcp.go:16-22`

**Step 1: Add nil-check at start of gcpRun**

Add the check immediately after the function signature, before creating the logger:

```go
func (h *streamingHandler) gcpRun(st *streaming.Streaming, conn net.Conn) error {
	if h.gcpClient == nil {
		return fmt.Errorf("GCP provider not initialized")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "gcpRun",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
```

**Step 2: Add fmt import if needed**

Check if `fmt` is already imported at the top of the file. If not, add it to the import block.

**Step 3: Run tests to verify no breakage**

Run: `go test -v ./pkg/streaminghandler -run Test_Start`

Expected: PASS (existing tests should still work)

**Step 4: Commit**

```bash
git add pkg/streaminghandler/gcp.go
git commit -m "feat: add nil-check defense to gcpRun

Add defensive nil-check for gcpClient to prevent panics
when GCP provider is not initialized.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Add nil-check defense to awsRun

**Files:**
- Modify: `pkg/streaminghandler/aws.go:35-40`

**Step 1: Add nil-check at start of awsRun**

Add the check immediately after the function signature, before creating the logger:

```go
func (h *streamingHandler) awsRun(st *streaming.Streaming, conn net.Conn) error {
	if h.awsClient == nil {
		return fmt.Errorf("AWS provider not initialized")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "awsRun",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
```

**Step 2: Run tests to verify no breakage**

Run: `go test -v ./pkg/streaminghandler -run Test_Start`

Expected: PASS (existing tests should still work)

**Step 3: Commit**

```bash
git add pkg/streaminghandler/aws.go
git commit -m "feat: add nil-check defense to awsRun

Add defensive nil-check for awsClient to prevent panics
when AWS provider is not initialized.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Implement dynamic handler list in runStart

**Files:**
- Modify: `pkg/streaminghandler/run.go:70-83`

**Step 1: Replace static handler list with dynamic construction**

Replace lines 70-73 with dynamic handler building:

```go
// Build handlers list dynamically based on available clients
handlers := []func(*streaming.Streaming, net.Conn) error{}
if h.gcpClient != nil {
	handlers = append(handlers, h.gcpRun)
}
if h.awsClient != nil {
	handlers = append(handlers, h.awsRun)
}

if len(handlers) == 0 {
	log.Error("No STT providers available for transcription")
	return
}
```

The existing loop (lines 75-81) remains unchanged.

**Step 2: Run tests to verify no breakage**

Run: `go test -v ./pkg/streaminghandler -run Test_Start`

Expected: PASS (tests with both clients initialized should work)

**Step 3: Commit**

```bash
git add pkg/streaminghandler/run.go
git commit -m "feat: implement dynamic handler list in runStart

Build handler list dynamically based on available clients.
Only include handlers for initialized providers.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Implement conditional client initialization in NewStreamingHandler

**Files:**
- Modify: `pkg/streaminghandler/main.go:93-107`

**Step 1: Add strings import**

Add `strings` to the import list at the top of the file (around line 5-27).

**Step 2: Replace unconditional client creation with conditional logic**

Replace lines 95-107 with the new conditional initialization logic:

```go
// Try to create GCP client (ADC-based)
gcpClient, err := speech.NewClient(context.Background())
if err != nil {
	log.Warnf("GCP client initialization failed (credentials not available): %v", err)
	gcpClient = nil
}

// Only try AWS if credentials are provided
var awsClient *transcribestreaming.Client
if awsAccessKey != "" && awsSecretKey != "" {
	awsClient, err = awsNewClient(awsAccessKey, awsSecretKey)
	if err != nil {
		log.Warnf("AWS client initialization failed: %v", err)
		awsClient = nil
	}
} else {
	log.Debug("AWS credentials not provided - AWS provider will be unavailable")
	awsClient = nil
}

// Validate at least one provider is available
var providers []string
if gcpClient != nil {
	providers = append(providers, "GCP")
}
if awsClient != nil {
	providers = append(providers, "AWS")
}
if len(providers) == 0 {
	log.Error("No STT providers available - at least one provider must be configured")
	return nil
}
log.Infof("STT providers initialized: %s", strings.Join(providers, ", "))
```

The return statement (lines 109-122) remains unchanged.

**Step 3: Run all tests**

Run: `go test -v ./pkg/streaminghandler`

Expected: PASS (existing tests should still work with both clients)

**Step 4: Commit**

```bash
git add pkg/streaminghandler/main.go
git commit -m "feat: implement conditional STT provider initialization

Make GCP and AWS client initialization optional:
- GCP: Always attempt via ADC, accept graceful failure
- AWS: Only attempt if credentials provided
- Require at least one provider available
- Log which providers were initialized

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Add test for NewStreamingHandler with only AWS

**Files:**
- Create: `pkg/streaminghandler/main_test.go` (if doesn't exist)
- Modify: `pkg/streaminghandler/main_test.go` (if exists)

**Step 1: Check if test file exists**

Run: `ls pkg/streaminghandler/main_test.go`

If file doesn't exist, create it with package declaration and imports. If it exists, add the new test to the existing file.

**Step 2: Create test for AWS-only scenario**

Add this test function:

```go
func TestNewStreamingHandler_AWSOnly(t *testing.T) {
	// This test verifies service works with only AWS credentials
	// GCP will fail to initialize (no credentials in test env)
	// AWS should succeed with valid test credentials

	reqHandler := requesthandler.NewRequestHandler(nil, "test-service")
	notifyHandler := notifyhandler.NewNotifyHandler(nil, reqHandler, "test-queue", "test-service")
	transcriptHandler := transcripthandler.NewTranscriptHandler(reqHandler, nil, notifyHandler)

	handler := NewStreamingHandler(
		reqHandler,
		notifyHandler,
		transcriptHandler,
		"127.0.0.1:8080",
		"test_access_key",
		"test_secret_key",
	)

	// Should return valid handler (AWS initialized, GCP may fail gracefully)
	if handler == nil {
		t.Fatal("Expected handler to be non-nil with AWS credentials")
	}
}
```

**Step 3: Create test for neither provider available**

Add this test function:

```go
func TestNewStreamingHandler_NoProviders(t *testing.T) {
	// This test verifies service fails when neither provider is available
	// GCP will fail (no credentials in test env)
	// AWS will fail (empty credentials)

	reqHandler := requesthandler.NewRequestHandler(nil, "test-service")
	notifyHandler := notifyhandler.NewNotifyHandler(nil, reqHandler, "test-queue", "test-service")
	transcriptHandler := transcripthandler.NewTranscriptHandler(reqHandler, nil, notifyHandler)

	handler := NewStreamingHandler(
		reqHandler,
		notifyHandler,
		transcriptHandler,
		"127.0.0.1:8080",
		"", // empty AWS credentials
		"",
	)

	// Should return nil when no providers available
	if handler != nil {
		t.Error("Expected handler to be nil when no providers available")
	}
}
```

**Step 4: Run the new tests**

Run: `go test -v ./pkg/streaminghandler -run TestNewStreamingHandler`

Expected: Both tests should PASS

**Step 5: Commit**

```bash
git add pkg/streaminghandler/main_test.go
git commit -m "test: add tests for optional provider initialization

Add tests for:
- AWS-only configuration (GCP fails gracefully)
- No providers configured (service fails)

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Run full test suite and verify

**Step 1: Run all streaminghandler tests**

Run: `go test -v ./pkg/streaminghandler`

Expected: All tests PASS

**Step 2: Run all tests in the service**

Run: `go test -v ./...`

Expected: All tests PASS

**Step 3: Run go vet**

Run: `go vet ./...`

Expected: No issues

**Step 4: Build the service**

Run: `go build -o ./bin/transcribe-manager ./cmd/transcribe-manager`

Expected: Build succeeds

---

## Task 7: Manual verification test plan

**Test Scenario 1: Both providers configured (backwards compatibility)**

Environment:
- `GOOGLE_APPLICATION_CREDENTIALS` set (or other ADC method)
- `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` set

Expected behavior:
- Service starts successfully
- Log shows: "STT providers initialized: GCP, AWS"
- Runtime uses GCP first, falls back to AWS on failure

**Test Scenario 2: Only AWS configured**

Environment:
- No GCP credentials (unset `GOOGLE_APPLICATION_CREDENTIALS`)
- `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` set

Expected behavior:
- Service starts successfully
- Log shows: "GCP client initialization failed (credentials not available): ..."
- Log shows: "STT providers initialized: AWS"
- Runtime uses only AWS

**Test Scenario 3: Only GCP configured**

Environment:
- `GOOGLE_APPLICATION_CREDENTIALS` set (or other ADC method)
- `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` empty/unset

Expected behavior:
- Service starts successfully
- Log shows: "AWS credentials not provided - AWS provider will be unavailable"
- Log shows: "STT providers initialized: GCP"
- Runtime uses only GCP

**Test Scenario 4: Neither configured (should fail)**

Environment:
- No GCP credentials
- `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` empty/unset

Expected behavior:
- Service fails to start
- Log shows: "No STT providers available - at least one provider must be configured"
- Exit with error

---

## Task 8: Update CLAUDE.md documentation

**Files:**
- Modify: `CLAUDE.md` (around line 40-50, "Configuration via Environment Variables" section)

**Step 1: Update AWS credentials description**

Change the existing line:
```
- `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`: AWS credentials for Transcribe
```

To:
```
- `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`: AWS credentials for Transcribe (optional if GCP configured)
```

**Step 2: Add note about provider requirements**

Add after the environment variables list:

```markdown
**STT Provider Requirements:**
- At least one STT provider must be configured (GCP or AWS)
- GCP: Uses Application Default Credentials (ADC) - can be from service account key, gcloud CLI, GKE metadata server, etc.
- AWS: Requires both `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` environment variables
- If both are configured, GCP is tried first with AWS as fallback
- Service fails to start if neither provider is available
```

**Step 3: Commit documentation**

```bash
git add CLAUDE.md
git commit -m "docs: update provider configuration requirements

Document that STT providers are now optional (at least
one required) instead of both being mandatory.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Success Criteria Checklist

- [ ] Service starts successfully with only GCP credentials configured
- [ ] Service starts successfully with only AWS credentials configured
- [ ] Service starts successfully with both credentials configured (existing behavior)
- [ ] Service fails to start with clear error message if neither provider is configured
- [ ] GCP is tried first, AWS as fallback when both available
- [ ] Runtime transcription works with single provider (no fallback available)
- [ ] All existing tests pass
- [ ] New tests added for optional provider scenarios
- [ ] Documentation updated
- [ ] No breaking changes to API or behavior

## Notes

- The implementation follows TDD principles where possible
- Each task is small and focused (2-5 minutes)
- Frequent commits ensure incremental progress is saved
- Defense-in-depth: Nil-checks in both initialization and runtime
- Backwards compatible: Existing deployments work unchanged
