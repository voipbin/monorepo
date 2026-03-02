# Validate tts_type and stt_type Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add input validation for `tts_type` and `stt_type` fields in ai-manager so invalid values are rejected at the API layer with helpful error messages.

**Architecture:** Map-based `IsValid()` and `ValidValues()` methods on `TTSType` and `STTType` types. Validation in `aihandler.Create()` / `Update()` and the `ai-control` CLI, consistent with existing `engineModel` validation.

**Tech Stack:** Go, table-driven tests, gomock

---

### Task 1: Add TTSType validation methods and tests

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go:160-197`
- Modify: `bin-ai-manager/models/ai/main_test.go` (append new tests)

**Step 1: Write the failing tests for TTSType.IsValid() and TTSType.ValidValues()**

Append to `bin-ai-manager/models/ai/main_test.go`:

```go
func TestTTSTypeIsValid(t *testing.T) {
	tests := []struct {
		name     string
		ttsType  TTSType
		expected bool
	}{
		{"empty_string_is_valid", TTSTypeNone, true},
		{"google_is_valid", TTSTypeGoogle, true},
		{"openai_is_valid", TTSTypeOpenAI, true},
		{"elevenlabs_is_valid", TTSTypeElevenLabs, true},
		{"cartesia_is_valid", TTSTypeCartesia, true},
		{"deepgram_is_valid", TTSTypeDeepgram, true},
		{"aws_is_valid", TTSTypeAWS, true},
		{"azure_is_valid", TTSTypeAzure, true},
		{"async_is_valid", TTSTypeAsync, true},
		{"fish_is_valid", TTSTypeFish, true},
		{"groq_is_valid", TTSTypeGroq, true},
		{"hume_is_valid", TTSTypeHume, true},
		{"inworld_is_valid", TTSTypeInworld, true},
		{"lmnt_is_valid", TTSTypeLMNT, true},
		{"minimax_is_valid", TTSTypeMiniMax, true},
		{"neuphonic_is_valid", TTSTypeNeuphonic, true},
		{"nvidia_riva_is_valid", TTSTypeNvidiaRiva, true},
		{"piper_is_valid", TTSTypePiper, true},
		{"playht_is_valid", TTSTypePlayHT, true},
		{"rime_is_valid", TTSTypeRime, true},
		{"sarvam_is_valid", TTSTypeSarvam, true},
		{"xtts_is_valid", TTSTypeXTTS, true},
		{"gcp_is_invalid", TTSType("gcp"), false},
		{"random_string_is_invalid", TTSType("random"), false},
		{"polly_is_invalid", TTSType("polly"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ttsType.IsValid() != tt.expected {
				t.Errorf("TTSType(%q).IsValid() = %v, want %v", tt.ttsType, !tt.expected, tt.expected)
			}
		})
	}
}

