package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	tmtts "monorepo/bin-tts-manager/models/tts"
	"monorepo/bin-tts-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// TTSV1SpeecheCreate create speech-to-text.
func (r *requestHandler) TTSV1SpeecheCreate(ctx context.Context, callID uuid.UUID, text string, gender tmtts.Gender, language string, timeout int) (*tmtts.TTS, error) {

	uri := "/v1/speeches"

	m, err := json.Marshal(request.V1DataSpeechesPost{
		CallID:   callID,
		Text:     text,
		Gender:   gender,
		Language: language,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTTS(ctx, uri, sock.RequestMethodPost, "tts/speeches", timeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtts.TTS
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
