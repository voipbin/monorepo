# Database Handler Consolidation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Consolidate database mapping utilities by moving flow-manager's dbutil to bin-common-handler, creating a unified PrepareFields API for all services.

**Architecture:** Copy dbutil functions to bin-common-handler/pkg/databasehandler, implement PrepareFields that handles both structs (tag-aware) and maps (tag-agnostic), migrate flow-manager to use the new API, and enable other services to adopt incrementally.

**Tech Stack:** Go reflection, Squirrel query builder, SQL database operations, struct tags

---

## Task 1: Copy GetDBFields to bin-common-handler

**Files:**
- Create: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping.go`
- Read: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbutil/main.go`
- Test: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_test.go`

**Context:** Start by copying the GetDBFields function and its recursive helper. This function reads struct tags and returns column names. It will remain as a public helper for SELECT queries.

**Step 1: Write test for GetDBFields**

Create `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_test.go`:

```go
package databasehandler

import (
	"testing"

	"github.com/gofrs/uuid"
)

// Test model
type testModel struct {
	ID     uuid.UUID `db:"id,uuid"`
	Name   string    `db:"name"`
	Count  int       `db:"count"`
	SkipMe bool      `db:"-"`
}

func TestGetDBFields_Basic(t *testing.T) {
	tests := []struct {
		name     string
		model    interface{}
		expected []string
	}{
		{
			name:     "basic model with UUID and string fields",
			model:    &testModel{},
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
			name: "handles embedded structs",
			model: &struct {
				testModel
				Extra string `db:"extra"`
			}{},
			expected: []string{"id", "name", "count", "extra"},
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

Run from bin-common-handler directory:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestGetDBFields_Basic -v
```

Expected: FAIL with "undefined: GetDBFields"

**Step 3: Copy GetDBFields implementation**

Create `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping.go`:

```go
package databasehandler

import (
	"reflect"
	"strings"
)

// GetDBFields returns ordered column names from struct tags
// Reads db:"column_name[,conversion_type]" tags from struct fields
// Skips fields tagged with db:"-"
// Recursively processes embedded structs
func GetDBFields(model interface{}) []string {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	return getDBFieldsRecursive(val, typ)
}

// getDBFieldsRecursive recursively extracts column names from struct fields
func getDBFieldsRecursive(val reflect.Value, typ reflect.Type) []string {
	var fields []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Handle embedded/anonymous structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			embeddedFields := getDBFieldsRecursive(fieldVal, field.Type)
			fields = append(fields, embeddedFields...)
			continue
		}

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get db tag
		tag := field.Tag.Get("db")
		if tag == "" {
			continue
		}

		// Skip fields with db:"-"
		if tag == "-" {
			continue
		}

		// Extract column name (before comma if conversion type specified)
		columnName := tag
		if idx := strings.Index(tag, ","); idx != -1 {
			columnName = tag[:idx]
		}

		fields = append(fields, columnName)
	}

	return fields
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestGetDBFields_Basic -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
git add pkg/databasehandler/mapping.go pkg/databasehandler/mapping_test.go
git commit -m "feat(databasehandler): add GetDBFields for struct tag parsing"
```

---

## Task 2: Add convertValue helper for type conversions

**Files:**
- Create: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_internal.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_test.go`

**Context:** Create the shared conversion logic that handles UUID→bytes and JSON marshaling. This will be used by both prepareFieldsFromStruct and prepareFieldsFromMap.

**Step 1: Write test for convertValue**

Add to `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_test.go`:

```go
func TestConvertValue(t *testing.T) {
	tests := []struct {
		name           string
		value          interface{}
		conversionType string
		wantType       string  // Expected type name
		wantErr        bool
	}{
		{
			name:           "UUID to bytes",
			value:          uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000")),
			conversionType: "uuid",
			wantType:       "[]uint8",
			wantErr:        false,
		},
		{
			name:           "slice to JSON",
			value:          []string{"a", "b", "c"},
			conversionType: "json",
			wantType:       "[]uint8",
			wantErr:        false,
		},
		{
			name:           "primitive passthrough",
			value:          "test string",
			conversionType: "",
			wantType:       "string",
			wantErr:        false,
		},
		{
			name:           "int passthrough",
			value:          42,
			conversionType: "",
			wantType:       "int",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertValue(tt.value, tt.conversionType)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				gotType := reflect.TypeOf(result).String()
				if gotType != tt.wantType {
					t.Errorf("convertValue() type = %v, want %v", gotType, tt.wantType)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestConvertValue -v
```

Expected: FAIL with "undefined: convertValue"

**Step 3: Implement convertValue**

Create `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_internal.go`:

```go
package databasehandler

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"
)

// convertValue converts a value based on conversion type
// - "uuid": uuid.UUID → []byte via .Bytes()
// - "json": any type → []byte via json.Marshal()
// - "": primitives pass through unchanged
func convertValue(value interface{}, conversionType string) (interface{}, error) {
	// Handle special conversion types
	if conversionType == "uuid" {
		if uuidVal, ok := value.(uuid.UUID); ok {
			return uuidVal.Bytes(), nil
		}
		return nil, fmt.Errorf("expected uuid.UUID for uuid conversion, got %T", value)
	}

	if conversionType == "json" {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("JSON marshal failed: %w", err)
		}
		return jsonBytes, nil
	}

	// Auto-detect JSON types if no conversion type specified
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map, reflect.Struct:
		// Only auto-marshal if it's not a basic type
		if rv.Kind() == reflect.Struct {
			// Skip basic structs like uuid.UUID (already handled above)
			if _, isUUID := value.(uuid.UUID); isUUID {
				return value, nil
			}
		}
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("JSON marshal failed: %w", err)
		}
		return jsonBytes, nil
	}

	// Primitives pass through unchanged
	return value, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestConvertValue -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
git add pkg/databasehandler/mapping_internal.go pkg/databasehandler/mapping_test.go
git commit -m "feat(databasehandler): add convertValue helper for type conversions"
```

---

## Task 3: Implement prepareFieldsFromStruct (tag-aware path)

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_internal.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_test.go`

**Context:** Implement the struct path for PrepareFields. This reads db tags and converts values accordingly.

**Step 1: Write test for prepareFieldsFromStruct**

Add to `mapping_test.go`:

```go
func TestPrepareFieldsFromStruct(t *testing.T) {
	id := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	tests := []struct {
		name      string
		input     interface{}
		wantKeys  []string
		checkFunc func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic struct with primitives",
			input: &struct {
				Name  string `db:"name"`
				Count int    `db:"count"`
			}{
				Name:  "test",
				Count: 42,
			},
			wantKeys: []string{"name", "count"},
			checkFunc: func(t *testing.T, result map[string]any) {
				if result["name"] != "test" {
					t.Errorf("name = %v, want test", result["name"])
				}
				if result["count"] != 42 {
					t.Errorf("count = %v, want 42", result["count"])
				}
			},
		},
		{
			name: "struct with UUID conversion",
			input: &struct {
				ID uuid.UUID `db:"id,uuid"`
			}{
				ID: id,
			},
			wantKeys: []string{"id"},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["id"].([]byte)
				if !ok {
					t.Errorf("id type = %T, want []byte", result["id"])
				}
				if len(bytes) != 16 {
					t.Errorf("id length = %d, want 16", len(bytes))
				}
			},
		},
		{
			name: "struct with JSON conversion",
			input: &struct {
				Tags []string `db:"tags,json"`
			}{
				Tags: []string{"a", "b"},
			},
			wantKeys: []string{"tags"},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["tags"].([]byte)
				if !ok {
					t.Errorf("tags type = %T, want []byte", result["tags"])
				}
				if string(bytes) != `["a","b"]` {
					t.Errorf("tags = %s, want [\"a\",\"b\"]", string(bytes))
				}
			},
		},
		{
			name: "skips db:\"-\" fields",
			input: &struct {
				Name   string `db:"name"`
				Secret string `db:"-"`
			}{
				Name:   "test",
				Secret: "hidden",
			},
			wantKeys: []string{"name"},
			checkFunc: func(t *testing.T, result map[string]any) {
				if _, exists := result["Secret"]; exists {
					t.Errorf("Secret field should be skipped")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.input)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}

			result, err := prepareFieldsFromStruct(val)
			if err != nil {
				t.Fatalf("prepareFieldsFromStruct() error = %v", err)
			}

			if len(result) != len(tt.wantKeys) {
				t.Errorf("got %d fields, want %d", len(result), len(tt.wantKeys))
			}

			for _, key := range tt.wantKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("missing key: %s", key)
				}
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestPrepareFieldsFromStruct -v
```

Expected: FAIL with "undefined: prepareFieldsFromStruct"

**Step 3: Implement prepareFieldsFromStruct**

Add to `mapping_internal.go`:

```go
// prepareFieldsFromStruct processes a struct value using db tags
// Reads tags, skips db:"-" fields, applies conversions
func prepareFieldsFromStruct(val reflect.Value) (map[string]any, error) {
	typ := val.Type()
	result := make(map[string]any)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Handle embedded/anonymous structs recursively
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			embeddedFields, err := prepareFieldsFromStruct(fieldVal)
			if err != nil {
				return nil, fmt.Errorf("embedded struct %s: %w", field.Name, err)
			}
			// Merge embedded fields into result
			for k, v := range embeddedFields {
				result[k] = v
			}
			continue
		}

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get db tag
		tag := field.Tag.Get("db")
		if tag == "" {
			continue
		}

		// Skip fields with db:"-"
		if tag == "-" {
			continue
		}

		// Parse tag: column_name[,conversion_type]
		columnName := tag
		conversionType := ""
		if idx := strings.Index(tag, ","); idx != -1 {
			columnName = tag[:idx]
			conversionType = tag[idx+1:]
		}

		// Convert value
		convertedVal, err := convertValue(fieldVal.Interface(), conversionType)
		if err != nil {
			return nil, fmt.Errorf("field %s (%s): %w", field.Name, columnName, err)
		}

		result[columnName] = convertedVal
	}

	return result, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestPrepareFieldsFromStruct -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
