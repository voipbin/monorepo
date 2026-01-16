package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	amai "monorepo/bin-ai-manager/models/ai"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIV1AIList sends a request to ai-manager
// to getting a list of ais info.
// it returns detail list of ais info if it succeed.
func (r *requestHandler) AIV1AIList(ctx context.Context, pageToken string, pageSize uint64, filters map[amai.Field]any) ([]amai.AI, error) {
	uri := fmt.Sprintf("/v1/ais?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []amai.AI
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AIGet returns the ai.
func (r *requestHandler) AIV1AIGet(ctx context.Context, aiID uuid.UUID) (*amai.AI, error) {

	uri := fmt.Sprintf("/v1/ais/%s", aiID.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais/<ai-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amai.AI
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AICreate sends a request to ai-manager
// to creating a ai.
// it returns created ai if it succeed.
func (r *requestHandler) AIV1AICreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType amai.EngineType,
	engineModel amai.EngineModel,
	engineData map[string]any,
	engineKey string,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
) (*amai.AI, error) {
	uri := "/v1/ais"

	data := &amrequest.V1DataAIsPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,

		EngineType:  engineType,
		EngineModel: engineModel,
		EngineData:  engineData,
		EngineKey:   engineKey,

		InitPrompt: initPrompt,

		TTSType:    ttsType,
		TTSVoiceID: ttsVoiceID,

		STTType: sttType,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/ais", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amai.AI
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIDelete sends a request to ai-manager
// to deleting a ai.
// it returns deleted ai if it succeed.
func (r *requestHandler) AIV1AIDelete(ctx context.Context, aiID uuid.UUID) (*amai.AI, error) {
	uri := fmt.Sprintf("/v1/ais/%s", aiID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/ais/<ai-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amai.AI
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIUpdate sends a request to ai-manager
// to updating an AI.
// it returns updated AI if it succeed.
func (r *requestHandler) AIV1AIUpdate(
	ctx context.Context,
	aiID uuid.UUID,
	name string,
	detail string,
	engineType amai.EngineType,
	engineModel amai.EngineModel,
	engineData map[string]any,
	engineKey string,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
) (*amai.AI, error) {
	uri := fmt.Sprintf("/v1/ais/%s", aiID)

	data := &amrequest.V1DataAIsIDPut{
		Name:   name,
		Detail: detail,

		EngineType:  engineType,
		EngineModel: engineModel,
		EngineData:  engineData,
		EngineKey:   engineKey,

		InitPrompt: initPrompt,

		TTSType:    ttsType,
		TTSVoiceID: ttsVoiceID,

		STTType: sttType,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPut, "ai/ais/<ai-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amai.AI
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
