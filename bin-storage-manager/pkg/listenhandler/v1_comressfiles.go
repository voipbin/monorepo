package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-storage-manager/pkg/listenhandler/models/request"

	"github.com/sirupsen/logrus"
)

// v1CompressfilesPost handles /v1/compressfiles POST request
// creates a new compress with given data and return the created compress info.
func (h *listenHandler) v1CompressfilesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CompressfilesPost",
		"request": m,
	})

	var req request.V1DataCompressfilesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create compress
	tmp, err := h.storageHandler.CompressfileCreate(ctx, req.ReferenceIDs, req.FileIDs)
	if err != nil {
		log.Errorf("Could not create a new file. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
