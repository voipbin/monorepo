# Database Mapping Improvement Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace manual ORM work with struct tags and reflection helpers to eliminate maintenance burden, scattered type conversions, and runtime errors.

**Architecture:** Create `pkg/dbutil` package with three reflection-based helpers (GetDBFields, PrepareValues, ScanRow) that read `db` struct tags. Add tags to models, then migrate dbhandler methods incrementally per model type.

**Tech Stack:** Go stdlib reflection, sql.Null* types, existing squirrel query builder

---

## Task 1: Create dbutil Package Structure

**Files:**
- Create: `pkg/dbutil/main.go`
- Create: `pkg/dbutil/main_test.go`

**Step 1: Write the test file structure**

Create `pkg/dbutil/main_test.go`:

```go
package dbutil

import (
	"testing"

	"github.com/gofrs/uuid"
)

// Test model for validating dbutil functions
type testModel struct {
	ID         uuid.UUID `db:"id,uuid"`
	Name       string    `db:"name"`
	Count      int       `db:"count"`
	SkipMe     bool      `db:"-"`
}

func TestGetDBFields_Basic(t *testing.T) {
	t.Skip("Not implemented yet")
}

func TestPrepareValues_Basic(t *testing.T) {
	t.Skip("Not implemented yet")
}

func TestScanRow_Basic(t *testing.T) {
	t.Skip("Not implemented yet")
}
```

**Step 2: Create package file**

Create `pkg/dbutil/main.go`:

```go
package dbutil

import (
	"database/sql"
)

// GetDBFields returns ordered column names from struct tags
func GetDBFields(model interface{}) []string {
	panic("not implemented")
}

// PrepareValues converts struct fields to database values for INSERT/UPDATE
func PrepareValues(model interface{}) ([]interface{}, error) {
	panic("not implemented")
}

// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
func ScanRow(row *sql.Rows, dest interface{}) error {
	panic("not implemented")
}
```

**Step 3: Verify package compiles**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go build ./pkg/dbutil/...`
Expected: SUCCESS (package compiles)

**Step 4: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go
git commit -m "feat: create dbutil package structure with placeholder functions"
```

---

## Task 2: Implement GetDBFields Helper

**Files:**
- Modify: `pkg/dbutil/main.go`
- Modify: `pkg/dbutil/main_test.go`

**Step 1: Write comprehensive tests**

Replace `TestGetDBFields_Basic` in `pkg/dbutil/main_test.go`:

```go
func TestGetDBFields_Basic(t *testing.T) {
	tests := []struct {
		name     string
		model    interface{}
		expected []string
	}{
		{
			name: "basic model with UUID and string fields",
			model: &testModel{},
			expected: []string{"id", "name", "count"},
		},
		{
			name: "skips fields with db:\"-\" tag",
			model: &struct {
				Field1 string `db:"field1"`
				Field2 string `db:"-"`
				Field3 string `db:"field3"`
			}{},
			expected: []string{"field1", "field3"},
		},
		{
			name: "handles fields without conversion type",
			model: &struct {
				ID   string `db:"id"`
				Name string `db:"name"`
			}{},
			expected: []string{"id", "name"},
		},
		{
			name: "handles fields with conversion types",
			model: &struct {
				ID   uuid.UUID `db:"id,uuid"`
				Data []string  `db:"data,json"`
			}{},
			expected: []string{"id", "data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDBFields(tt.model)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d", len(tt.expected), len(result))
				return
			}

			for i, field := range result {
				if field != tt.expected[i] {
					t.Errorf("field[%d]: expected %s, got %s", i, tt.expected[i], field)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestGetDBFields`
Expected: FAIL with "not implemented" panic

**Step 3: Implement GetDBFields**

Replace `GetDBFields` function in `pkg/dbutil/main.go`:

```go
import (
	"database/sql"
	"reflect"
	"strings"
)

// GetDBFields returns ordered column names from struct tags
func GetDBFields(model interface{}) []string {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	fields := []string{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: "column_name" or "column_name,conversion_type"
		parts := strings.Split(tag, ",")
		columnName := parts[0]

		fields = append(fields, columnName)
	}

	return fields
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestGetDBFields`
Expected: PASS (all test cases pass)

