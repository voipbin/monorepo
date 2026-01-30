package databasehandler

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/gofrs/uuid"
)

// Platform-independent int bounds for overflow checking
const (
	maxInt = int(^uint(0) >> 1)
	minInt = -maxInt - 1
)

var (
	// fieldTypeCache caches reflection results for performance
	// map[reflect.Type]map[string]reflect.Type (struct type -> field name -> field type)
	fieldTypeCache sync.Map
)

// ConvertMapToTypedMap converts a map[string]any to typed values based on a model struct's field types.
// It uses reflection to automatically determine the correct type for each field based on the struct's db tags.
//
// Example usage:
//   typed, err := ConvertMapToTypedMap(filters, agent.Agent{})
//
// The function handles:
// - uuid.UUID fields (converts string to UUID)
// - bool fields (converts string "true"/"false" to bool)
// - string fields and custom string types (enums)
// - int fields (converts float64 from JSON to int)
// - Special "deleted" filter field (always converted to bool)
//
// Performance: Uses caching to avoid repeated reflection on the same struct type.
func ConvertMapToTypedMap(src map[string]any, modelStruct any) (map[string]any, error) {
	result := make(map[string]any)

	// Get struct type info
	structType := reflect.TypeOf(modelStruct)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	// Try to get cached field type map
	var fieldTypeMap map[string]reflect.Type
	if cached, ok := fieldTypeCache.Load(structType); ok {
		fieldTypeMap = cached.(map[string]reflect.Type)
	} else {
		// Build and cache field type map (only happens once per struct type)
		fieldTypeMap = make(map[string]reflect.Type)
		buildFieldTypeMap(structType, fieldTypeMap)
		fieldTypeCache.Store(structType, fieldTypeMap)
	}

	// Convert each source value based on target field type
	for key, val := range src {
		// Special handling for "deleted" filter (not in struct, filter-only field)
		if key == "deleted" {
			converted, err := convertToBool(val)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", key, err)
			}
			result[key] = converted
			continue
		}

		targetType, exists := fieldTypeMap[key]
		if !exists {
			// Unknown field - pass through as-is for flexibility
			result[key] = val
			continue
		}

		converted, err := convertValue(val, targetType)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", key, err)
		}
		result[key] = converted
	}

	return result, nil
}

// convertValue converts a value to the target type using reflection
func convertValue(val any, targetType reflect.Type) (any, error) {
	// If already correct type, return as-is
	valType := reflect.TypeOf(val)
	if valType == targetType {
		return val, nil
	}

	// Handle uuid.UUID
	if targetType == reflect.TypeOf(uuid.UUID{}) {
		return convertToUUID(val)
	}

	// Handle bool
	if targetType.Kind() == reflect.Bool {
		return convertToBool(val)
	}

	// Handle string and custom string types (enums like agent.Status, queue.RoutingMethod)
	if targetType.Kind() == reflect.String {
		return convertToString(val)
	}

	// Handle int types
	if targetType.Kind() == reflect.Int {
		return convertToInt(val)
	}

	// Handle float64 (JSON numbers)
	if targetType.Kind() == reflect.Float64 {
		return convertToFloat64(val)
	}

	return nil, fmt.Errorf("unsupported type conversion from %T to %v", val, targetType)
}

// convertToUUID converts various types to uuid.UUID
func convertToUUID(val any) (uuid.UUID, error) {
	switch v := val.(type) {
	case string:
		// Empty string is allowed for nullable UUID fields
		if v == "" {
			return uuid.Nil, nil
		}
		id, err := uuid.FromString(v)
		if err != nil {
			// Return error for invalid non-empty UUID strings to detect data corruption
			return uuid.Nil, fmt.Errorf("invalid UUID string %q: %w", v, err)
		}
		return id, nil
	case uuid.UUID:
		return v, nil
	default:
		return uuid.Nil, fmt.Errorf("cannot convert %T to uuid.UUID", val)
	}
}

// convertToBool converts various types to bool
func convertToBool(val any) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		// String "true" becomes true, anything else (including "false") becomes false
		return v == "true", nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", val)
	}
}

// convertToString converts various types to string
func convertToString(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		return "", fmt.Errorf("cannot convert %T to string", val)
	}
}

// convertToInt converts various types to int
func convertToInt(val any) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case float64:
		// JSON numbers come as float64
		// Check for overflow on 32-bit systems
		if v > float64(maxInt) || v < float64(minInt) {
			return 0, fmt.Errorf("float64 value %v overflows int", v)
		}
		return int(v), nil
	case int64:
		// Check for overflow on 32-bit systems
		if v > int64(maxInt) || v < int64(minInt) {
			return 0, fmt.Errorf("int64 value %v overflows int", v)
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", val)
	}
}

// convertToFloat64 converts various types to float64
func convertToFloat64(val any) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

// buildFieldTypeMap recursively builds field name to type mapping including embedded struct fields
func buildFieldTypeMap(structType reflect.Type, fieldTypeMap map[string]reflect.Type) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Check if field has a db tag
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			// Extract field name from db tag (before comma if present)
			// Example: "customer_id,uuid" -> "customer_id"
			fieldName := dbTag
			if commaIdx := strings.IndexByte(dbTag, ','); commaIdx != -1 {
				fieldName = dbTag[:commaIdx]
			}
			fieldTypeMap[fieldName] = field.Type
		} else if field.Anonymous {
			// Recursively process embedded struct fields
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct {
				buildFieldTypeMap(fieldType, fieldTypeMap)
			}
		}
	}
}