git add pkg/databasehandler/mapping_internal.go pkg/databasehandler/mapping_test.go
git commit -m "feat(databasehandler): implement prepareFieldsFromStruct with tag parsing"
```

---

## Task 4: Implement prepareFieldsFromMap (tag-agnostic path)

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_internal.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_test.go`

**Context:** Implement the map path for PrepareFields. This detects types by reflection and applies conversions without reading tags.

**Step 1: Write test for prepareFieldsFromMap**

Add to `mapping_test.go`:

```go
func TestPrepareFieldsFromMap(t *testing.T) {
	id := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	tests := []struct {
		name      string
		input     map[string]any
		checkFunc func(t *testing.T, result map[string]any)
	}{
		{
			name: "primitives pass through",
			input: map[string]any{
				"name":  "test",
				"count": 42,
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				if result["name"] != "test" {
					t.Errorf("name = %v, want test", result["name"])
				}
				if result["count"] != 42 {
					t.Errorf("count = %v, want 42", result["count"])
				}
			},
		},
		{
			name: "UUID auto-detected and converted",
			input: map[string]any{
				"id": id,
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["id"].([]byte)
				if !ok {
					t.Errorf("id type = %T, want []byte", result["id"])
				}
				if len(bytes) != 16 {
					t.Errorf("id length = %d, want 16", len(bytes))
				}
			},
		},
		{
			name: "slice auto-marshaled to JSON",
			input: map[string]any{
				"tags": []string{"a", "b"},
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["tags"].([]byte)
				if !ok {
					t.Errorf("tags type = %T, want []byte", result["tags"])
				}
			},
		},
		{
			name: "nil value preserved",
			input: map[string]any{
				"optional": nil,
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				if result["optional"] != nil {
					t.Errorf("optional = %v, want nil", result["optional"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareFieldsFromMap(tt.input)
			if err != nil {
				t.Fatalf("prepareFieldsFromMap() error = %v", err)
			}

			if len(result) != len(tt.input) {
				t.Errorf("got %d fields, want %d", len(result), len(tt.input))
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestPrepareFieldsFromMap -v
```

