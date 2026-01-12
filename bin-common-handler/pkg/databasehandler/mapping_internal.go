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
