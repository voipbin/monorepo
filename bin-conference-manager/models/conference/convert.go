package conference

import (
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with conference.Field keys,
// using reflection-based type conversion from bin-common-handler.
// Accepts map[string]string for URL query parameter compatibility.
func ConvertStringMapToFieldMap(src map[string]string) (map[Field]any, error) {
	// Convert map[string]string to map[string]any for reflection converter
	srcAny := make(map[string]any, len(src))
	for k, v := range src {
		srcAny[k] = v
	}

	// Use reflection-based converter with Conference struct
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, Conference{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[Field]any, len(typed))
	for k, v := range typed {
		result[Field(k)] = v
	}

	return result, nil
}
