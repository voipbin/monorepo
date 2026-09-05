package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/sirupsen/logrus"
)

// processEventTMTranscriptCreated handles transcribe-manager's
// transcript_created event.
//
// It does as little as possible on purpose. This fires for EVERY final STT
// result on the platform -- flow-driven, summary-driven, customer-started -- and
// processEventRun spawns a goroutine per event, so anything expensive here is
// multiplied by total platform transcription volume, not by the number of calls
// actually being listened to. Unmarshal, hand off, return.
//
// The ownership filter (one Redis SMEMBERS) lives in
// aicallHandler.EventTMTranscriptCreated, which is where the listen state is.
func (h *subscribeHandler) processEventTMTranscriptCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventTMTranscriptCreated",
		"event": m,
	})

	var evt tmtranscript.Transcript
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.aicallHandler.EventTMTranscriptCreated(ctx, &evt)

	return nil
}
