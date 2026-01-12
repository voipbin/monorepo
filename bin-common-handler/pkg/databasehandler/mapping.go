package databasehandler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofrs/uuid"
)

// ============================================================================
// Public API Functions
// ============================================================================

// GetDBFields returns ordered column names from struct tags
// Reads db:"column_name[,conversion_type]" tags from struct fields
// Skips fields tagged with db:"-"
// Recursively processes embedded structs
func GetDBFields(model interface{}) []string {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return []string{}
		}
		val = val.Elem()
	}

	return getDBFieldsRecursive(val, val.Type())
}

// PrepareFields converts struct or map to database-ready values
// - Struct input: reads db tags, skips db:"-", converts UUID/JSON based on tags
// - Map input: auto-detects types, converts UUID/JSON without tag filtering
// Returns map[string]any suitable for squirrel.Insert().SetMap() or Update().SetMap()
func PrepareFields(data any) (map[string]any, error) {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return prepareFieldsFromStruct(val)
	case reflect.Map:
		return prepareFieldsFromMap(data)
	default:
		return nil, fmt.Errorf("PrepareFields: expected struct or map, got %T", data)
	}
}

// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
// Handles NULL values by using sql.Null* types internally
// Supports UUID and JSON conversions via db tag conversion types
func ScanRow(row *sql.Rows, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	destVal = destVal.Elem()
	if destVal.Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	// Build scan targets with NULL handling
	targets := buildScanTargets(destVal)

	// Extract scan interfaces for row.Scan
	scanPtrs := make([]interface{}, len(targets))
	for i, t := range targets {
		scanPtrs[i] = t.scanPtr
	}

	// Scan the row
	if err := row.Scan(scanPtrs...); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Copy values from sql.Null* types to actual fields
	for _, t := range targets {
		if err := t.copyToField(); err != nil {
			return fmt.Errorf("copy to field failed: %w", err)
		}
	}

	return nil
}

// ============================================================================
// Internal: Type Conversion
// ============================================================================

// convertValueForDB converts a Go value to database-ready format
// This is the unified converter used by both struct and map processing
//
// Conversion rules:
// - conversionType "uuid": uuid.UUID → []byte
// - conversionType "json": any → []byte (JSON marshaled)
// - Auto-detect: Slice/Map/Struct (non-UUID) → []byte (JSON marshaled)
// - Primitives: pass through unchanged
func convertValueForDB(value interface{}, conversionType string) (interface{}, error) {
	// Explicit UUID conversion
	if conversionType == "uuid" {
		if uuidVal, ok := value.(uuid.UUID); ok {
			return uuidVal.Bytes(), nil
		}
		return nil, fmt.Errorf("expected uuid.UUID for uuid conversion, got %T", value)
	}

	// Explicit JSON conversion
	if conversionType == "json" {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("JSON marshal failed: %w", err)
		}
		return jsonBytes, nil
	}

	// Auto-detect: UUID passes through (will be handled by caller or detected later)
	if _, isUUID := value.(uuid.UUID); isUUID {
		return value, nil
	}

	// Auto-detect: complex types get JSON marshaled
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map, reflect.Struct:
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("JSON marshal failed: %w", err)
		}
		return jsonBytes, nil
	}

	// Primitives pass through unchanged
	return value, nil
}

// ============================================================================
// Internal: GetDBFields helpers
// ============================================================================

func getDBFieldsRecursive(val reflect.Value, typ reflect.Type) []string {
	var fields []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Handle embedded/anonymous structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			fields = append(fields, getDBFieldsRecursive(fieldVal, field.Type)...)
			continue
		}

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get db tag
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
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

// ============================================================================
// Internal: PrepareFields helpers
// ============================================================================

