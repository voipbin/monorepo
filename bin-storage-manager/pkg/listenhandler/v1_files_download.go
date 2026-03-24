package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// v1FilesIDDownloadURIRefresh handles /v1/files/<id>/download_uri_refresh POST request
func (h *listenHandler) v1FilesIDDownloadURIRefresh(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FilesIDDownloadURIRefresh",
		"request": m,
	})

	// "/v1/files/<uuid>/download_uri_refresh"
	tmpVals := strings.Split(m.URI, "/")
	fileID := uuid.FromStringOrNil(tmpVals[3])

	downloadURI, err := h.storageHandler.FileDownloadURIRefresh(ctx, fileID)
	if err != nil {
		log.Errorf("Could not refresh download URI. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(downloadURI)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