func TestTTSTypeValidValues(t *testing.T) {
	values := TTSTypeNone.ValidValues()

	if len(values) == 0 {
		t.Fatal("ValidValues() returned empty slice")
	}

	// Should not contain empty string
	for _, v := range values {
		if v == "" {
			t.Error("ValidValues() should not contain empty string")
		}
	}

	// Should be sorted
	for i := 1; i < len(values); i++ {
		if values[i] < values[i-1] {
			t.Errorf("ValidValues() not sorted: %q comes after %q", values[i], values[i-1])
		}
	}

	// Should contain known values
	knownValues := map[string]bool{
		"google": false, "openai": false, "elevenlabs": false, "cartesia": false,
	}
	for _, v := range values {
		if _, ok := knownValues[v]; ok {
			knownValues[v] = true
		}
	}
	for k, found := range knownValues {
		if !found {
			t.Errorf("ValidValues() missing expected value: %q", k)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd bin-ai-manager && go test -v ./models/ai/ -run "TestTTSTypeIsValid|TestTTSTypeValidValues"`
Expected: FAIL — `IsValid` and `ValidValues` methods not defined

**Step 3: Implement TTSType validation methods**

Add to `bin-ai-manager/models/ai/main.go` after the TTSType constants block (after line 187), before the STTType definition:

```go
var validTTSTypes = map[TTSType]bool{
	TTSTypeNone: true, TTSTypeAsync: true, TTSTypeAWS: true,
	TTSTypeAzure: true, TTSTypeCartesia: true, TTSTypeDeepgram: true,
	TTSTypeElevenLabs: true, TTSTypeFish: true, TTSTypeGoogle: true,
	TTSTypeGroq: true, TTSTypeHume: true, TTSTypeInworld: true,
	TTSTypeLMNT: true, TTSTypeMiniMax: true, TTSTypeNeuphonic: true,
	TTSTypeNvidiaRiva: true, TTSTypeOpenAI: true, TTSTypePiper: true,
	TTSTypePlayHT: true, TTSTypeRime: true, TTSTypeSarvam: true,
	TTSTypeXTTS: true,
}

// IsValid returns true if the TTSType is a known valid value.
func (t TTSType) IsValid() bool {
	return validTTSTypes[t]
}

// ValidValues returns a sorted list of valid TTSType values (excluding empty string).
func (t TTSType) ValidValues() []string {
	res := make([]string, 0, len(validTTSTypes))
	for k := range validTTSTypes {
		if k != TTSTypeNone {
			res = append(res, string(k))
		}
	}
	sort.Strings(res)
	return res
}
```

Also add `"sort"` to the imports in `main.go`.

**Step 4: Run tests to verify they pass**

Run: `cd bin-ai-manager && go test -v ./models/ai/ -run "TestTTSTypeIsValid|TestTTSTypeValidValues"`
Expected: PASS

**Step 5: Commit**

```bash
git add bin-ai-manager/models/ai/main.go bin-ai-manager/models/ai/main_test.go
git commit -m "NOJIRA-validate-tts-stt-types

- bin-ai-manager: Add TTSType.IsValid() and TTSType.ValidValues() with map-based validation"
```

---

### Task 2: Add STTType validation methods and tests

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go:189-197` (after STTType constants)
- Modify: `bin-ai-manager/models/ai/main_test.go` (append new tests)

**Step 1: Write the failing tests for STTType.IsValid() and STTType.ValidValues()**

Append to `bin-ai-manager/models/ai/main_test.go`:

```go
func TestSTTTypeIsValid(t *testing.T) {
	tests := []struct {
		name     string
		sttType  STTType
		expected bool
	}{
		{"empty_string_is_valid", STTTypeNone, true},
		{"cartesia_is_valid", STTTypeCartesia, true},
		{"deepgram_is_valid", STTTypeDeepgram, true},
		{"elevenlabs_is_valid", STTTypeElevenLabs, true},
		{"gcp_is_invalid", STTType("gcp"), false},
		{"google_is_invalid", STTType("google"), false},
		{"random_string_is_invalid", STTType("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sttType.IsValid() != tt.expected {
				t.Errorf("STTType(%q).IsValid() = %v, want %v", tt.sttType, !tt.expected, tt.expected)
			}
		})
	}
}

func TestSTTTypeValidValues(t *testing.T) {
	values := STTTypeNone.ValidValues()

	if len(values) == 0 {
		t.Fatal("ValidValues() returned empty slice")
	}

	// Should not contain empty string
	for _, v := range values {
		if v == "" {
			t.Error("ValidValues() should not contain empty string")
		}
	}

	// Should be sorted
	for i := 1; i < len(values); i++ {
		if values[i] < values[i-1] {
			t.Errorf("ValidValues() not sorted: %q comes after %q", values[i], values[i-1])
		}
	}

	// Should contain known values
	knownValues := map[string]bool{
		"deepgram": false, "cartesia": false, "elevenlabs": false,
	}
	for _, v := range values {
		if _, ok := knownValues[v]; ok {
			knownValues[v] = true
		}
	}
	for k, found := range knownValues {
		if !found {
			t.Errorf("ValidValues() missing expected value: %q", k)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd bin-ai-manager && go test -v ./models/ai/ -run "TestSTTTypeIsValid|TestSTTTypeValidValues"`
Expected: FAIL — `IsValid` and `ValidValues` methods not defined on STTType

**Step 3: Implement STTType validation methods**

Add to `bin-ai-manager/models/ai/main.go` after the STTType constants block (after line 197):

```go
var validSTTTypes = map[STTType]bool{
	STTTypeNone: true, STTTypeCartesia: true,
	STTTypeDeepgram: true, STTTypeElevenLabs: true,
}

// IsValid returns true if the STTType is a known valid value.
func (s STTType) IsValid() bool {
	return validSTTTypes[s]
}

// ValidValues returns a sorted list of valid STTType values (excluding empty string).
func (s STTType) ValidValues() []string {
	res := make([]string, 0, len(validSTTTypes))
	for k := range validSTTTypes {
		if k != STTTypeNone {
			res = append(res, string(k))
		}
	}
	sort.Strings(res)
	return res
}
```

**Step 4: Run tests to verify they pass**

Run: `cd bin-ai-manager && go test -v ./models/ai/ -run "TestSTTTypeIsValid|TestSTTTypeValidValues"`
Expected: PASS

**Step 5: Commit**

```bash
git add bin-ai-manager/models/ai/main.go bin-ai-manager/models/ai/main_test.go
git commit -m "NOJIRA-validate-tts-stt-types

- bin-ai-manager: Add STTType.IsValid() and STTType.ValidValues() with map-based validation"
```

---

### Task 3: Add validation in aihandler Create/Update and tests

**Files:**
- Modify: `bin-ai-manager/pkg/aihandler/chatbot.go:1-67`
- Modify: `bin-ai-manager/pkg/aihandler/chatbot_test.go`

**Step 1: Write the failing tests for invalid tts_type and stt_type**

Add new test cases to the existing `TestCreate` test table in `bin-ai-manager/pkg/aihandler/chatbot_test.go`. The existing tests use `ai.TTSTypeNone` and `ai.STTTypeNone` which are valid — add cases with invalid types.

The existing test struct only has `engineModel` as a field, not `ttsType`/`sttType`. You need to add `ttsType ai.TTSType` and `sttType ai.STTType` fields to the test struct, set them to valid defaults in existing test cases, and add new invalid cases.

Add to `TestCreate` test table:
```go
{
    name:        "fails_with_invalid_tts_type",
    customerID:  uuid.Must(uuid.NewV4()),
    aiName:      "Test AI",
    engineModel: ai.EngineModelOpenaiGPT4O,
    ttsType:     ai.TTSType("gcp"),
    sttType:     ai.STTTypeNone,
    setupMock: func(m *dbhandler.MockDBHandler) {
        // Should not call database
    },
    wantError: true,
    errorMsg:  "invalid tts_type",
},
{
    name:        "fails_with_invalid_stt_type",
    customerID:  uuid.Must(uuid.NewV4()),
    aiName:      "Test AI",
    engineModel: ai.EngineModelOpenaiGPT4O,
    ttsType:     ai.TTSTypeNone,
    sttType:     ai.STTType("gcp"),
    setupMock: func(m *dbhandler.MockDBHandler) {
        // Should not call database
    },
    wantError: true,
    errorMsg:  "invalid stt_type",
},
```

Update existing test cases to include valid `ttsType` and `sttType` fields. Update the `h.Create()` call to use `tt.ttsType` and `tt.sttType` instead of `ai.TTSTypeNone` and `ai.STTTypeNone`.

Add same pattern for `TestUpdate` — add `ttsType`/`sttType` fields, update existing cases, add invalid cases.

**Step 2: Run tests to verify they fail**

Run: `cd bin-ai-manager && go test -v ./pkg/aihandler/ -run "TestCreate|TestUpdate"`
Expected: FAIL — invalid tts_type/stt_type cases pass when they should fail (no validation yet)

**Step 3: Add validation to aihandler.Create() and aihandler.Update()**

Modify `bin-ai-manager/pkg/aihandler/chatbot.go`. Add `"strings"` to imports. After the `engineModel` validation (line 28-30) in `Create()`, add:

```go
if !ttsType.IsValid() {
    return nil, fmt.Errorf("invalid tts_type: %s. valid values: %s", ttsType, strings.Join(ttsType.ValidValues(), ", "))
}

if !sttType.IsValid() {
    return nil, fmt.Errorf("invalid stt_type: %s. valid values: %s", sttType, strings.Join(sttType.ValidValues(), ", "))
}
```

Add the same validation in `Update()` after line 56-58.

**Step 4: Run tests to verify they pass**

Run: `cd bin-ai-manager && go test -v ./pkg/aihandler/ -run "TestCreate|TestUpdate"`
Expected: PASS

**Step 5: Commit**

```bash
git add bin-ai-manager/pkg/aihandler/chatbot.go bin-ai-manager/pkg/aihandler/chatbot_test.go
git commit -m "NOJIRA-validate-tts-stt-types

- bin-ai-manager: Add tts_type and stt_type validation in aihandler Create and Update"
```

---

### Task 4: Add validation in ai-control CLI

**Files:**
- Modify: `bin-ai-manager/cmd/ai-control/main.go:165-209` (runCreate) and `bin-ai-manager/cmd/ai-control/main.go:257-301` (runUpdate)

**Step 1: Add tts_type/stt_type validation in runCreate**

In `runCreate()`, after the engine model validation block (lines 185-188), add:

```go
// Validate tts_type
if ttsType != "" && !ttsType.IsValid() {
    return fmt.Errorf("invalid tts_type: %s. valid values: %s", ttsType, strings.Join(ttsType.ValidValues(), ", "))
}

// Validate stt_type
if sttType != "" && !sttType.IsValid() {
    return fmt.Errorf("invalid stt_type: %s. valid values: %s", sttType, strings.Join(sttType.ValidValues(), ", "))
}
```

Note: The CLI uses `!= ""` guard because the default flag value is `""` (meaning "not provided"), and empty is a valid value in the handler. The handler will accept empty — we only reject non-empty invalid values at the CLI level. The handler-level validation catches all cases including programmatic callers.

Add `"strings"` to imports.

**Step 2: Add same validation in runUpdate**

In `runUpdate()`, after the engine model validation block (lines 277-280), add the same validation block.

**Step 3: Run verification**

Run: `cd bin-ai-manager && go build ./cmd/ai-control/`
Expected: BUILD SUCCESS

**Step 4: Commit**

```bash
git add bin-ai-manager/cmd/ai-control/main.go
git commit -m "NOJIRA-validate-tts-stt-types

- bin-ai-manager: Add tts_type and stt_type validation in ai-control CLI"
```

---

### Task 5: Run full verification workflow

**Step 1: Run full verification**

```bash
cd bin-ai-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS

**Step 2: Fix any lint or test issues found**

If any lint issues, fix them and re-run.

**Step 3: Squash into a single commit or leave as-is per user preference**

Ask the user whether to squash commits or keep them as separate commits before pushing.
