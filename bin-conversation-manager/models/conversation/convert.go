package conversation

import (
	"github.com/sirupsen/logrus"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
)

// ConvertStringMapToFieldMap converts a map with string keys to a map with conversation.Field keys,
// using reflection-based type conversion from bin-common-handler.
func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ConvertStringMapToFieldMap",
		"source": src,
	})
	log.Debug("Converting string map to field map - BEFORE conversion")

	// Use reflection-based converter with Conversation struct
	typed, err := commondatabasehandler.ConvertMapToTypedMap(src, Conversation{})
	if err != nil {
		log.Errorf("UUID conversion failed. err: %v", err)
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[Field]any, len(typed))
	for k, v := range typed {
		result[Field(k)] = v
	}

	log.WithFields(logrus.Fields{
		"result": result,
	}).Debug("Converting string map to field map - AFTER conversion (check UUID types)")

	return result, nil
}
