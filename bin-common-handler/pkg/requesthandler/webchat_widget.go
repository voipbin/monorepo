package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"

	wcwidget "monorepo/bin-webchat-manager/models/widget"
	wcrequest "monorepo/bin-webchat-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// WebchatV1WidgetCreate sends a request to webchat-manager to create a widget.
func (r *requestHandler) WebchatV1WidgetCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	sessionFlowID uuid.UUID,
	messageFlowID uuid.UUID,
	sessionIdleTimeout int,
	themeConfig *wcwidget.ThemeConfig,
) (*wcwidget.Widget, error) {
	uri := "/v1/widgets"

	data := &wcrequest.V1DataWidgetsPost{
		CustomerID:         customerID,
		Name:                name,
		SessionFlowID:       sessionFlowID,
		MessageFlowID:       messageFlowID,
		SessionIdleTimeout:  sessionIdleTimeout,
		ThemeConfig:         themeConfig,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodPost, "webchat/widgets", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res wcwidget.Widget
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1WidgetGet sends a request to webchat-manager to get the widget.
func (r *requestHandler) WebchatV1WidgetGet(ctx context.Context, id uuid.UUID) (*wcwidget.Widget, error) {
	uri := fmt.Sprintf("/v1/widgets/%s", id)

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodGet, "webchat/widgets/<widget-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res wcwidget.Widget
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1WidgetList sends a request to webchat-manager to get a list of widgets.
func (r *requestHandler) WebchatV1WidgetList(ctx context.Context, pageToken string, pageSize uint64, filters map[wcwidget.Field]any) ([]*wcwidget.Widget, error) {
	uri := fmt.Sprintf("/v1/widgets?page_token=%s&page_size=%d", pageToken, pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodGet, "webchat/widgets", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*wcwidget.Widget
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// WebchatV1WidgetUpdate sends a request to webchat-manager to update the widget's basic info.
func (r *requestHandler) WebchatV1WidgetUpdate(
	ctx context.Context,
	id uuid.UUID,
	name string,
	sessionFlowID uuid.UUID,
	messageFlowID uuid.UUID,
	sessionIdleTimeout int,
	themeConfig *wcwidget.ThemeConfig,
) (*wcwidget.Widget, error) {
	uri := fmt.Sprintf("/v1/widgets/%s", id)

	data := &wcrequest.V1DataWidgetsIDPut{
		Name:               name,
		SessionFlowID:      sessionFlowID,
		MessageFlowID:      messageFlowID,
		SessionIdleTimeout: sessionIdleTimeout,
		ThemeConfig:        themeConfig,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodPut, "webchat/widgets/<widget-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res wcwidget.Widget
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1WidgetDelete sends a request to webchat-manager to delete the widget.
func (r *requestHandler) WebchatV1WidgetDelete(ctx context.Context, id uuid.UUID) (*wcwidget.Widget, error) {
	uri := fmt.Sprintf("/v1/widgets/%s", id)

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodDelete, "webchat/widgets/<widget-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res wcwidget.Widget
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1WidgetDirectHashRegenerate sends a request to webchat-manager to regenerate the widget's direct hash.
func (r *requestHandler) WebchatV1WidgetDirectHashRegenerate(ctx context.Context, id uuid.UUID) (*wcwidget.Widget, error) {
	uri := fmt.Sprintf("/v1/widgets/%s/direct-hash-regenerate", id)

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodPost, "webchat/widgets/<widget-id>/direct-hash-regenerate", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res wcwidget.Widget
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
