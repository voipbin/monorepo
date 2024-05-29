package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-storage-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// v1FilesPost handles /v1/files POST request
// creates a new file with given data and return the created file info.
func (h *listenHandler) v1FilesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FilesPost",
		"request": m,
	})

	var req request.V1DataFilesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create file
	tmp, err := h.storageHandler.FileCreate(
		ctx,
		req.CustomerID,
		req.OwnerID,
		req.ReferenceType,
		req.ReferenceID,
		req.Name,
		req.Detail,
		req.Filename,
		req.BucketName,
		req.Filepath,
	)
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

// v1FilesGet handles /v1/files GET request
func (h *listenHandler) v1FilesGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FilesGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// parse the filters
	filters := h.utilHandler.URLParseFilters(u)

	// gets the list of files
	tmp, err := h.storageHandler.FileGets(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get files. err: %v", err)
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

// v1FilesIDGet handles /v1/files/<id> GET request
// creates a new tts audio for the given text and upload the file to the bucket. Returns uploaded filename with path.
func (h *listenHandler) v1FilesIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FilesIDGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	fileID := uuid.FromStringOrNil(tmpVals[3])

	// get recording
	rec, err := h.storageHandler.FileGet(ctx, fileID)
	if err != nil {
		log.Errorf("Could not get file info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(rec)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1FilesIDDelete handles
// /v1/files/{id} DELETE
func (h *listenHandler) v1FilesIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FilesIDDelete",
		"request": m,
	})

	// "/v1/files/be2692f8-066a-11eb-847f-1b4de696fafb"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.storageHandler.FileDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete file. err: %v", err)
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
