package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// TTSV1SpeakingCreate creates a speaking session.
func (r *requestHandler) TTSV1SpeakingCreate(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType tmstreaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	provider string,
	voiceID string,
	direction tmstreaming.Direction,
) (*tmspeaking.Speaking, error) {
	uri := "/v1/speakings"

	m, err := json.Marshal(request.V1DataSpeakingsPost{
		CustomerID:    customerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Language:      language,
		Provider:      provider,
		VoiceID:       voiceID,
		Direction:     direction,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodPost, "tts/speakings", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingGet gets a speaking session by ID.
func (r *requestHandler) TTSV1SpeakingGet(ctx context.Context, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s", speakingID)

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodGet, "tts/speakings/<speaking-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingGets lists speaking sessions.
func (r *requestHandler) TTSV1SpeakingGets(ctx context.Context, pageToken string, pageSize uint64, filters map[tmspeaking.Field]any) ([]*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings?page_token=%s&page_size=%d", pageToken, pageSize)

	if v, ok := filters[tmspeaking.FieldCustomerID]; ok {
		uri += fmt.Sprintf("&customer_id=%s", v)
	}
	if v, ok := filters[tmspeaking.FieldReferenceType]; ok {
		uri += fmt.Sprintf("&reference_type=%s", v)
	}
	if v, ok := filters[tmspeaking.FieldReferenceID]; ok {
		uri += fmt.Sprintf("&reference_id=%s", v)
	}
	if v, ok := filters[tmspeaking.FieldStatus]; ok {
		uri += fmt.Sprintf("&status=%s", v)
	}

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodGet, "tts/speakings", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []*tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// TTSV1SpeakingSay sends text to a speaking session. Pod-targeted.
func (r *requestHandler) TTSV1SpeakingSay(ctx context.Context, podID string, speakingID uuid.UUID, text string) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s/say", speakingID)

	m, err := json.Marshal(request.V1DataSpeakingsIDSayPost{
		Text: text,
	})
	if err != nil {
		return nil, err
	}

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/speakings/<speaking-id>/say", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingFlush flushes a speaking session. Pod-targeted.
func (r *requestHandler) TTSV1SpeakingFlush(ctx context.Context, podID string, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s/flush", speakingID)

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/speakings/<speaking-id>/flush", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingStop stops a speaking session. Pod-targeted.
func (r *requestHandler) TTSV1SpeakingStop(ctx context.Context, podID string, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s/stop", speakingID)

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/speakings/<speaking-id>/stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1SpeakingDelete soft-deletes a speaking session.
func (r *requestHandler) TTSV1SpeakingDelete(ctx context.Context, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	uri := fmt.Sprintf("/v1/speakings/%s", speakingID)

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodDelete, "tts/speakings/<speaking-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmspeaking.Speaking
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