**Step 5: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go
git commit -m "feat: implement GetDBFields helper with reflection"
```

---

## Task 3: Implement PrepareValues Helper (Part 1: Basic Types)

**Files:**
- Modify: `pkg/dbutil/main.go`
- Modify: `pkg/dbutil/main_test.go`

**Step 1: Write tests for basic types**

Replace `TestPrepareValues_Basic` in `pkg/dbutil/main_test.go`:

```go
func TestPrepareValues_Basic(t *testing.T) {
	tests := []struct {
		name     string
		model    interface{}
		expected []interface{}
	}{
		{
			name: "basic string and int fields",
			model: &struct {
				Name  string `db:"name"`
				Count int    `db:"count"`
			}{
				Name:  "test",
				Count: 42,
			},
			expected: []interface{}{"test", 42},
		},
		{
			name: "skips fields with db:\"-\" tag",
			model: &struct {
				Field1 string `db:"field1"`
				Field2 string `db:"-"`
				Field3 string `db:"field3"`
			}{
				Field1: "value1",
				Field2: "skip",
				Field3: "value3",
			},
			expected: []interface{}{"value1", "value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareValues(tt.model)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d values, got %d", len(tt.expected), len(result))
				return
			}

			for i, val := range result {
				if val != tt.expected[i] {
					t.Errorf("value[%d]: expected %v, got %v", i, tt.expected[i], val)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestPrepareValues`
Expected: FAIL with "not implemented" panic

**Step 3: Implement PrepareValues for basic types**

Replace `PrepareValues` function in `pkg/dbutil/main.go`:

```go
// PrepareValues converts struct fields to database values for INSERT/UPDATE
func PrepareValues(model interface{}) ([]interface{}, error) {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	values := []interface{}{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		fieldVal := val.Field(i)
		values = append(values, fieldVal.Interface())
	}

	return values, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestPrepareValues`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go
git commit -m "feat: implement PrepareValues helper for basic types"
```

---

## Task 4: Implement PrepareValues Helper (Part 2: UUID Conversion)

**Files:**
- Modify: `pkg/dbutil/main.go`
- Modify: `pkg/dbutil/main_test.go`

**Step 1: Write tests for UUID conversion**

Add to `pkg/dbutil/main_test.go`:

```go
func TestPrepareValues_UUID(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name     string
		model    interface{}
		validate func([]interface{}) error
	}{
		{
			name: "converts UUID to bytes",
			model: &struct {
				ID uuid.UUID `db:"id,uuid"`
			}{
				ID: testID,
			},
			validate: func(values []interface{}) error {
				if len(values) != 1 {
					return fmt.Errorf("expected 1 value, got %d", len(values))
				}
				bytes, ok := values[0].([]byte)
				if !ok {
					return fmt.Errorf("expected []byte, got %T", values[0])
				}
				if len(bytes) != 16 {
					return fmt.Errorf("expected 16 bytes, got %d", len(bytes))
				}
				return nil
			},
		},
		{
			name: "converts uuid.Nil to nil UUID bytes",
			model: &struct {
				ID uuid.UUID `db:"id,uuid"`
			}{
				ID: uuid.Nil,
			},
			validate: func(values []interface{}) error {
				if len(values) != 1 {
					return fmt.Errorf("expected 1 value, got %d", len(values))
				}
				bytes, ok := values[0].([]byte)
				if !ok {
					return fmt.Errorf("expected []byte, got %T", values[0])
				}
				// uuid.Nil.Bytes() still returns 16 bytes of zeros
				if len(bytes) != 16 {
					return fmt.Errorf("expected 16 bytes, got %d", len(bytes))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareValues(tt.model)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := tt.validate(result); err != nil {
				t.Errorf("validation failed: %v", err)
			}
		})
	}
}
```

Add import at top of file:

```go
import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
)
```

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestPrepareValues_UUID`
Expected: FAIL (UUID not converted to bytes)

**Step 3: Update PrepareValues to handle UUID conversion**

Update `PrepareValues` function in `pkg/dbutil/main.go`:

```go
import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofrs/uuid"
)

// PrepareValues converts struct fields to database values for INSERT/UPDATE
func PrepareValues(model interface{}) ([]interface{}, error) {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	values := []interface{}{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: "column_name" or "column_name,conversion_type"
		parts := strings.Split(tag, ",")
		conversionType := ""
		if len(parts) > 1 {
			conversionType = parts[1]
		}

		fieldVal := val.Field(i)

		// Apply conversions based on type
		switch conversionType {
		case "uuid":
			// Convert uuid.UUID to []byte
			if fieldVal.Type() == reflect.TypeOf(uuid.UUID{}) {
				uuidVal := fieldVal.Interface().(uuid.UUID)
				values = append(values, uuidVal.Bytes())
			} else {
				return nil, fmt.Errorf("field %s: expected uuid.UUID type for uuid conversion", field.Name)
			}
		default:
			values = append(values, fieldVal.Interface())
		}
	}

	return values, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestPrepareValues`
Expected: PASS (all PrepareValues tests pass)

**Step 5: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go
git commit -m "feat: add UUID conversion support to PrepareValues"
```

---

## Task 5: Implement PrepareValues Helper (Part 3: JSON Conversion)

**Files:**
- Modify: `pkg/dbutil/main.go`
- Modify: `pkg/dbutil/main_test.go`

**Step 1: Write tests for JSON conversion**

Add to `pkg/dbutil/main_test.go`:

```go
func TestPrepareValues_JSON(t *testing.T) {
	type nestedStruct struct {
		Key   string `json:"key"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name     string
		model    interface{}
		expected string
	}{
		{
			name: "converts slice to JSON",
			model: &struct {
				Items []string `db:"items,json"`
			}{
				Items: []string{"a", "b", "c"},
			},
			expected: `["a","b","c"]`,
		},
		{
			name: "converts empty slice to JSON array",
			model: &struct {
				Items []string `db:"items,json"`
			}{
				Items: []string{},
			},
			expected: `[]`,
		},
		{
			name: "converts struct to JSON",
			model: &struct {
				Data nestedStruct `db:"data,json"`
			}{
				Data: nestedStruct{Key: "test", Value: 42},
			},
			expected: `{"key":"test","value":42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareValues(tt.model)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("expected 1 value, got %d", len(result))
			}

			jsonStr, ok := result[0].(string)
			if !ok {
				t.Fatalf("expected string, got %T", result[0])
			}

			if jsonStr != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, jsonStr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestPrepareValues_JSON`
Expected: FAIL (JSON not converted)

**Step 3: Update PrepareValues to handle JSON conversion**

Update the switch statement in `PrepareValues` function in `pkg/dbutil/main.go`:

```go
import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofrs/uuid"
)

// PrepareValues converts struct fields to database values for INSERT/UPDATE
func PrepareValues(model interface{}) ([]interface{}, error) {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	values := []interface{}{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: "column_name" or "column_name,conversion_type"
		parts := strings.Split(tag, ",")
		conversionType := ""
		if len(parts) > 1 {
			conversionType = parts[1]
		}

		fieldVal := val.Field(i)

		// Apply conversions based on type
		switch conversionType {
		case "uuid":
			// Convert uuid.UUID to []byte
			if fieldVal.Type() == reflect.TypeOf(uuid.UUID{}) {
				uuidVal := fieldVal.Interface().(uuid.UUID)
				values = append(values, uuidVal.Bytes())
			} else {
				return nil, fmt.Errorf("field %s: expected uuid.UUID type for uuid conversion", field.Name)
			}
		case "json":
			// Convert to JSON string
			jsonBytes, err := json.Marshal(fieldVal.Interface())
			if err != nil {
				return nil, fmt.Errorf("field %s: cannot marshal to JSON: %w", field.Name, err)
			}
			values = append(values, string(jsonBytes))
		default:
			values = append(values, fieldVal.Interface())
		}
	}

	return values, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestPrepareValues`
Expected: PASS (all PrepareValues tests pass)

**Step 5: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go
git commit -m "feat: add JSON conversion support to PrepareValues"
```

---

## Task 6: Implement ScanRow Helper (Part 1: Basic Types)

**Files:**
- Modify: `pkg/dbutil/main.go`
- Modify: `pkg/dbutil/main_test.go`
- Create: `pkg/dbutil/scan_test_helper.go` (test utilities)

**Step 1: Create test helper for mock rows**

Create `pkg/dbutil/scan_test_helper.go`:

```go
package dbutil

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// createMockRows creates a mock sql.Rows for testing
func createMockRows(t *testing.T, columns []string, values [][]interface{}) *sql.Rows {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	rows := mock.NewRows(columns)
	for _, row := range values {
		rows.AddRow(row...)
	}

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("failed to create rows: %v", err)
	}

	return result
}
```

Add go-sqlmock dependency:

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go get github.com/DATA-DOG/go-sqlmock`

**Step 2: Write tests for basic type scanning**

Replace `TestScanRow_Basic` in `pkg/dbutil/main_test.go`:

```go
func TestScanRow_Basic(t *testing.T) {
	tests := []struct {
		name     string
		columns  []string
		values   []interface{}
		dest     interface{}
		validate func(interface{}) error
	}{
		{
			name:    "scans string and int fields",
			columns: []string{"name", "count"},
			values:  []interface{}{"test", 42},
			dest: &struct {
				Name  string `db:"name"`
				Count int    `db:"count"`
			}{},
			validate: func(dest interface{}) error {
				v := dest.(*struct {
					Name  string `db:"name"`
					Count int    `db:"count"`
				})
				if v.Name != "test" {
					return fmt.Errorf("expected name='test', got '%s'", v.Name)
				}
				if v.Count != 42 {
					return fmt.Errorf("expected count=42, got %d", v.Count)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := createMockRows(t, tt.columns, [][]interface{}{tt.values})
			defer rows.Close()

			if !rows.Next() {
				t.Fatal("expected row")
			}

			err := ScanRow(rows, tt.dest)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := tt.validate(tt.dest); err != nil {
				t.Errorf("validation failed: %v", err)
			}
		})
	}
}
```

**Step 3: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestScanRow`
Expected: FAIL with "not implemented" panic

**Step 4: Implement ScanRow for basic types**

Replace `ScanRow` function in `pkg/dbutil/main.go`:

```go
// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
func ScanRow(row *sql.Rows, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	destVal = destVal.Elem()
	destType := destVal.Type()

	// Build list of scan targets in order of db tags
	scanTargets := []interface{}{}

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		fieldVal := destVal.Field(i)
		scanTargets = append(scanTargets, fieldVal.Addr().Interface())
	}

	// Scan the row
	if err := row.Scan(scanTargets...); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	return nil
}
```

**Step 5: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestScanRow_Basic`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go pkg/dbutil/scan_test_helper.go
git commit -m "feat: implement ScanRow helper for basic types"
```

---

## Task 7: Implement ScanRow Helper (Part 2: NULL Handling)

**Files:**
- Modify: `pkg/dbutil/main.go`
- Modify: `pkg/dbutil/main_test.go`

**Step 1: Write tests for NULL handling**

Add to `pkg/dbutil/main_test.go`:

```go
func TestScanRow_NullHandling(t *testing.T) {
	tests := []struct {
		name     string
		columns  []string
		values   []interface{}
		dest     interface{}
		validate func(interface{}) error
	}{
		{
			name:    "converts NULL string to empty string",
			columns: []string{"name"},
			values:  []interface{}{nil},
			dest: &struct {
				Name string `db:"name"`
			}{},
			validate: func(dest interface{}) error {
				v := dest.(*struct {
					Name string `db:"name"`
				})
				if v.Name != "" {
					return fmt.Errorf("expected empty string, got '%s'", v.Name)
				}
				return nil
			},
		},
		{
			name:    "converts NULL int to zero",
			columns: []string{"count"},
			values:  []interface{}{nil},
			dest: &struct {
				Count int `db:"count"`
			}{},
			validate: func(dest interface{}) error {
				v := dest.(*struct {
					Count int `db:"count"`
				})
				if v.Count != 0 {
					return fmt.Errorf("expected 0, got %d", v.Count)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := createMockRows(t, tt.columns, [][]interface{}{tt.values})
			defer rows.Close()

			if !rows.Next() {
				t.Fatal("expected row")
			}

			err := ScanRow(rows, tt.dest)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := tt.validate(tt.dest); err != nil {
				t.Errorf("validation failed: %v", err)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestScanRow_NullHandling`
Expected: FAIL (NULL values cause errors)

**Step 3: Update ScanRow to handle NULL values**

Replace `ScanRow` function in `pkg/dbutil/main.go`:

```go
// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
// Handles NULL values by converting to zero values
func ScanRow(row *sql.Rows, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	destVal = destVal.Elem()
	destType := destVal.Type()

	// Build list of scan targets with NULL handling
	scanTargets := []interface{}{}
	fieldIndices := []int{}

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		fieldIndices = append(fieldIndices, i)

		// Use sql.Null* types for scanning to handle NULL
		fieldVal := destVal.Field(i)
		switch fieldVal.Kind() {
		case reflect.String:
			scanTargets = append(scanTargets, new(sql.NullString))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			scanTargets = append(scanTargets, new(sql.NullInt64))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			scanTargets = append(scanTargets, new(sql.NullInt64))
		case reflect.Float32, reflect.Float64:
			scanTargets = append(scanTargets, new(sql.NullFloat64))
		case reflect.Bool:
			scanTargets = append(scanTargets, new(sql.NullBool))
		default:
			// For complex types, scan directly (will handle in next task)
			scanTargets = append(scanTargets, fieldVal.Addr().Interface())
		}
	}

	// Scan the row
	if err := row.Scan(scanTargets...); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Copy values from Null* types to actual fields
	for i, fieldIdx := range fieldIndices {
		fieldVal := destVal.Field(fieldIdx)

		switch v := scanTargets[i].(type) {
		case *sql.NullString:
			if v.Valid {
				fieldVal.SetString(v.String)
			}
			// else: leave as zero value (empty string)
		case *sql.NullInt64:
			if v.Valid {
				fieldVal.SetInt(v.Int64)
			}
			// else: leave as zero value (0)
		case *sql.NullFloat64:
			if v.Valid {
				fieldVal.SetFloat(v.Float64)
			}
			// else: leave as zero value (0.0)
		case *sql.NullBool:
			if v.Valid {
				fieldVal.SetBool(v.Bool)
			}
			// else: leave as zero value (false)
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestScanRow`
Expected: PASS (all ScanRow tests pass)

**Step 5: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go
git commit -m "feat: add NULL handling to ScanRow helper"
```

---

## Task 8: Implement ScanRow Helper (Part 3: UUID Conversion)

**Files:**
- Modify: `pkg/dbutil/main.go`
- Modify: `pkg/dbutil/main_test.go`

**Step 1: Write tests for UUID scanning**

Add to `pkg/dbutil/main_test.go`:

```go
func TestScanRow_UUID(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name     string
		columns  []string
		values   []interface{}
		dest     interface{}
		validate func(interface{}) error
	}{
		{
			name:    "scans UUID from bytes",
			columns: []string{"id"},
			values:  []interface{}{testID.Bytes()},
			dest: &struct {
				ID uuid.UUID `db:"id,uuid"`
			}{},
			validate: func(dest interface{}) error {
				v := dest.(*struct {
					ID uuid.UUID `db:"id,uuid"`
				})
				if v.ID != testID {
					return fmt.Errorf("expected %s, got %s", testID, v.ID)
				}
				return nil
			},
		},
		{
			name:    "scans NULL to uuid.Nil",
			columns: []string{"id"},
			values:  []interface{}{nil},
			dest: &struct {
				ID uuid.UUID `db:"id,uuid"`
			}{},
			validate: func(dest interface{}) error {
				v := dest.(*struct {
					ID uuid.UUID `db:"id,uuid"`
				})
				if v.ID != uuid.Nil {
					return fmt.Errorf("expected uuid.Nil, got %s", v.ID)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := createMockRows(t, tt.columns, [][]interface{}{tt.values})
			defer rows.Close()

			if !rows.Next() {
				t.Fatal("expected row")
			}

			err := ScanRow(rows, tt.dest)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := tt.validate(tt.dest); err != nil {
				t.Errorf("validation failed: %v", err)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestScanRow_UUID`
Expected: FAIL (UUID not converted from bytes)

**Step 3: Update ScanRow to handle UUID conversion**

Update `ScanRow` function in `pkg/dbutil/main.go` to parse conversion types and handle UUID:

```go
// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
// Handles NULL values by converting to zero values
func ScanRow(row *sql.Rows, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	destVal = destVal.Elem()
	destType := destVal.Type()

	// Build list of scan targets with NULL handling
	type fieldInfo struct {
		index          int
		conversionType string
	}

	scanTargets := []interface{}{}
	fields := []fieldInfo{}

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: "column_name" or "column_name,conversion_type"
		parts := strings.Split(tag, ",")
		conversionType := ""
		if len(parts) > 1 {
			conversionType = parts[1]
		}

		fields = append(fields, fieldInfo{
			index:          i,
			conversionType: conversionType,
		})

		fieldVal := destVal.Field(i)

		// Determine scan target based on conversion type
		switch conversionType {
		case "uuid":
			// UUID stored as bytes, can be NULL
			scanTargets = append(scanTargets, new(sql.NullString))
		default:
			// Use sql.Null* types for scanning to handle NULL
			switch fieldVal.Kind() {
			case reflect.String:
				scanTargets = append(scanTargets, new(sql.NullString))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				scanTargets = append(scanTargets, new(sql.NullInt64))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				scanTargets = append(scanTargets, new(sql.NullInt64))
			case reflect.Float32, reflect.Float64:
				scanTargets = append(scanTargets, new(sql.NullFloat64))
			case reflect.Bool:
				scanTargets = append(scanTargets, new(sql.NullBool))
			default:
				// For complex types without conversion tag, scan directly
				scanTargets = append(scanTargets, fieldVal.Addr().Interface())
			}
		}
	}

	// Scan the row
	if err := row.Scan(scanTargets...); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Copy values from scan targets to actual fields with conversions
	for i, fieldInfo := range fields {
		fieldVal := destVal.Field(fieldInfo.index)

		switch fieldInfo.conversionType {
		case "uuid":
			nullStr := scanTargets[i].(*sql.NullString)
			if nullStr.Valid && len(nullStr.String) > 0 {
				uuidVal, err := uuid.FromBytes([]byte(nullStr.String))
				if err != nil {
					return fmt.Errorf("field %d: cannot convert bytes to UUID: %w", fieldInfo.index, err)
				}
				fieldVal.Set(reflect.ValueOf(uuidVal))
			} else {
				// NULL or empty -> uuid.Nil
				fieldVal.Set(reflect.ValueOf(uuid.Nil))
			}
		default:
			// Handle standard types
			switch v := scanTargets[i].(type) {
			case *sql.NullString:
				if v.Valid {
					fieldVal.SetString(v.String)
				}
			case *sql.NullInt64:
				if v.Valid {
					fieldVal.SetInt(v.Int64)
				}
			case *sql.NullFloat64:
				if v.Valid {
					fieldVal.SetFloat(v.Float64)
				}
			case *sql.NullBool:
				if v.Valid {
					fieldVal.SetBool(v.Bool)
				}
			}
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestScanRow`
Expected: PASS (all ScanRow tests pass including UUID)

**Step 5: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go
git commit -m "feat: add UUID conversion to ScanRow helper"
```

---

## Task 9: Implement ScanRow Helper (Part 4: JSON Conversion)

**Files:**
- Modify: `pkg/dbutil/main.go`
- Modify: `pkg/dbutil/main_test.go`

**Step 1: Write tests for JSON scanning**

Add to `pkg/dbutil/main_test.go`:

```go
func TestScanRow_JSON(t *testing.T) {
	tests := []struct {
		name     string
		columns  []string
		values   []interface{}
		dest     interface{}
		validate func(interface{}) error
	}{
		{
			name:    "scans JSON array to slice",
			columns: []string{"items"},
			values:  []interface{}{`["a","b","c"]`},
			dest: &struct {
				Items []string `db:"items,json"`
			}{},
			validate: func(dest interface{}) error {
				v := dest.(*struct {
					Items []string `db:"items,json"`
				})
				if len(v.Items) != 3 {
					return fmt.Errorf("expected 3 items, got %d", len(v.Items))
				}
				if v.Items[0] != "a" || v.Items[1] != "b" || v.Items[2] != "c" {
					return fmt.Errorf("unexpected items: %v", v.Items)
				}
				return nil
			},
		},
		{
			name:    "scans NULL JSON to empty slice",
			columns: []string{"items"},
			values:  []interface{}{nil},
			dest: &struct {
				Items []string `db:"items,json"`
			}{},
			validate: func(dest interface{}) error {
				v := dest.(*struct {
					Items []string `db:"items,json"`
				})
				if len(v.Items) != 0 {
					return fmt.Errorf("expected empty slice, got %d items", len(v.Items))
				}
				return nil
			},
		},
		{
			name:    "scans empty JSON array to empty slice",
			columns: []string{"items"},
			values:  []interface{}{`[]`},
			dest: &struct {
				Items []string `db:"items,json"`
			}{},
			validate: func(dest interface{}) error {
				v := dest.(*struct {
					Items []string `db:"items,json"`
				})
				if len(v.Items) != 0 {
					return fmt.Errorf("expected empty slice, got %d items", len(v.Items))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := createMockRows(t, tt.columns, [][]interface{}{tt.values})
			defer rows.Close()

			if !rows.Next() {
				t.Fatal("expected row")
			}

			err := ScanRow(rows, tt.dest)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := tt.validate(tt.dest); err != nil {
				t.Errorf("validation failed: %v", err)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestScanRow_JSON`
Expected: FAIL (JSON not converted)

**Step 3: Update ScanRow to handle JSON conversion**

Update the conversion type handling in `ScanRow` function in `pkg/dbutil/main.go`:

```go
// Update the switch statement after scanning to add json case:

		switch fieldInfo.conversionType {
		case "uuid":
			nullStr := scanTargets[i].(*sql.NullString)
			if nullStr.Valid && len(nullStr.String) > 0 {
				uuidVal, err := uuid.FromBytes([]byte(nullStr.String))
				if err != nil {
					return fmt.Errorf("field %d: cannot convert bytes to UUID: %w", fieldInfo.index, err)
				}
				fieldVal.Set(reflect.ValueOf(uuidVal))
			} else {
				// NULL or empty -> uuid.Nil
				fieldVal.Set(reflect.ValueOf(uuid.Nil))
			}
		case "json":
			nullStr := scanTargets[i].(*sql.NullString)
			if nullStr.Valid && len(nullStr.String) > 0 {
				// Unmarshal JSON into field
				if err := json.Unmarshal([]byte(nullStr.String), fieldVal.Addr().Interface()); err != nil {
					return fmt.Errorf("field %d: cannot unmarshal JSON: %w", fieldInfo.index, err)
				}
			}
			// else: NULL or empty -> leave as zero value (empty slice/struct)
		default:
			// ... existing code for standard types ...
```

Also update the scan target selection to handle json:

```go
		// Determine scan target based on conversion type
		switch conversionType {
		case "uuid":
			// UUID stored as bytes, can be NULL
			scanTargets = append(scanTargets, new(sql.NullString))
		case "json":
			// JSON stored as string, can be NULL
			scanTargets = append(scanTargets, new(sql.NullString))
		default:
			// ... existing code ...
```

**Step 4: Run test to verify it passes**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbutil/... -v -run TestScanRow`
Expected: PASS (all ScanRow tests pass)

**Step 5: Commit**

```bash
git add pkg/dbutil/main.go pkg/dbutil/main_test.go
git commit -m "feat: add JSON conversion to ScanRow helper"
```

---

## Task 10: Add DB Tags to Flow Model

**Files:**
- Modify: `models/flow/flow.go`
- Modify: `models/identity/identity.go` (in bin-common-handler)

**Step 1: Add db tags to Identity struct**

Note: This modifies shared code in bin-common-handler. Coordinate with other services.

Edit `/home/pchero/gitvoipbin/monorepo/bin-common-handler/models/identity/identity.go`:

```go
package identity

import "github.com/gofrs/uuid"

// Identity represents
type Identity struct {
	// identity
	ID         uuid.UUID `json:"id" db:"id,uuid"`                    // resource identifier
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"` // resource's customer id
}
```

**Step 2: Add db tags to Flow struct**

Edit `models/flow/flow.go`:

```go
// Flow struct
type Flow struct {
	commonidentity.Identity

	Type Type `json:"type,omitempty" db:"type"`

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	Persist bool `json:"persist,omitempty" db:"-"` // Not stored in database

	Actions []action.Action `json:"actions,omitempty" db:"actions,json"`

	OnCompleteFlowID uuid.UUID `json:"on_complete_flow_id,omitempty" db:"on_complete_flow_id,uuid"`

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}
```

**Step 3: Verify models compile**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go build ./models/flow/...`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add models/flow/flow.go
git add ../bin-common-handler/models/identity/identity.go
git commit -m "feat: add db tags to Flow model and Identity struct"
```

---

## Task 11: Update FlowCreate to Use dbutil

**Files:**
- Modify: `pkg/dbhandler/flows.go`

**Step 1: Import dbutil package**

Add import to `pkg/dbhandler/flows.go`:

```go
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/dbutil"
)
```

**Step 2: Update FlowCreate to use dbutil.PrepareValues**

Replace `FlowCreate` method in `pkg/dbhandler/flows.go`:

```go
func (h *handler) FlowCreate(ctx context.Context, f *flow.Flow) error {
	now := h.util.TimeGetCurTime()

	// Set timestamps
	f.TMCreate = now
	f.TMUpdate = commondatabasehandler.DefaultTimeStamp
	f.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use dbutil to get fields and values
	fields := dbutil.GetDBFields(f)
	values, err := dbutil.PrepareValues(f)
	if err != nil {
		return fmt.Errorf("could not prepare values. FlowCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(flowsTable).
		Columns(fields...).
		Values(values...).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. FlowCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. FlowCreate. err: %v", err)
	}

	_ = h.flowUpdateToCache(ctx, f.ID)
	return nil
}
```

**Step 3: Run existing tests to verify nothing broke**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbhandler/... -v -run TestFlowCreate`
Expected: PASS (or at least no new failures)

**Step 4: Commit**

```bash
git add pkg/dbhandler/flows.go
git commit -m "refactor: update FlowCreate to use dbutil.PrepareValues"
```

---

## Task 12: Update flowGetFromRow to Use dbutil

**Files:**
- Modify: `pkg/dbhandler/flows.go`

**Step 1: Replace flowGetFromRow method**

Replace `flowGetFromRow` method in `pkg/dbhandler/flows.go`:

```go
// flowGetFromRow gets the flow from the row.
func (h *handler) flowGetFromRow(row *sql.Rows) (*flow.Flow, error) {
	res := &flow.Flow{}

	if err := dbutil.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. flowGetFromRow. err: %v", err)
	}

	res.Persist = true
	return res, nil
}
```

**Step 2: Remove manual flowsFields variable**

Delete the `flowsFields` variable at the top of `pkg/dbhandler/flows.go`:

```go
// DELETE THESE LINES:
var (
	flowsTable  = "flow_flows"
	flowsFields = []string{
		string(flow.FieldID),
		string(flow.FieldCustomerID),
		// ... all the field strings
	}
)

// KEEP ONLY:
var (
	flowsTable = "flow_flows"
)
```

**Step 3: Update FlowGets to use dbutil.GetDBFields**

Update `FlowGets` method in `pkg/dbhandler/flows.go`:

```go
func (h *handler) FlowGets(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	// Get fields from model instead of hardcoded list
	fields := dbutil.GetDBFields(&flow.Flow{})

	sb := squirrel.
		Select(fields...).
		From(flowsTable).
		Where(squirrel.Lt{string(flow.FieldTMCreate): token}).
		OrderBy(string(flow.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. FlowGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. FlowGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. FlowGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*flow.Flow{}
	for rows.Next() {
		u, err := h.flowGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. FlowGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. FlowGets. err: %v", err)
	}

	return res, nil
}
```

**Step 4: Update flowGetFromDB to use dbutil.GetDBFields**

Update `flowGetFromDB` method in `pkg/dbhandler/flows.go`:

```go
func (h *handler) flowGetFromDB(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	// Get fields from model instead of hardcoded list
	fields := dbutil.GetDBFields(&flow.Flow{})

	query, args, err := squirrel.
		Select(fields...).
		From(flowsTable).
		Where(squirrel.Eq{string(flow.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. flowGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. flowGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. flowGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.flowGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. flowGetFromDB. id: %s", id)
	}

	return res, nil
}
```

**Step 5: Run all flow tests**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbhandler/... -v -run ".*Flow.*"`
Expected: PASS (all flow-related tests pass)

**Step 6: Commit**

```bash
git add pkg/dbhandler/flows.go
git commit -m "refactor: update flow read operations to use dbutil helpers"
```

---

## Task 13: Run Full Test Suite for Flows

**Files:**
- None (verification step)

**Step 1: Run complete flow-manager test suite**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./... -v`
Expected: PASS (all tests pass, no regressions)

**Step 2: If tests fail, fix issues**

Common issues:
- Field order mismatch: GetDBFields returns fields in struct order
- Missing db tags: Add tags to any fields that should be in DB
- Type conversion errors: Check UUID/JSON conversions

**Step 3: Run linter**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && golangci-lint run -v --timeout 5m`
Expected: PASS (no linting errors)

**Step 4: Manual verification (optional but recommended)**

Start flow-manager locally and verify:
- Create a flow via API
- Retrieve flow via API
- Update flow via API
- List flows via API

**Step 5: Document completion of flows migration**

No commit needed - this is a verification checkpoint.

---

## Task 14: Add DB Tags to Activeflow Model

**Files:**
- Modify: `models/activeflow/activeflow.go`
- Modify: `models/activeflow/field.go` (check field names)

**Step 1: Review activeflow database fields**

Check `pkg/dbhandler/activeflow.go` for `activeflowFields` to understand database column names.

Run: `grep -A 30 "activeflowFields" /home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/activeflow.go`

**Step 2: Add db tags to Activeflow struct**

Edit `models/activeflow/activeflow.go`:

```go
// Activeflow struct
type Activeflow struct {
	commonidentity.Identity

	FlowID uuid.UUID `json:"flow_id,omitempty" db:"flow_id,uuid"`
	Status Status    `json:"status,omitempty" db:"status"`

	ReferenceType         ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID           uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`
	ReferenceActiveflowID uuid.UUID     `json:"reference_activeflow_id,omitempty" db:"reference_activeflow_id,uuid"`

	OnCompleteFlowID uuid.UUID `json:"on_complete_flow_id,omitempty" db:"on_complete_flow_id,uuid"`

	// stack
	StackMap map[uuid.UUID]*stack.Stack `json:"stack_map,omitempty" db:"stack_map,json"`

	// current info
	CurrentStackID uuid.UUID     `json:"current_stack_id,omitempty" db:"current_stack_id,uuid"`
	CurrentAction  action.Action `json:"current_action,omitempty" db:"current_action,json"`

	// forward info
	ForwardStackID  uuid.UUID `json:"forward_stack_id,omitempty" db:"forward_stack_id,uuid"`
	ForwardActionID uuid.UUID `json:"forward_action_id,omitempty" db:"forward_action_id,uuid"`

	// execute
	ExecuteCount    uint64          `json:"execute_count,omitempty" db:"execute_count"`
	ExecutedActions []action.Action `json:"executed_actions,omitempty" db:"executed_actions,json"`

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}
```

**Step 3: Verify models compile**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go build ./models/activeflow/...`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add models/activeflow/activeflow.go
git commit -m "feat: add db tags to Activeflow model"
```

---

## Task 15: Update Activeflow CRUD Operations to Use dbutil

**Files:**
- Modify: `pkg/dbhandler/activeflow.go`

**Step 1: Update activeflowGetFromRow**

Replace `activeflowGetFromRow` method in `pkg/dbhandler/activeflow.go`:

```go
// activeflowGetFromRow gets the activeflow from the row.
func (h *handler) activeflowGetFromRow(row *sql.Rows) (*activeflow.Activeflow, error) {
	res := &activeflow.Activeflow{}

	if err := dbutil.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. activeflowGetFromRow. err: %v", err)
	}

	return res, nil
}
```

**Step 2: Remove activeflowFields variable**

Delete the `activeflowFields` variable at the top of `pkg/dbhandler/activeflow.go`.

**Step 3: Update ActiveflowCreate to use dbutil**

Replace `ActiveflowCreate` method:

```go
func (h *handler) ActiveflowCreate(ctx context.Context, af *activeflow.Activeflow) error {
	now := h.util.TimeGetCurTime()

	// Set timestamps
	af.TMCreate = now
	af.TMUpdate = commondatabasehandler.DefaultTimeStamp
	af.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use dbutil to get fields and values
	fields := dbutil.GetDBFields(af)
	values, err := dbutil.PrepareValues(af)
	if err != nil {
		return fmt.Errorf("could not prepare values. ActiveflowCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(activeflowsTable).
		Columns(fields...).
		Values(values...).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ActiveflowCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ActiveflowCreate. err: %v", err)
	}

	_ = h.activeflowUpdateToCache(ctx, af.ID)
	return nil
}
```

**Step 4: Update read operations to use dbutil.GetDBFields**

Update `ActiveflowGets`, `activeflowGetFromDB` similar to how Flow methods were updated (replace hardcoded field lists with `dbutil.GetDBFields(&activeflow.Activeflow{})`).

**Step 5: Run activeflow tests**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbhandler/... -v -run ".*Activeflow.*"`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/dbhandler/activeflow.go
git commit -m "refactor: update activeflow operations to use dbutil helpers"
```

---

## Task 16: Add DB Tags to Variable Model and Migrate Operations

**Files:**
- Modify: `models/variable/variable.go`
- Modify: `pkg/dbhandler/variable.go`

**Step 1: Add db tags to Variable struct**

Edit `models/variable/variable.go` to add db tags (check existing field names in `pkg/dbhandler/variable.go` first).

**Step 2: Update variable CRUD operations**

Similar pattern to Flow and Activeflow:
- Replace `variableGetFromRow` to use `dbutil.ScanRow`
- Update `VariableCreate` to use `dbutil.PrepareValues`
- Replace hardcoded field lists with `dbutil.GetDBFields`

**Step 3: Run variable tests**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./pkg/dbhandler/... -v -run ".*Variable.*"`
Expected: PASS

**Step 4: Commit**

```bash
git add models/variable/variable.go pkg/dbhandler/variable.go
git commit -m "refactor: migrate variable operations to use dbutil helpers"
```

---

## Task 17: Final Integration Testing and Cleanup

**Files:**
- None (verification and cleanup)

**Step 1: Run complete test suite**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && go test ./... -v`
Expected: PASS (all tests pass)

**Step 2: Run linter**

Run: `cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager && golangci-lint run -v --timeout 5m`
Expected: PASS

**Step 3: Run pre-commit workflow from CLAUDE.md**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All steps succeed

**Step 4: Verify no unused code remains**

Search for any old field list variables that weren't removed:
- No `flowsFields`
- No `activeflowFields`
- No `variableFields`

**Step 5: Manual integration test (recommended)**

Start flow-manager and test critical flows:
1. Create flow with actions
2. Create activeflow
3. Execute activeflow
4. Query flows and activeflows
5. Update operations

**Step 6: Update CLAUDE.md if needed**

Document the new dbutil package and db tag convention in `CLAUDE.md` for future reference.

---

## Task 18: Update bin-common-handler Dependents (If Identity Changed)

**Files:**
- All services in monorepo that depend on bin-common-handler

**Step 1: Check if Identity struct was modified**

If we added db tags to `bin-common-handler/models/identity/identity.go`, we need to update all dependent services.

**Step 2: Run update workflow from root**

From monorepo root:
```bash
find . -maxdepth 2 -name "go.mod" -execdir bash -c "go mod tidy && go mod vendor" \;
```

**Step 3: Verify other services still compile**

Test a few other services:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-api-manager && go build ./...
cd /home/pchero/gitvoipbin/monorepo/bin-call-manager && go build ./...
```

Expected: SUCCESS (db tags don't break anything)

**Step 4: Commit updates to other services**

```bash
git add ../bin-api-manager/go.mod ../bin-api-manager/go.sum ../bin-api-manager/vendor
git add ../bin-call-manager/go.mod ../bin-call-manager/go.sum ../bin-call-manager/vendor
# ... other services as needed
git commit -m "chore: update dependencies after Identity db tags addition"
```

---

## Success Criteria

After completing all tasks:

✅ All tests pass: `go test ./...`
✅ Linter passes: `golangci-lint run`
✅ Pre-commit workflow succeeds
✅ Manual integration testing shows no regressions
✅ No hardcoded field lists remain (flowsFields, etc.)
✅ All model structs have db tags
✅ Code is cleaner and more maintainable

## Next Steps (Future Enhancements)

After this implementation is stable, consider:

1. **Add ValidateModel helper** - Validate struct tags at startup
2. **Performance benchmarking** - Measure reflection overhead
3. **Extend to other services** - Apply same pattern to other managers
4. **Add pointer field support** - For truly nullable columns (`*string`, `*int`)
5. **Cache reflection results** - Optimize GetDBFields with sync.Map cache
