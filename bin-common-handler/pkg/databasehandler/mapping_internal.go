package databasehandler

import (
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
