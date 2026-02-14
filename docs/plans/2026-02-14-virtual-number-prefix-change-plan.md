# Virtual Number Prefix Change (+999 to +899) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Change the virtual number prefix from `+999` to `+899` to avoid confusion with the `999` emergency number in Commonwealth countries.

**Architecture:** Pure constant-swap with refactoring to ensure all production code references constants instead of hardcoded prefix strings. Test files use hardcoded strings. Database migration renames existing numbers.

**Tech Stack:** Go, Alembic (Python), MySQL

---

### Task 1: Update constants and add VirtualNumberCountryCode

**Files:**
- Modify: `bin-number-manager/models/number/validate.go:8-17`

**Step 1: Update the constants**

Change `validate.go` constants block to:

```go
const (
	// VirtualNumberPrefix is the required prefix for virtual numbers
	VirtualNumberPrefix = "+899"

	// VirtualNumberLength is the required length for virtual numbers (+ followed by 12 digits)
	VirtualNumberLength = 13

	// VirtualNumberReservedPrefix is the prefix for reserved virtual numbers (+899000XXXXXX)
	VirtualNumberReservedPrefix = "+899000"

	// VirtualNumberCountryCode is the country code used for virtual numbers in available number responses
	VirtualNumberCountryCode = "899"
)
```

Also update the function comment on line 20:

```go
// ValidateVirtualNumber validates a virtual number string.
// If allowReserved is false, numbers in the reserved range (+899000XXXXXX) are rejected.
```

**Step 2: Run test to verify it compiles but tests fail (expected)**

Run: `cd bin-number-manager && go build ./...`
Expected: Compiles successfully (validation logic uses constants, no hardcoded strings)

**Step 3: Commit**

```
git add bin-number-manager/models/number/validate.go
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- bin-number-manager: Update VirtualNumberPrefix from +999 to +899
- bin-number-manager: Update VirtualNumberReservedPrefix from +999000 to +899000
- bin-number-manager: Add VirtualNumberCountryCode constant"
```

---

### Task 2: Update validate_test.go

**Files:**
- Modify: `bin-number-manager/models/number/validate_test.go`

**Step 1: Update all test case strings**

Replace all `+999` with `+899` in test data. The full updated test table:

```go
	tests := []struct {
		name          string
		num           string
		allowReserved bool
		expectErr     bool
	}{
		// valid cases
		{"valid virtual number", "+899001000001", false, false},
		{"valid virtual number max", "+899999999999", false, false},
		{"valid virtual number mid", "+899500123456", false, false},
		{"valid reserved with allow", "+899000000000", true, false},
		{"valid reserved max with allow", "+899000999999", true, false},

		// invalid prefix
		{"missing +899 prefix", "+15551234567", false, true},
		{"missing + sign", "899001000001", false, true},
		{"wrong prefix", "+998001000001", false, true},

		// invalid length
		{"too short", "+89900100000", false, true},
		{"too long", "+8990010000001", false, true},
		{"empty string", "", false, true},
		{"just prefix", "+899", false, true},

		// invalid characters
		{"contains letter", "+899001a00001", false, true},
		{"contains space", "+899001 00001", false, true},
		{"contains dash", "+899-01000001", false, true},

		// reserved range
		{"reserved range rejected", "+899000000000", false, true},
		{"reserved range max rejected", "+899000999999", false, true},
		{"reserved range mid rejected", "+899000500000", false, true},
	}
```

**Step 2: Run test to verify it passes**

Run: `cd bin-number-manager && go test -v ./models/number/...`
Expected: PASS — all 20 test cases pass

**Step 3: Commit**

```
git add bin-number-manager/models/number/validate_test.go
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- bin-number-manager: Update validate_test.go test cases from +999 to +899"
```

---

### Task 3: Refactor available_number.go to use constants

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/available_number.go:39-74`

**Step 1: Replace hardcoded format string and country code**

On line 39, update the comment:
```go
		// generate random number in range +899001000000 to +899999999999
```

On line 43, replace the hardcoded format string:
```go
		num := fmt.Sprintf("%s%03d%06d", number.VirtualNumberPrefix, areaCode, subscriber)
```

On line 74, replace the hardcoded country:
```go
			Country:      number.VirtualNumberCountryCode,
