package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// TTSV1StreamingCreate create streaming tts.
func (r *requestHandler) TTSV1StreamingCreate(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType tmstreaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	gender tmstreaming.Gender,
	direction tmstreaming.Direction,
) (*tmstreaming.Streaming, error) {

	uri := "/v1/streamings"

	m, err := json.Marshal(request.V1DataStreamingsPost{
		CustomerID:    customerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Language:      language,
		Gender:        gender,
		Direction:     direction,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodPost, "tts/streamings", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmstreaming.Streaming
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TTSV1StreamingDelete deletes a streaming tts by its ID.
func (r *requestHandler) TTSV1StreamingDelete(ctx context.Context, streamingID uuid.UUID) (*tmstreaming.Streaming, error) {
	uri := fmt.Sprintf("/v1/streamings/%s", streamingID)

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodDelete, "tts/streamings/<streaming-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmstreaming.Streaming
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TTSV1StreamingSay say text in streaming tts.
func (r *requestHandler) TTSV1StreamingSay(ctx context.Context, podID string, streamingID uuid.UUID, messageID uuid.UUID, text string) error {
	uri := fmt.Sprintf("/v1/streamings/%s/say", streamingID)

	m, err := json.Marshal(request.V1DataStreamingsIDSayPost{
		MessageID: messageID,
		Text:      text,
	})
	if err != nil {
		return err
	}

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/streamings/<streaming-id>/say", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// TTSV1StreamingSay say text in streaming tts.
func (r *requestHandler) TTSV1StreamingSayAdd(ctx context.Context, podID string, streamingID uuid.UUID, messageID uuid.UUID, text string) error {
	uri := fmt.Sprintf("/v1/streamings/%s/say_add", streamingID)

	m, err := json.Marshal(request.V1DataStreamingsIDSayAddPost{
		MessageID: messageID,
		Text:      text,
	})
	if err != nil {
		return err
	}

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/streamings/<streaming-id>/say_add", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// TTSV1StreamingSayStop stops the saying streaming tts.
func (r *requestHandler) TTSV1StreamingSayStop(ctx context.Context, podID string, streamingID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/streamings/%s/say_stop", streamingID)

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/streamings/<streaming-id>/say_stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
