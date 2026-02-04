# Address Validation Implementation Plan

## Tasks

### Task 1: Create validate.go
Create `bin-common-handler/models/address/validate.go` with:
- `Validate()` method on `*Address`
- `ValidateTarget()` function
- Internal validators: `validateTel`, `validateEmail`, `validateSIP`, `validateUUID`
- Regex pattern for tel validation

### Task 2: Create validate_test.go
Create `bin-common-handler/models/address/validate_test.go` with:
- Table-driven tests for all address types
- Edge cases: empty, invalid format, valid format
- Test both `Validate()` and `ValidateTarget()`

### Task 3: Run verification workflow
```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

### Task 4: Commit implementation
Commit all changes with appropriate message.
