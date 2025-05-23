package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// CallV1ExternalMediaGets sends a request to call-manager
// to getting a list of external media info.
// it returns detail list of external medias info if it succeed.
func (r *requestHandler) CallV1ExternalMediaGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cmexternalmedia.ExternalMedia, error) {
	uri := fmt.Sprintf("/v1/external-medias?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/calls", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cmexternalmedia.ExternalMedia
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CallV1ExternalMediaGet sends a request to call-manager
// to get the external media info.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ExternalMediaGet(ctx context.Context, externalMediaID uuid.UUID) (*cmexternalmedia.ExternalMedia, error) {
	uri := fmt.Sprintf("/v1/external-medias/%s", externalMediaID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/external-medias", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmexternalmedia.ExternalMedia
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ExternalMediaStart sends a request to call-manager
// to start the external media.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ExternalMediaStart(ctx context.Context, externalMediaID uuid.UUID, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, noInsertMedia bool, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*cmexternalmedia.ExternalMedia, error) {
	uri := "/v1/external-medias"

	reqData := &cmrequest.V1DataExternalMediasPost{
		ID:             externalMediaID,
		ReferenceType:  referenceType,
		ReferenceID:    referenceID,
		NoInsertMedia:  noInsertMedia,
		ExternalHost:   externalHost,
		Encapsulation:  encapsulation,
		Transport:      transport,
		ConnectionType: connectionType,
		Format:         format,
		Direction:      direction,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/external-medias", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmexternalmedia.ExternalMedia
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ExternalMediaStop sends a request to call-manager
// to stop the external media.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ExternalMediaStop(ctx context.Context, externalMediaID uuid.UUID) (*cmexternalmedia.ExternalMedia, error) {
	uri := fmt.Sprintf("/v1/external-medias/%s", externalMediaID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodDelete, "call/external-medias", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmexternalmedia.ExternalMedia
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
