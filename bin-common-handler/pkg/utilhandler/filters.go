package utilhandler

import (
	"encoding/json"
	"reflect"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ParseFiltersFromRequestBody unmarshals filters from request body JSON
// Returns map[string]any to preserve type information (bool, numbers, UUIDs, etc.)
func ParseFiltersFromRequestBody(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}

	var filters map[string]any
	if err := json.Unmarshal(data, &filters); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal filters from request body")
	}

	return filters, nil
}

// ConvertFilters converts map[string]any filters to typed Field map
// by validating against FieldStruct definition and using struct field types for conversion
//
// FS: FieldStruct type that defines allowed filters with `filter:` tags (e.g., ai.FieldStruct)
// F: Field type for map keys (e.g., ai.Field)
// fieldStruct: Instance of FieldStruct (can be zero value)
// filters: Raw filters from request body
//
// Example usage:
//
//	type FieldStruct struct {
//	    CustomerID uuid.UUID `filter:"customer_id"`
//	    Deleted    bool      `filter:"deleted"`
//	}
//	filters, err := ConvertFilters[FieldStruct, Field](FieldStruct{}, rawFilters)
func ConvertFilters[FS any, F ~string](fieldStruct FS, filters map[string]any) (map[F]any, error) {
	result := make(map[F]any)

	// Get FieldStruct type
	typ := reflect.TypeOf(fieldStruct)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Loop through all fields in FieldStruct
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Get filter key from tag
		filterKey := field.Tag.Get("filter")
		if filterKey == "" || filterKey == "-" {
			continue
		}

		// Check if this filter exists in received data
		filterValue, exists := filters[filterKey]
		if !exists {
			continue
		}

		// Convert value based on FieldStruct field type
		converted, err := convertValueToType(filterValue, field.Type)
		if err != nil {
			return nil, errors.Wrapf(err, "could not convert filter %s", filterKey)
		}

		// Add to result with Field key
		result[F(filterKey)] = converted
	}

	return result, nil
}

// convertValueToType converts filter value to match struct field type
func convertValueToType(value any, targetType reflect.Type) (any, error) {
	// Handle nil
	if value == nil {
		return nil, nil
	}

	// If types already match, return as-is
	valueType := reflect.TypeOf(value)
	if valueType == targetType {
		return value, nil
	}

	// Handle uuid.UUID conversion
	if targetType.String() == "uuid.UUID" {
		if str, ok := value.(string); ok {
			id, err := uuid.FromString(str)
			if err != nil {
				return nil, errors.Wrap(err, "invalid UUID format")
			}
			return id, nil
		}
	}

	// Handle bool
	if targetType.Kind() == reflect.Bool {
		if b, ok := value.(bool); ok {
			return b, nil
		}
	}

	// Handle integers (JSON unmarshals numbers as float64)
	if targetType.Kind() >= reflect.Int && targetType.Kind() <= reflect.Uint64 {
		if f, ok := value.(float64); ok {
			return int64(f), nil
		}
	}

	// Handle string
	if targetType.Kind() == reflect.String {
		if str, ok := value.(string); ok {
			return str, nil
		}
	}

	// Default: return as-is
	return value, nil
}
