package requesthandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
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

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodPost, "storage/compressfiles", requestTimeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res smcompressfile.CompressFile
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