Expected: FAIL with "undefined: prepareFieldsFromMap"

**Step 3: Implement prepareFieldsFromMap**

Add to `mapping_internal.go`:

```go
// prepareFieldsFromMap processes a map without tag awareness
// Auto-detects UUID and JSON types, applies conversions
func prepareFieldsFromMap(data any) (map[string]any, error) {
	inputMap, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map[string]any, got %T", data)
	}

	result := make(map[string]any, len(inputMap))

	for key, value := range inputMap {
		// Preserve nil values
		if value == nil {
			result[key] = nil
			continue
		}

		// Auto-detect UUID
		if uuidVal, ok := value.(uuid.UUID); ok {
			result[key] = uuidVal.Bytes()
			continue
		}

		// Auto-detect complex types that need JSON marshaling
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Map:
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("field %s: JSON marshal failed: %w", key, err)
			}
			result[key] = jsonBytes

		case reflect.Struct:
			// Skip UUID (already handled)
			if _, isUUID := value.(uuid.UUID); !isUUID {
				jsonBytes, err := json.Marshal(value)
				if err != nil {
					return nil, fmt.Errorf("field %s: JSON marshal failed: %w", key, err)
				}
				result[key] = jsonBytes
			} else {
				result[key] = value
			}

		default:
			// Primitives pass through
			result[key] = value
		}
	}

	return result, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestPrepareFieldsFromMap -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
git add pkg/databasehandler/mapping_internal.go pkg/databasehandler/mapping_test.go
git commit -m "feat(databasehandler): implement prepareFieldsFromMap with auto type detection"
```

---

