package campaign

import (
	"github.com/sirupsen/logrus"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
)

// ConvertStringMapToFieldMap converts a map[string]any to map[Field]any
// This function also converts string UUIDs to uuid.UUID types using the Campaign model's field tags
func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ConvertStringMapToFieldMap",
		"source": src,
	})
	log.Debug("Converting string map to field map - BEFORE conversion")

	// Use commondatabasehandler.ConvertMapToTypedMap to convert string UUIDs to uuid.UUID
	typed, err := commondatabasehandler.ConvertMapToTypedMap(src, Campaign{})
	if err != nil {
		log.Errorf("UUID conversion failed. err: %v", err)
		return nil, err
	}

	// Convert map[string]any to map[Field]any
	result := make(map[Field]any, len(typed))
	for k, v := range typed {
		result[Field(k)] = v
	}

	log.WithFields(logrus.Fields{
		"result": result,
	}).Debug("Converting string map to field map - AFTER conversion (check UUID types)")

	return result, nil
}
