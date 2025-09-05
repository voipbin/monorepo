package requesthandler

import (
	"context"
	"encoding/json"

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
	if err != nil {
		return nil, err
	}

	var res tmtts.TTS
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