## Task 5: Implement PrepareFields main function

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_test.go`

**Context:** Create the main PrepareFields function that routes to struct or map path based on input type.

**Step 1: Write test for PrepareFields**

Add to `mapping_test.go`:

```go
func TestPrepareFields(t *testing.T) {
	id := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	t.Run("accepts struct", func(t *testing.T) {
		input := &testModel{
			ID:    id,
			Name:  "test",
			Count: 42,
		}

		result, err := PrepareFields(input)
		if err != nil {
			t.Fatalf("PrepareFields() error = %v", err)
		}

		if len(result) != 3 { // id, name, count (SkipMe excluded)
			t.Errorf("got %d fields, want 3", len(result))
		}

		if _, exists := result["id"]; !exists {
			t.Errorf("missing id field")
		}
		if result["name"] != "test" {
			t.Errorf("name = %v, want test", result["name"])
		}
		if result["count"] != 42 {
			t.Errorf("count = %v, want 42", result["count"])
		}
	})

	t.Run("accepts map", func(t *testing.T) {
		input := map[string]any{
			"name":  "updated",
			"count": 100,
		}

		result, err := PrepareFields(input)
		if err != nil {
			t.Fatalf("PrepareFields() error = %v", err)
		}

		if len(result) != 2 {
			t.Errorf("got %d fields, want 2", len(result))
		}

		if result["name"] != "updated" {
			t.Errorf("name = %v, want updated", result["name"])
		}
	})

	t.Run("rejects invalid type", func(t *testing.T) {
		input := []string{"not", "supported"}

		_, err := PrepareFields(input)
		if err == nil {
			t.Errorf("PrepareFields() expected error for slice input")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestPrepareFields -v
```

Expected: FAIL with "undefined: PrepareFields"

**Step 3: Implement PrepareFields**

Add to `mapping.go`:

```go
// PrepareFields converts struct or map to database-ready values
// - Struct input: reads db tags, skips db:"-", converts UUID/JSON based on tags
// - Map input: auto-detects types, converts UUID/JSON without tag filtering
// Returns map[string]any suitable for squirrel.Insert().SetMap() or Update().SetMap()
func PrepareFields(data any) (map[string]any, error) {
	val := reflect.ValueOf(data)

	// Dereference pointer if needed
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		// Tag-aware path for INSERT with structs
		return prepareFieldsFromStruct(val)

	case reflect.Map:
		// Tag-agnostic path for UPDATE with maps
		return prepareFieldsFromMap(data)

	default:
		return nil, fmt.Errorf("PrepareFields: expected struct or map, got %T", data)
	}
}
```

Also add import for `fmt` if not already present at top of file.

**Step 4: Run test to verify it passes**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestPrepareFields -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
git add pkg/databasehandler/mapping.go pkg/databasehandler/mapping_test.go
git commit -m "feat(databasehandler): add PrepareFields unified API for struct and map"
```

---

## Task 6: Copy ScanRow from dbutil

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbutil/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_internal.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/mapping_test.go`

**Context:** Copy ScanRow and its internal helpers from flow-manager's dbutil. This function is already well-tested and doesn't need changes.

**Step 1: Copy ScanRow tests from dbutil**

Read the ScanRow tests from `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbutil/main_test.go` and add relevant ones to `mapping_test.go`. For brevity, add a basic test:

```go
func TestScanRow_Basic(t *testing.T) {
	// Note: Full tests will be copied from dbutil/main_test.go
	// This is a minimal test to verify the function exists

	t.Run("rejects non-pointer", func(t *testing.T) {
		// Create mock rows (simplified - full implementation uses test helper)
		var dest testModel
		err := ScanRow(nil, dest) // Non-pointer

		if err == nil {
			t.Errorf("ScanRow should reject non-pointer destination")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestScanRow_Basic -v
```

Expected: FAIL with "undefined: ScanRow"

**Step 3: Copy ScanRow implementation**

Copy the ScanRow function and its helpers from `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbutil/main.go` to `mapping.go` and `mapping_internal.go`.

Add to `mapping.go`:

```go
// ScanRow scans sql.Rows into struct using db tags
// Handles NULL values, UUID bytes→UUID, JSON string→struct
// Automatically applies conversions based on db tag conversion types
func ScanRow(row *sql.Rows, dest any) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("ScanRow: dest must be pointer to struct, got %T", dest)
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("ScanRow: dest must be pointer to struct, got pointer to %s", val.Kind())
	}

	typ := val.Type()

	// Build scan targets for all db-tagged fields
	scanTargets, fieldPtrs := buildScanTargetsRecursive(val, typ)

	// Scan into targets
	if err := row.Scan(scanTargets...); err != nil {
		return fmt.Errorf("ScanRow: scan failed: %w", err)
	}

	// Copy from NULL-safe targets to actual struct fields
	for i, target := range scanTargets {
		fieldPtr := fieldPtrs[i]
		conversionType := fieldPtr.conversionType

		if err := copyFromNullType(target, &fieldPtr.fieldVal, conversionType); err != nil {
			return fmt.Errorf("ScanRow: field %s: %w", fieldPtr.name, err)
		}
	}

	return nil
}
```

Add to `mapping_internal.go` (copy the helper functions from dbutil):

```go
type fieldPtr struct {
	name           string
	fieldVal       reflect.Value
	conversionType string
}

// buildScanTargetsRecursive builds scan targets for all db-tagged fields
func buildScanTargetsRecursive(val reflect.Value, typ reflect.Type) ([]any, []fieldPtr) {
	var scanTargets []any
	var fieldPtrs []fieldPtr

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Handle embedded structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			embeddedTargets, embeddedPtrs := buildScanTargetsRecursive(fieldVal, field.Type)
			scanTargets = append(scanTargets, embeddedTargets...)
			fieldPtrs = append(fieldPtrs, embeddedPtrs...)
			continue
		}

		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}

		// Parse conversion type
		conversionType := ""
		if idx := strings.Index(tag, ","); idx != -1 {
			conversionType = tag[idx+1:]
		}

		// Create NULL-safe scan target
		target := createNullScanTarget(fieldVal, conversionType)
		scanTargets = append(scanTargets, target)
		fieldPtrs = append(fieldPtrs, fieldPtr{
			name:           field.Name,
			fieldVal:       fieldVal,
			conversionType: conversionType,
		})
	}

	return scanTargets, fieldPtrs
}

// createNullScanTarget creates sql.Null* wrapper for field
func createNullScanTarget(fieldVal reflect.Value, conversionType string) any {
	if conversionType == "uuid" || conversionType == "json" {
		return new(sql.NullString)
	}

	switch fieldVal.Kind() {
	case reflect.String:
		return new(sql.NullString)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return new(sql.NullInt64)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return new(sql.NullInt64)
	case reflect.Float32, reflect.Float64:
		return new(sql.NullFloat64)
	case reflect.Bool:
		return new(sql.NullBool)
	default:
		return fieldVal.Addr().Interface()
	}
}

// copyFromNullType copies value from sql.Null* to field
func copyFromNullType(scanTarget any, fieldVal *reflect.Value, conversionType string) error {
	if conversionType == "uuid" {
		nullStr := scanTarget.(*sql.NullString)
		if nullStr.Valid && len(nullStr.String) > 0 {
			uuidVal, err := uuid.FromBytes([]byte(nullStr.String))
			if err != nil {
				return fmt.Errorf("UUID conversion failed: %w", err)
			}
			fieldVal.Set(reflect.ValueOf(uuidVal))
		} else {
			fieldVal.Set(reflect.ValueOf(uuid.Nil))
		}
		return nil
	}

	if conversionType == "json" {
		nullStr := scanTarget.(*sql.NullString)
		if nullStr.Valid && len(nullStr.String) > 0 {
			if err := json.Unmarshal([]byte(nullStr.String), fieldVal.Addr().Interface()); err != nil {
				return fmt.Errorf("JSON unmarshal failed: %w", err)
			}
		}
		return nil
	}

	// Handle basic types
	switch target := scanTarget.(type) {
	case *sql.NullString:
		if target.Valid {
			fieldVal.SetString(target.String)
		}
	case *sql.NullInt64:
		if target.Valid {
			fieldVal.SetInt(target.Int64)
		}
	case *sql.NullFloat64:
		if target.Valid {
			fieldVal.SetFloat(target.Float64)
		}
	case *sql.NullBool:
		if target.Valid {
			fieldVal.SetBool(target.Bool)
		}
	}

	return nil
}
```

Add imports to `mapping.go`:
```go
import (
	"database/sql"
	// ... other imports
)
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestScanRow_Basic -v
```

Expected: PASS (even with minimal test)

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
git add pkg/databasehandler/mapping.go pkg/databasehandler/mapping_internal.go pkg/databasehandler/mapping_test.go
git commit -m "feat(databasehandler): add ScanRow for struct scanning with NULL handling"
```

---

## Task 7: Update PrepareUpdateFields to use PrepareFields

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-common-handler/pkg/databasehandler/main_test.go`

**Context:** Update the existing PrepareUpdateFields to delegate to PrepareFields internally. Mark it as deprecated for future reference.

**Step 1: Write backward compatibility test**

Add to `main_test.go`:

```go
func TestPrepareUpdateFields_BackwardCompatibility(t *testing.T) {
	id := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	input := map[string]any{
		"id":   id,
		"name": "test",
	}

	// Old API
	oldResult := PrepareUpdateFields(input)

	// New API
	newResult, err := PrepareFields(input)
	if err != nil {
		t.Fatalf("PrepareFields() error = %v", err)
	}

	// Results should be identical
	if len(oldResult) != len(newResult) {
		t.Errorf("length mismatch: old=%d, new=%d", len(oldResult), len(newResult))
	}

	for key, oldVal := range oldResult {
		newVal, exists := newResult[key]
		if !exists {
			t.Errorf("key %s missing in new result", key)
			continue
		}

		// Compare byte slices properly
		oldBytes, oldIsBytes := oldVal.([]byte)
		newBytes, newIsBytes := newVal.([]byte)

		if oldIsBytes && newIsBytes {
			if string(oldBytes) != string(newBytes) {
				t.Errorf("key %s: bytes mismatch", key)
			}
		} else if oldVal != newVal {
			t.Errorf("key %s: old=%v, new=%v", key, oldVal, newVal)
		}
	}
}
```

**Step 2: Run test to verify current behavior**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestPrepareUpdateFields_BackwardCompatibility -v
```

Expected: PASS (shows current behavior works)

**Step 3: Update PrepareUpdateFields implementation**

Modify in `main.go`:

```go
// PrepareUpdateFields processes a map of fields for update operations.
// Deprecated: Use PrepareFields instead, which handles both structs and maps.
// This function is kept for backward compatibility and now delegates to PrepareFields.
func PrepareUpdateFields[K ~string](fields map[K]any) map[string]any {
	// Convert K ~string keys to string
	stringMap := make(map[string]any, len(fields))
	for k, v := range fields {
		stringMap[string(k)] = v
	}

	// Delegate to PrepareFields
	result, err := PrepareFields(stringMap)
	if err != nil {
		// For backward compatibility, return original behavior on error
		// (original didn't return errors)
		logrus.Warnf("PrepareUpdateFields: PrepareFields error, falling back: %v", err)

		// Fallback to original logic (keep old implementation as backup)
		res := make(map[string]any, len(fields))
		for k, v := range fields {
			key := string(k)

			switch val := v.(type) {
			case uuid.UUID:
				res[key] = val.Bytes()
			case json.Marshaler:
				b, err := val.MarshalJSON()
				if err == nil {
					res[key] = b
				} else {
					res[key] = nil
				}
			default:
				rv := reflect.ValueOf(v)
				rt := rv.Type()
				if rt.Kind() == reflect.Map || rt.Kind() == reflect.Slice || rt.Kind() == reflect.Struct {
					b, err := json.Marshal(v)
					if err == nil {
						res[key] = b
					} else {
						res[key] = nil
					}
				} else {
					res[key] = v
				}
			}
		}
		return res
	}

	return result
}
```

**Step 4: Run test to verify it still passes**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./pkg/databasehandler -run TestPrepareUpdateFields_BackwardCompatibility -v
go test ./pkg/databasehandler -v  # Run all tests
```

Expected: PASS (backward compatibility maintained)

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
git add pkg/databasehandler/main.go pkg/databasehandler/main_test.go
git commit -m "refactor(databasehandler): update PrepareUpdateFields to use PrepareFields internally"
```

---

## Task 8: Run full test suite for bin-common-handler

**Context:** Verify all tests pass in bin-common-handler after adding the new mapping functions.

**Step 1: Run all tests**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test ./... -v
```

Expected: All tests PASS

**Step 2: Check coverage**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
go test -coverprofile=coverage.out ./pkg/databasehandler/...
go tool cover -func=coverage.out | grep "databasehandler.*mapping"
```

Expected: 80%+ coverage for new mapping functions

**Step 3: Run linter**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-common-handler
golangci-lint run -v --timeout 5m
```

Expected: 0 issues

**Step 4: Commit test results**

If all tests pass and linter is clean, ready to proceed to Phase 2.

---

## Task 9: Update flow-manager imports to use databasehandler

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/flows.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/activeflow.go`

**Context:** Update imports in flow-manager to use the new databasehandler package instead of dbutil.

**Step 1: Update imports in flows.go**

Modify `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/flows.go`:

```go
// Change:
import "monorepo/bin-flow-manager/pkg/dbutil"

// To:
import commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
```

Note: The alias `commondatabasehandler` is already used in the file, so this maintains consistency.

**Step 2: Update imports in activeflow.go**

Modify `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/activeflow.go`:

Same change as above - replace dbutil import with databasehandler import.

**Step 3: Verify imports compile**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go build ./pkg/dbhandler/...
```

Expected: Build errors about undefined functions (we'll fix in next tasks)

**Step 4: Commit import changes**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
git add pkg/dbhandler/flows.go pkg/dbhandler/activeflow.go
git commit -m "refactor(dbhandler): update imports to use common databasehandler"
```

---

## Task 10: Update FlowCreate to use PrepareFields with SetMap

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/flows.go`

**Context:** Update FlowCreate to use PrepareFields + SetMap instead of GetDBFields + PrepareValues + Columns/Values.

**Step 1: Update FlowCreate implementation**

Modify `FlowCreate` in `flows.go`:

```go
func (h *handler) FlowCreate(ctx context.Context, f *flow.Flow) error {
	now := h.util.TimeGetCurTime()

	// Set timestamps
	f.TMCreate = now
	f.TMUpdate = commondatabasehandler.DefaultTimeStamp
	f.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(f)
	if err != nil {
		return fmt.Errorf("could not prepare fields. FlowCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(flowsTable).
		SetMap(fields).
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

**Step 2: Test FlowCreate still works**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go test ./pkg/dbhandler -run Test_FlowCreate -v
```

Expected: PASS

**Step 3: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
git add pkg/dbhandler/flows.go
git commit -m "refactor(dbhandler): update FlowCreate to use PrepareFields with SetMap"
```

---

## Task 11: Update flow read operations to use GetDBFields and ScanRow

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/flows.go`

**Context:** Update flowGetFromDB, FlowGets, and flowGetFromRow to use databasehandler functions.

**Step 1: Update flowGetFromRow**

Already uses dbutil.ScanRow - just change the package prefix:

```go
func (h *handler) flowGetFromRow(row *sql.Rows) (*flow.Flow, error) {
	res := &flow.Flow{}

	// Change: dbutil.ScanRow → commondatabasehandler.ScanRow
	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. flowGetFromRow. err: %v", err)
	}

	res.Persist = true
	return res, nil
}
```

**Step 2: Update flowGetFromDB**

Change `dbutil.GetDBFields` → `commondatabasehandler.GetDBFields`:

```go
func (h *handler) flowGetFromDB(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	// Change: dbutil.GetDBFields → commondatabasehandler.GetDBFields
	fields := commondatabasehandler.GetDBFields(&flow.Flow{})

	query, args, err := squirrel.
		Select(fields...).
		From(flowsTable).
		Where(squirrel.Eq{string(flow.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()

	// ... rest unchanged
}
```

**Step 3: Update FlowGets**

Same change - replace `dbutil.GetDBFields` with `commondatabasehandler.GetDBFields`:

```go
func (h *handler) FlowGets(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	// Change: dbutil.GetDBFields → commondatabasehandler.GetDBFields
	fields := commondatabasehandler.GetDBFields(&flow.Flow{})

	// ... rest unchanged
}
```

**Step 4: Test flow read operations**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go test ./pkg/dbhandler -run Test.*Flow -v
```

Expected: All flow tests PASS

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
git add pkg/dbhandler/flows.go
git commit -m "refactor(dbhandler): update flow read operations to use databasehandler"
```

---

## Task 12: Update ActiveflowCreate to use PrepareFields with SetMap

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/activeflow.go`

**Context:** Same changes as FlowCreate - use PrepareFields + SetMap.

**Step 1: Update ActiveflowCreate**

Modify `ActiveflowCreate` in `activeflow.go`:

```go
func (h *handler) ActiveflowCreate(ctx context.Context, f *activeflow.Activeflow) error {
	now := h.util.TimeGetCurTime()

	// Set timestamps
	f.TMCreate = now
	f.TMUpdate = commondatabasehandler.DefaultTimeStamp
	f.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields
	fields, err := commondatabasehandler.PrepareFields(f)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ActiveflowCreate. err: %v", err)
	}

	// Use SetMap
	sb := squirrel.
		Insert(activeflowsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ActiveflowCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ActiveflowCreate. err: %v", err)
	}

	_ = h.activeflowUpdateToCache(ctx, f.ID)
	return nil
}
```

**Step 2: Test ActiveflowCreate**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go test ./pkg/dbhandler -run Test_ActiveflowCreate -v
```

Expected: PASS

**Step 3: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
git add pkg/dbhandler/activeflow.go
git commit -m "refactor(dbhandler): update ActiveflowCreate to use PrepareFields with SetMap"
```

---

## Task 13: Update activeflow read operations

**Files:**
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbhandler/activeflow.go`

**Context:** Update activeflow read operations like we did for flow.

**Step 1: Update activeflowGetFromRow**

```go
func (h *handler) activeflowGetFromRow(row *sql.Rows) (*activeflow.Activeflow, error) {
	res := &activeflow.Activeflow{}

	// Change: dbutil.ScanRow → commondatabasehandler.ScanRow
	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. activeflowGetFromRow. err: %v", err)
	}

	return res, nil
}
```

**Step 2: Update activeflowGetFromDB and ActiveflowGets**

Same pattern - replace `dbutil.GetDBFields` with `commondatabasehandler.GetDBFields`:

```go
func (h *handler) activeflowGetFromDB(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	fields := commondatabasehandler.GetDBFields(&activeflow.Activeflow{})
	// ... rest unchanged
}

func (h *handler) ActiveflowGets(ctx context.Context, token string, size uint64, filters map[activeflow.Field]any) ([]*activeflow.Activeflow, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&activeflow.Activeflow{})
	// ... rest unchanged
}
```

**Step 3: Test activeflow operations**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go test ./pkg/dbhandler -run Test.*Activeflow -v
```

Expected: All activeflow tests PASS

**Step 4: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
git add pkg/dbhandler/activeflow.go
git commit -m "refactor(dbhandler): update activeflow read operations to use databasehandler"
```

---

## Task 14: Delete pkg/dbutil directory

**Files:**
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-flow-manager/pkg/dbutil/`

**Context:** Now that all code uses databasehandler, remove the old dbutil package.

**Step 1: Verify no imports remain**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
grep -r "pkg/dbutil" --include="*.go" .
```

Expected: No output (no imports found)

**Step 2: Delete dbutil directory**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
rm -rf pkg/dbutil/
```

**Step 3: Verify build still works**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go build ./...
```

Expected: SUCCESS

**Step 4: Commit deletion**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
git add -A
git commit -m "refactor: remove pkg/dbutil after migrating to common databasehandler"
```

---

## Task 15: Run full test suite for flow-manager

**Context:** Verify all tests pass after migration to databasehandler.

**Step 1: Run all tests**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go test ./... -v
```

Expected: All tests PASS

**Step 2: Check test coverage**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
go test -coverprofile=coverage.out ./pkg/dbhandler/...
go tool cover -func=coverage.out | tail -10
```

Expected: Coverage remains 85%+ (similar to before migration)

**Step 3: Run linter**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
golangci-lint run -v --timeout 5m
```

Expected: 0 issues

**Step 4: Document test results**

If all tests pass, ready for Phase 3.

---

## Task 16: Update bin-common-handler dependencies in all services

**Context:** After modifying bin-common-handler, all services need their vendor directories updated.

**Step 1: Run dependency update workflow**

Run from monorepo root:
```bash
cd /home/pchero/gitvoipbin/monorepo
find . -maxdepth 2 -name "go.mod" -execdir bash -c "echo 'Updating $(pwd)...' && go mod tidy && go mod vendor" \;
```

Expected: All services update successfully

**Step 2: Verify a few services compile**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-api-manager && go build ./...
cd /home/pchero/gitvoipbin/monorepo/bin-call-manager && go build ./...
```

Expected: Both compile successfully

**Step 3: Commit dependency updates**

```bash
cd /home/pchero/gitvoipbin/monorepo
git add */go.mod */go.sum */vendor
git commit -m "chore: update dependencies after databasehandler consolidation"
```

---

## Task 17: Push to remote and create summary

**Context:** Push all changes and prepare summary for review.

**Step 1: Push to remote**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/bin-flow-manager
git push origin VOIP-1189-flow-manager-fix-database-handle
```

**Step 2: Verify GitHub status**

Check that all commits are visible on GitHub.

**Step 3: Create summary**

Document what was accomplished:
- ✅ Added PrepareFields, ScanRow, GetDBFields to bin-common-handler
- ✅ Migrated flow-manager to use unified API
- ✅ Eliminated GetDBFields + PrepareValues → simplified to PrepareFields
- ✅ Changed INSERT to use SetMap for safer column/value matching
- ✅ Updated all dependent services
- ✅ All tests passing (85%+ coverage maintained)
- ✅ 0 linter issues

---

## Success Criteria

**Phase 1 Complete:**
- [ ] bin-common-handler has PrepareFields, ScanRow, GetDBFields
- [ ] All tests pass in bin-common-handler
- [ ] PrepareUpdateFields delegates to PrepareFields
- [ ] 85%+ test coverage

**Phase 2 Complete:**
- [ ] flow-manager uses databasehandler instead of dbutil
- [ ] FlowCreate uses PrepareFields + SetMap
- [ ] ActiveflowCreate uses PrepareFields + SetMap
- [ ] All read operations use GetDBFields and ScanRow
- [ ] pkg/dbutil deleted
- [ ] All tests pass in flow-manager
- [ ] 0 linter issues

**Phase 3 Complete:**
- [ ] All services' dependencies updated
- [ ] At least 2 other services compile successfully
- [ ] Changes pushed to remote

**Overall Benefits Achieved:**
- Simplified INSERT (one function instead of two)
- Safer INSERT (SetMap prevents column/value mismatch)
- Shared utilities (all services can now adopt)
- Reduced duplication (single UUID/JSON conversion logic)
