package messagechat

import (
	"github.com/sirupsen/logrus"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
)

func ConvertStringMapToFieldMap(src map[string]any) (map[Field]any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ConvertStringMapToFieldMap",
		"source": src,
	})
	log.Debug("Converting string map to field map - BEFORE conversion")

	typed, err := commondatabasehandler.ConvertMapToTypedMap(src, Messagechat{})
	if err != nil {
		log.Errorf("UUID conversion failed. err: %v", err)
		return nil, err
	}

	result := make(map[Field]any, len(typed))
	for k, v := range typed {
		result[Field(k)] = v
	}

	log.WithFields(logrus.Fields{
		"result": result,
	}).Debug("Converting string map to field map - AFTER conversion (check UUID types)")

	return result, nil
}