```

**Step 2: Run test to verify it compiles**

Run: `cd bin-number-manager && go build ./...`
Expected: Compiles successfully

**Step 3: Commit**

```
git add bin-number-manager/pkg/numberhandler/available_number.go
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- bin-number-manager: Refactor available_number.go to use VirtualNumberPrefix and VirtualNumberCountryCode constants"
```

---

### Task 4: Update number_test.go

**Files:**
- Modify: `bin-number-manager/pkg/numberhandler/number_test.go`

**Step 1: Replace all +999 with +899 in test data**

Lines to change (all are test data strings):
- Line 696: `"+999123456789"` → `"+899123456789"`
- Line 706: `"+999123456789"` → `"+899123456789"`
- Line 716: `"+999123456789"` → `"+899123456789"`
- Line 788: `"+99912345"` → `"+89912345"`
- Line 794: `"+999000123456"` → `"+899000123456"`
- Line 800: `"+999123456789"` → `"+899123456789"`
- Line 806: `"+999123456789"` → `"+899123456789"`

**Step 2: Run tests to verify they pass**

Run: `cd bin-number-manager && go test -v ./pkg/numberhandler/...`
Expected: PASS — all numberhandler tests pass

**Step 3: Commit**

```
git add bin-number-manager/pkg/numberhandler/number_test.go
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- bin-number-manager: Update numberhandler test data from +999 to +899"
```

---

### Task 5: Update CLI comment

**Files:**
- Modify: `bin-number-manager/cmd/number-control/main.go:188`

**Step 1: Update the comment**

Change line 188 from:
```go
		// CLI allows reserved range (+999000XXXXXX)
```
to:
```go
		// CLI allows reserved range (VirtualNumberReservedPrefix)
```

**Step 2: Verify it compiles**

Run: `cd bin-number-manager && go build ./cmd/...`
Expected: Compiles successfully

**Step 3: Commit**

```
git add bin-number-manager/cmd/number-control/main.go
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- bin-number-manager: Update CLI comment to reference constant instead of hardcoded prefix"
```

---

### Task 6: Run full verification for bin-number-manager

**Step 1: Run the full verification workflow**

```bash
cd bin-number-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass. `go generate` regenerates mocks (no functional changes). All tests pass. Lint is clean.

**Step 2: Commit any generated file changes**

If `go generate` changed mock files, commit them:

```
git add bin-number-manager/
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- bin-number-manager: Regenerate mocks and vendor after prefix change"
```

---

### Task 7: Update bin-api-manager test

**Files:**
- Modify: `bin-api-manager/server/available_numbers_test.go:72,81`

**Step 1: Update test data**

Change line 72:
```go
					Number:  "+899123456789",
```

Change line 81:
```go
			expectedRes:       `{"result":[{"number":"+899123456789","country":"","region":"","postal_code":"","features":null}]}`,
```

**Step 2: Run the full verification workflow for bin-api-manager**

```bash
cd bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass.

**Step 3: Commit**

```
git add bin-api-manager/
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- bin-api-manager: Update available_numbers_test.go test data from +999 to +899"
```

---

### Task 8: Create Alembic migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/fd9ebdbd7baa_number_update_virtual_prefix_899.py`

**Step 1: Create the migration file**

```python
"""number_update_virtual_prefix_899

Revision ID: fd9ebdbd7baa
Revises: fd9ebdbd7acc
Create Date: 2026-02-14 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'fd9ebdbd7baa'
down_revision = 'fd9ebdbd7acc'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        UPDATE number_numbers
        SET number = REPLACE(number, '+999', '+899')
        WHERE type = 'virtual';
    """)


def downgrade():
    op.execute("""
        UPDATE number_numbers
        SET number = REPLACE(number, '+899', '+999')
        WHERE type = 'virtual';
    """)
```

Note: The `down_revision` should point to the latest existing migration. Check the current head with `alembic heads` and update if needed.

**IMPORTANT:** Do NOT run `alembic upgrade`. Only create the file. Human runs the migration.

**Step 2: Commit**

```
git add bin-dbscheme-manager/bin-manager/main/versions/fd9ebdbd7baa_number_update_virtual_prefix_899.py
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- bin-dbscheme-manager: Add migration to update virtual number prefix from +999 to +899"
```

---

### Task 9: Update original design doc

**Files:**
- Modify: `docs/plans/2026-02-10-virtual-number-design.md`

**Step 1: Replace all +999 references with +899**

Key changes:
- Line 9: `+999` → `+899`
- Line 14: `+999` → `+899`
- Line 18: `+999 XXX YYYYYY` → `+899 XXX YYYYYY`, `country 999` → `country 899`
- Line 22: `+999000000000` through `+999000999999` → `+899000000000` through `+899000999999`
- Line 24-25: `+999000XXXXXX` → `+899000XXXXXX`
- Line 29: `+999` → `+899`
- All other occurrences

**Step 2: Commit**

```
git add docs/plans/2026-02-10-virtual-number-design.md
git commit -m "NOJIRA-Change-virtual-number-prefix-899

- docs: Update virtual number design doc to reflect +899 prefix"
```

---

### Task 10: Final verification and squash

**Step 1: Run full verification for both services one final time**

```bash
cd bin-number-manager && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: Both pass cleanly.

**Step 2: Fetch latest main and check for conflicts**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

Expected: No conflicts.

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Change-virtual-number-prefix-899
```

Create PR with title `NOJIRA-Change-virtual-number-prefix-899` and description summarizing all changes.
