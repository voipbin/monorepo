package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	smfile "monorepo/bin-storage-manager/models/file"
	smrequest "monorepo/bin-storage-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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

	tmp, err := r.sendRequestStorage(ctx, uri, rabbitmqhandler.RequestMethodPost, "storage/files", requestTimeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res smfile.File
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
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

	_, err = r.sendRequestStorage(ctx, uri, rabbitmqhandler.RequestMethodPost, "storage/files", requestTimeoutDefault, delay, ContentTypeJSON, m)
	return err
}

// StorageV1FileGets sends a request to storage-manager
// to getting a list of files.
// it returns file list of flows if it succeed.
func (r *requestHandler) StorageV1FileGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]smfile.File, error) {
	uri := fmt.Sprintf("/v1/files?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestStorage(ctx, uri, rabbitmqhandler.RequestMethodGet, "storage/files", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []smfile.File
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// StorageV1FileGet sends a request to storage-manager
// to getting a file info.
// it returns file info if it succeed.
func (r *requestHandler) StorageV1FileGet(ctx context.Context, fileID uuid.UUID) (*smfile.File, error) {
	uri := fmt.Sprintf("/v1/files/%s", fileID)

	res, err := r.sendRequestStorage(ctx, uri, rabbitmqhandler.RequestMethodGet, "storage/files/<file-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data smfile.File
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// StorageV1FileDelete sends a request to storage-manager
// to deleting a files.
// it returns error if it fails
func (r *requestHandler) StorageV1FileDelete(ctx context.Context, fileID uuid.UUID, requestTimeout int) (*smfile.File, error) {
	uri := fmt.Sprintf("/v1/files/%s", fileID)

	res, err := r.sendRequestStorage(ctx, uri, rabbitmqhandler.RequestMethodDelete, "storage/files/<file-id>", requestTimeout, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data smfile.File
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return nil, err
	}

	return &data, nil
}
