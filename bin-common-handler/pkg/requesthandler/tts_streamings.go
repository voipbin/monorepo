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
	activeflowID uuid.UUID,
	referenceType tmstreaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	gender tmstreaming.Gender,
	direction tmstreaming.Direction,
) (*tmstreaming.Streaming, error) {

	uri := "/v1/streamings"

	m, err := json.Marshal(request.V1DataStreamingsPost{
		CustomerID:    customerID,
		ActiveflowID:  activeflowID,
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
	if err != nil {
		return nil, err
	}

	var res tmstreaming.Streaming
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1StreamingDelete deletes a streaming tts by its ID.
func (r *requestHandler) TTSV1StreamingDelete(ctx context.Context, streamingID uuid.UUID) (*tmstreaming.Streaming, error) {
	uri := fmt.Sprintf("/v1/streamings/%s", streamingID)

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodDelete, "tts/streamings/<streaming-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res tmstreaming.Streaming
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1StreamingSayInit initializes the saying in a streaming tts session.
func (r *requestHandler) TTSV1StreamingSayInit(ctx context.Context, podID string, streamingID uuid.UUID, messageID uuid.UUID) (*tmstreaming.Streaming, error) {
	uri := fmt.Sprintf("/v1/streamings/%s/say_init", streamingID)

	m, err := json.Marshal(request.V1DataStreamingsIDSayInitPost{
		MessageID: messageID,
	})
	if err != nil {
		return nil, err
	}

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/streamings/<streaming-id>/say_init", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmstreaming.Streaming
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TTSV1StreamingSayAdd adds text to be said in a streaming tts session.
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
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// TTSV1StreamingSayStop stops the saying streaming tts.
func (r *requestHandler) TTSV1StreamingSayStop(ctx context.Context, podID string, streamingID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/streamings/%s/say_stop", streamingID)

	queueName := fmt.Sprintf("bin-manager.tts-manager.request.%s", podID)

	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "tts/streamings/<streaming-id>/say_stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
