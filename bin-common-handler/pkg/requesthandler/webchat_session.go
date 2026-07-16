package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"

	wcsession "monorepo/bin-webchat-manager/models/session"
	wcrequest "monorepo/bin-webchat-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// WebchatV1SessionCreate sends a request to webchat-manager to create a session.
func (r *requestHandler) WebchatV1SessionCreate(ctx context.Context, customerID uuid.UUID, widgetID uuid.UUID) (*wcsession.Session, error) {
	uri := "/v1/sessions"

	data := &wcrequest.V1DataSessionsPost{
		CustomerID: customerID,
		WidgetID:   widgetID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodPost, "webchat/sessions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res wcsession.Session
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1SessionGet sends a request to webchat-manager to get the session.
func (r *requestHandler) WebchatV1SessionGet(ctx context.Context, id uuid.UUID) (*wcsession.Session, error) {
	uri := fmt.Sprintf("/v1/sessions/%s", id)

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodGet, "webchat/sessions/<session-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res wcsession.Session
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1SessionList sends a request to webchat-manager to get a list of sessions.
func (r *requestHandler) WebchatV1SessionList(ctx context.Context, pageToken string, pageSize uint64, filters map[wcsession.Field]any) ([]*wcsession.Session, error) {
	uri := fmt.Sprintf("/v1/sessions?page_token=%s&page_size=%d", pageToken, pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodGet, "webchat/sessions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*wcsession.Session
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// WebchatV1SessionDelete sends a request to webchat-manager to delete the session.
func (r *requestHandler) WebchatV1SessionDelete(ctx context.Context, id uuid.UUID) (*wcsession.Session, error) {
	uri := fmt.Sprintf("/v1/sessions/%s", id)

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodDelete, "webchat/sessions/<session-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res wcsession.Session
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1SessionEnd sends a request to webchat-manager to end the session.
func (r *requestHandler) WebchatV1SessionEnd(ctx context.Context, id uuid.UUID) (*wcsession.Session, error) {
	uri := fmt.Sprintf("/v1/sessions/%s/end", id)

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodPost, "webchat/sessions/<session-id>/end", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res wcsession.Session
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
