package streaminghandler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// sttResult is a provider-agnostic STT result.
type sttResult struct {
	isFinal bool
	message string
}

// resultProcessor handles VAD state, webhook publishing, and transcript creation.
type resultProcessor struct {
	st                *streaming.Streaming
	notifyHandler     notifyhandler.NotifyHandler
	transcriptHandler transcripthandler.TranscriptHandler

	speaking bool
	t1       time.Time
}

// newResultProcessor creates a resultProcessor for the given streaming session.
func newResultProcessor(st *streaming.Streaming, nh notifyhandler.NotifyHandler, th transcripthandler.TranscriptHandler) *resultProcessor {
	return &resultProcessor{
		st:                st,
		notifyHandler:     nh,
		transcriptHandler: th,
		t1:                time.Now(),
	}
}

// process handles a single normalized STT result.
func (rp *resultProcessor) process(ctx context.Context, r sttResult) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "resultProcessor.process",
		"streaming_id":  rp.st.ID,
		"transcribe_id": rp.st.TranscribeID,
	})

	if !r.isFinal {
		// interim result — publish VAD events
		if !rp.speaking {
			rp.speaking = true
			now := time.Now()
			evt := rp.st.NewSpeech("", &now)
			rp.notifyHandler.PublishWebhookEvent(ctx, rp.st.CustomerID, streaming.EventTypeSpeechStarted, evt)
			log.Debugf("Published speech_started. transcribe_id: %s, direction: %s", rp.st.TranscribeID, rp.st.Direction)
		}

		now := time.Now()
		evt := rp.st.NewSpeech(r.message, &now)
		rp.notifyHandler.PublishWebhookEvent(ctx, rp.st.CustomerID, streaming.EventTypeSpeechInterim, evt)
		log.Debugf("Published speech_interim. transcribe_id: %s, direction: %s, message: %s", rp.st.TranscribeID, rp.st.Direction, r.message)
		return
	}

	// final result — publish speech_ended if was speaking
	if rp.speaking {
		rp.speaking = false
		now := time.Now()
		evt := rp.st.NewSpeech("", &now)
		rp.notifyHandler.PublishWebhookEvent(ctx, rp.st.CustomerID, streaming.EventTypeSpeechEnded, evt)
		log.Debugf("Published speech_ended. transcribe_id: %s, direction: %s", rp.st.TranscribeID, rp.st.Direction)
	}

	if len(r.message) == 0 {
		return
	}
	log.Debugf("Received transcript message. transcribe_id: %s, direction: %s, message: %s", rp.st.TranscribeID, rp.st.Direction, r.message)

	t2 := time.Now()
	t3 := t2.Sub(rp.t1)
	tmGap := time.Time{}.Add(t3)

	ts, err := rp.transcriptHandler.Create(ctx, rp.st.CustomerID, rp.st.TranscribeID, rp.st.Direction, r.message, &tmGap)
	if err != nil {
		log.Errorf("Could not create transcript. err: %v", err)
		return
	}
	log.WithField("transcript", ts).Debugf("Created transcript. transcribe_id: %s, direction: %s", ts.TranscribeID, ts.Direction)
}
