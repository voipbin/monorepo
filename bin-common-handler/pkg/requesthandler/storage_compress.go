package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	smcompressfile "monorepo/bin-storage-manager/models/compressfile"
	smrequest "monorepo/bin-storage-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// StorageV1CompressfileCreate sends a request to storage-manager
// to creating an compressfile.
// it returns created compressfile if it succeed.
// requestTimeout: milliseconds
func (r *requestHandler) StorageV1CompressfileCreate(ctx context.Context, referenceIDs []uuid.UUID, fileIDs []uuid.UUID, requestTimeout int) (*smcompressfile.CompressFile, error) {
	uri := "/v1/compressfiles"

	data := &smrequest.V1DataCompressfilesPost{
		ReferenceIDs: referenceIDs,
		FileIDs:      fileIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestStorage(ctx, uri, rabbitmqhandler.RequestMethodPost, "storage/compressfiles", requestTimeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res smcompressfile.CompressFile
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
