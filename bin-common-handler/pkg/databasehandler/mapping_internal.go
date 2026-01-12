package databasehandler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

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
			// UUID already handled above, all other structs get JSON marshaled
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("field %s: JSON marshal failed: %w", key, err)
			}
			result[key] = jsonBytes

		default:
			// Primitives pass through
			result[key] = value
		}
	}

	return result, nil
}

// fieldScanTarget represents a field's scan metadata
type fieldScanTarget struct {
	fieldVal       *reflect.Value
	scanTarget     interface{}
	conversionType string // Conversion type from db tag (e.g., "uuid", "json")
}

// buildScanTargetsRecursive recursively builds scan targets for embedded structs
func buildScanTargetsRecursive(val reflect.Value) []fieldScanTarget {
	// Handle pointer dereferencing
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return []fieldScanTarget{}
	}

	typ := val.Type()
	scanTargets := []fieldScanTarget{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Handle embedded structs recursively
		if field.Anonymous {
			embeddedVal := val.Field(i)
			embeddedTargets := buildScanTargetsRecursive(embeddedVal)
			scanTargets = append(scanTargets, embeddedTargets...)
			continue
		}

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

		// Create appropriate sql.Null* type based on field type and conversion
		scanTarget := createNullScanTarget(fieldVal, conversionType)

		// Store both the field reference and scan target
		fieldValCopy := fieldVal
		scanTargets = append(scanTargets, fieldScanTarget{
			fieldVal:       &fieldValCopy,
			scanTarget:     scanTarget,
			conversionType: conversionType,
		})
	}

	return scanTargets
}

// createNullScanTarget creates appropriate sql.Null* type for a field
func createNullScanTarget(fieldVal reflect.Value, conversionType string) interface{} {
	// Handle special conversion types
	if conversionType == "uuid" {
		// UUID stored as bytes in database, scan as NullString to handle NULL
		return new(sql.NullString)
	}
	if conversionType == "json" {
		// JSON stored as string in database
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
		// For complex types (JSON, etc.), scan directly
		return fieldVal.Addr().Interface()
	}
}

// copyFromNullType copies value from sql.Null* type to field if valid
func copyFromNullType(scanTarget interface{}, fieldVal *reflect.Value, conversionType string) error {
	// Handle special conversion types
	if conversionType == "uuid" {
		nullStr := scanTarget.(*sql.NullString)
		if nullStr.Valid && len(nullStr.String) > 0 {
			// Convert bytes (stored as string) to UUID
			uuidVal, err := uuid.FromBytes([]byte(nullStr.String))
			if err != nil {
				return fmt.Errorf("cannot convert bytes to UUID: %w", err)
			}
			fieldVal.Set(reflect.ValueOf(uuidVal))
		} else {
			// NULL or empty -> uuid.Nil
			fieldVal.Set(reflect.ValueOf(uuid.Nil))
		}
		return nil
	}
	if conversionType == "json" {
		nullStr := scanTarget.(*sql.NullString)
		if nullStr.Valid && len(nullStr.String) > 0 {
			// Unmarshal JSON string into field
			if err := json.Unmarshal([]byte(nullStr.String), fieldVal.Addr().Interface()); err != nil {
				return fmt.Errorf("cannot unmarshal JSON: %w", err)
			}
		}
		// else: NULL or empty -> leave as zero value (empty slice/struct)
		return nil
	}

	switch v := scanTarget.(type) {
	case *sql.NullString:
		if v.Valid {
			fieldVal.SetString(v.String)
		}
		// If NULL, leave field as zero value (empty string)
	case *sql.NullInt64:
		if v.Valid {
			// Convert to appropriate int type
			switch fieldVal.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldVal.SetInt(v.Int64)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fieldVal.SetUint(uint64(v.Int64))
			}
		}
		// If NULL, leave field as zero value (0)
	case *sql.NullFloat64:
		if v.Valid {
			fieldVal.SetFloat(v.Float64)
		}
		// If NULL, leave field as zero value (0.0)
	case *sql.NullBool:
		if v.Valid {
			fieldVal.SetBool(v.Bool)
		}
		// If NULL, leave field as zero value (false)
	default:
		// For complex types, the value was scanned directly - no copy needed
	}
	return nil
}