// prepareFieldsFromStruct processes a struct value using db tags
func prepareFieldsFromStruct(val reflect.Value) (map[string]any, error) {
	typ := val.Type()
	result := make(map[string]any)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Handle embedded/anonymous structs recursively
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			embedded, err := prepareFieldsFromStruct(fieldVal)
			if err != nil {
				return nil, fmt.Errorf("embedded struct %s: %w", field.Name, err)
			}
			for k, v := range embedded {
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
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: column_name[,conversion_type]
		columnName, conversionType := parseDBTag(tag)

		// Convert value
		converted, err := convertValueForDB(fieldVal.Interface(), conversionType)
		if err != nil {
			return nil, fmt.Errorf("field %s (%s): %w", field.Name, columnName, err)
		}

		result[columnName] = converted
	}

	return result, nil
}

// prepareFieldsFromMap processes a map without tag awareness
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

		// Auto-detect UUID and convert to bytes
		if uuidVal, ok := value.(uuid.UUID); ok {
			result[key] = uuidVal.Bytes()
			continue
		}

		// Use unified converter for other types (no explicit conversion type)
		converted, err := convertValueForDB(value, "")
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", key, err)
		}
		result[key] = converted
	}

	return result, nil
}

// parseDBTag extracts column name and conversion type from db tag
// Format: "column_name" or "column_name,conversion_type"
func parseDBTag(tag string) (columnName, conversionType string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tag[idx+1:]
	}
	return tag, ""
}

// ============================================================================
// Internal: ScanRow helpers
// ============================================================================

// scanTarget holds scan metadata for a single field
type scanTarget struct {
	fieldVal       reflect.Value
	scanPtr        interface{}
	conversionType string
}

// copyToField copies the scanned value to the struct field
func (t *scanTarget) copyToField() error {
	switch t.conversionType {
	case "uuid":
		return t.copyUUID()
	case "json":
		return t.copyJSON()
	default:
		return t.copyPrimitive()
	}
}

func (t *scanTarget) copyUUID() error {
	nullStr := t.scanPtr.(*sql.NullString)
	if nullStr.Valid && len(nullStr.String) > 0 {
		uuidVal, err := uuid.FromBytes([]byte(nullStr.String))
		if err != nil {
			return fmt.Errorf("cannot convert bytes to UUID: %w", err)
		}
		t.fieldVal.Set(reflect.ValueOf(uuidVal))
	} else {
		t.fieldVal.Set(reflect.ValueOf(uuid.Nil))
	}
	return nil
}

func (t *scanTarget) copyJSON() error {
	nullStr := t.scanPtr.(*sql.NullString)
	if nullStr.Valid && len(nullStr.String) > 0 {
		if err := json.Unmarshal([]byte(nullStr.String), t.fieldVal.Addr().Interface()); err != nil {
			return fmt.Errorf("cannot unmarshal JSON: %w", err)
		}
	}
	// NULL or empty -> leave as zero value
	return nil
}

func (t *scanTarget) copyPrimitive() error {
	switch v := t.scanPtr.(type) {
	case *sql.NullString:
		if v.Valid {
			t.fieldVal.SetString(v.String)
		}
	case *sql.NullInt64:
		if v.Valid {
			switch t.fieldVal.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				t.fieldVal.SetInt(v.Int64)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				t.fieldVal.SetUint(uint64(v.Int64))
			}
		}
	case *sql.NullFloat64:
		if v.Valid {
			t.fieldVal.SetFloat(v.Float64)
		}
	case *sql.NullBool:
		if v.Valid {
			t.fieldVal.SetBool(v.Bool)
		}
	}
	return nil
}

// buildScanTargets builds scan targets for all db-tagged fields
func buildScanTargets(val reflect.Value) []scanTarget {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	var targets []scanTarget
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Handle embedded structs recursively
		if field.Anonymous {
			targets = append(targets, buildScanTargets(val.Field(i))...)
			continue
		}

		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}

		_, conversionType := parseDBTag(tag)
		fieldVal := val.Field(i)

		targets = append(targets, scanTarget{
			fieldVal:       fieldVal,
			scanPtr:        createScanPtr(fieldVal, conversionType),
			conversionType: conversionType,
		})
	}

	return targets
}

// createScanPtr creates appropriate sql.Null* pointer for scanning
func createScanPtr(fieldVal reflect.Value, conversionType string) interface{} {
	// UUID and JSON both scan as NullString
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
		// Complex types scan directly
		return fieldVal.Addr().Interface()
	}
}
