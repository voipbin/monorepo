package file

import (
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with file.Field keys,
// using reflection-based type conversion from bin-common-handler.
func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	// Use reflection-based converter with File struct
	typed, err := commondatabasehandler.ConvertMapToTypedMap(src, File{})
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
