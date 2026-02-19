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
	"github.com/pkg/errors"
)

// CallV1ExternalMediaList sends a request to call-manager
// to getting a list of external media info.
// it returns detail list of external medias info if it succeed.
func (r *requestHandler) CallV1ExternalMediaList(ctx context.Context, pageToken string, pageSize uint64, filters map[cmexternalmedia.Field]any) ([]cmexternalmedia.ExternalMedia, error) {
	uri := fmt.Sprintf("/v1/external-medias?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/calls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cmexternalmedia.ExternalMedia
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// CallV1ExternalMediaGet sends a request to call-manager
// to get the external media info.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ExternalMediaGet(ctx context.Context, externalMediaID uuid.UUID) (*cmexternalmedia.ExternalMedia, error) {
	uri := fmt.Sprintf("/v1/external-medias/%s", externalMediaID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/external-medias", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmexternalmedia.ExternalMedia
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1ExternalMediaStart sends a request to call-manager
// to start the external media.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ExternalMediaStart(
	ctx context.Context,
	externalMediaID uuid.UUID,
	typ cmexternalmedia.Type,
	referenceType cmexternalmedia.ReferenceType,
	referenceID uuid.UUID,
	externalHost string,
	encapsulation string,
	transport string,
	connectionType string,
	format string,
	directionListen cmexternalmedia.Direction,
	directionSpeak cmexternalmedia.Direction,
) (*cmexternalmedia.ExternalMedia, error) {
	uri := "/v1/external-medias"

	reqData := &cmrequest.V1DataExternalMediasPost{
		ID:            externalMediaID,
		Type:          typ,
		ReferenceType: referenceType,
		ReferenceID:     referenceID,
		ExternalHost:    externalHost,
		Encapsulation:   encapsulation,
		Transport:       transport,
		ConnectionType:  connectionType,
		Format:          format,
		DirectionListen: directionListen,
		DirectionSpeak:  directionSpeak,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/external-medias", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmexternalmedia.ExternalMedia
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1ExternalMediaStop sends a request to call-manager
// to stop the external media.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ExternalMediaStop(ctx context.Context, externalMediaID uuid.UUID) (*cmexternalmedia.ExternalMedia, error) {
	uri := fmt.Sprintf("/v1/external-medias/%s", externalMediaID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodDelete, "call/external-medias", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmexternalmedia.ExternalMedia
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
