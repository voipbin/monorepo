package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-storage-manager/pkg/listenhandler/models/request"

	"github.com/pkg/errors"
)

// v1CompressfilesPost handles /v1/compressfiles POST request
// creates a new compress with given data and return the created compress info.
func (h *listenHandler) v1CompressfilesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	var req request.V1DataCompressfilesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal the request.")
	}

	// create compress
	tmp, err := h.storageHandler.CompressfileCreate(ctx, req.ReferenceIDs, req.FileIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create compress file. request: %v", req)
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the response. res: %v", tmp)
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
