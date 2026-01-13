package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	smfile "monorepo/bin-storage-manager/models/file"
	smrequest "monorepo/bin-storage-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// StorageV1FileCreate sends a request to storage-manager
// to creating a file.
// it returns created file if it succeed.
func (r *requestHandler) StorageV1FileCreate(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType smfile.ReferenceType,
	referenceID uuid.UUID,
	name string,
	detail string,
	filename string,
	bucketName string,
	filepath string,
	requestTimeout int, // milliseconds
) (*smfile.File, error) {
	uri := "/v1/files"

	data := &smrequest.V1DataFilesPost{
		CustomerID:    customerID,
		OwnerID:       ownerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Name:          name,
		Detail:        detail,
		Filename:      filename,
		BucketName:    bucketName,
		Filepath:      filepath,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodPost, "storage/files", requestTimeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res smfile.File
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// StorageV1FileCreateWithDelay sends a request to storage-manager
// to creating a file with the given delay.
// the request will be delievered after delay time.
// it returns created file if it succeed.
func (r *requestHandler) StorageV1FileCreateWithDelay(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType smfile.ReferenceType,
	referenceID uuid.UUID,
	name string,
	detail string,
	filename string,
	bucketName string,
	filepath string,
	delay int, // milliseconds
) error {
	uri := "/v1/files"

	data := &smrequest.V1DataFilesPost{
		CustomerID:    customerID,
		OwnerID:       ownerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Name:          name,
		Detail:        detail,
		Filename:      filename,
		BucketName:    bucketName,
		Filepath:      filepath,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodPost, "storage/files", requestTimeoutDefault, delay, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// StorageV1FileGets sends a request to storage-manager
// to getting a list of files.
// it returns file list of flows if it succeed.
func (r *requestHandler) StorageV1FileGets(ctx context.Context, pageToken string, pageSize uint64, filters map[smfile.Field]any) ([]smfile.File, error) {
	uri := fmt.Sprintf("/v1/files?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodGet, "storage/files", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []smfile.File
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// StorageV1FileGet sends a request to storage-manager
// to getting a file info.
// it returns file info if it succeed.
func (r *requestHandler) StorageV1FileGet(ctx context.Context, fileID uuid.UUID) (*smfile.File, error) {
	uri := fmt.Sprintf("/v1/files/%s", fileID)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodGet, "storage/files/<file-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res smfile.File
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// StorageV1FileDelete sends a request to storage-manager
// to deleting a files.
// it returns error if it fails
func (r *requestHandler) StorageV1FileDelete(ctx context.Context, fileID uuid.UUID, requestTimeout int) (*smfile.File, error) {
	uri := fmt.Sprintf("/v1/files/%s", fileID)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodDelete, "storage/files/<file-id>", requestTimeout, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res smfile.File
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
