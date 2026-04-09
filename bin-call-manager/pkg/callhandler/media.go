package callhandler

import (
	"context"
	"fmt"

	tmstreaming "monorepo/bin-tts-manager/models/streaming"
	tmtts "monorepo/bin-tts-manager/models/tts"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/playback"
)

// Talk plays the tts to the given call id.
// runNext: if it true, the call will execute the next action after talk.
func (h *callHandler) Talk(ctx context.Context, callID uuid.UUID, runNext bool, text string, language string, provider string, voiceID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Talk",
		"call_id":  callID,
		"run_next": runNext,
	})

	c, err := h.Get(ctx, callID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "could not get call info")
	}

	// answer the call if not answered
	if c.Status != call.StatusProgressing {
		if errAnswer := h.channelHandler.Answer(ctx, c.ChannelID); errAnswer != nil {
			log.Errorf("Could not answer the call. err: %v", errAnswer)
			return fmt.Errorf("could not answer the call. err: %v", errAnswer)
		}
	}

	// send request for create wav file
	tts, err := h.reqHandler.TTSV1SpeecheCreate(ctx, c.ID, text, language, tmtts.Provider(provider), voiceID, 10000)
	if err != nil {
		log.Errorf("Could not create speech file. err: %v", err)
		return fmt.Errorf("could not create tts wav. err: %v", err)
	}
	log.WithField("tts", tts).Debugf("Received tts speech result. medial_filepath: %s", tts.MediaFilepath)

	// generate url for file play
	url := fmt.Sprintf("sound:%s", tts.MediaFilepath)

	// create a media urls
	medias := []string{
		url,
	}

	actionID := c.Action.ID
	if !runNext {
		// we don't want to execute the next action
		// generate a new uuid
		actionID = h.utilHandler.UUIDCreate()
	}

	// play
	playbackID := fmt.Sprintf("%s%s", playback.IDPrefixCall, actionID.String())
	if errPlay := h.channelHandler.Play(ctx, c.ChannelID, playbackID, medias, "", 0, 0); errPlay != nil {
		log.Errorf("Could not play the media for tts. medias: %v, err: %v", medias, errPlay)
		return errors.Wrap(errPlay, "could not play the media for tts")
	}
	log.Debugf("Played the tts media. playback_id: %s, medias: %v", playbackID, medias)

	return nil
}

// StreamingTalk plays TTS via real-time streaming instead of batch file generation.
// Audio is streamed to Asterisk via ExternalMedia AudioSocket in real-time (~200ms to first audio).
// Returns nil on success, error on failure (caller should fall back to batch Talk).
func (h *callHandler) StreamingTalk(ctx context.Context, callID uuid.UUID, text string, language string, provider string, voiceID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "StreamingTalk",
		"call_id": callID,
	})

	c, err := h.Get(ctx, callID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "could not get call info")
	}
	log.WithField("call", c).Debugf("Retrieved call info. call_id: %s", c.ID)

	// answer the call if not answered
	if c.Status != call.StatusProgressing {
		if errAnswer := h.channelHandler.Answer(ctx, c.ChannelID); errAnswer != nil {
			log.Errorf("Could not answer the call. err: %v", errAnswer)
			return errors.Wrap(errAnswer, "could not answer the call")
		}
	}

	// default provider to "gcp" if not specified
	if provider == "" {
		provider = "gcp"
	}

	// Only GCP streaming is production-ready. Non-GCP vendors (ElevenLabs, AWS)
	// use ConnAstDone for WaitFinish which won't close until Stop() is called,
	// causing a 60s RPC timeout. Return error to trigger batch fallback.
	if provider != "gcp" {
		return fmt.Errorf("streaming talk only supports gcp provider, got: %s", provider)
	}

	// Step 1: Create streaming session with provider/voiceID
	st, err := h.reqHandler.TTSV1StreamingCreateWithProvider(ctx,
		c.CustomerID,
		tmstreaming.ReferenceTypeCall,
		c.ID,
		language,
		provider,
		voiceID,
		tmstreaming.DirectionBoth,
	)
	if err != nil {
		log.Errorf("Could not create streaming session. err: %v", err)
		return errors.Wrap(err, "could not create streaming session")
	}
	log.WithField("streaming", st).Debugf("Created streaming session. streaming_id: %s, pod_id: %s", st.ID, st.PodID)

	// Step 2: SayInit — begin message
	messageID := uuid.Must(uuid.NewV4())
	if _, errInit := h.reqHandler.TTSV1StreamingSayInit(ctx, st.PodID, st.ID, messageID); errInit != nil {
		log.Errorf("Could not init streaming say. streaming_id: %s, err: %v", st.ID, errInit)
		h.streamingCleanup(ctx, st)
		return errors.Wrap(errInit, "could not init streaming say")
	}

	// Step 3: SayAdd — send text
	if errAdd := h.reqHandler.TTSV1StreamingSayAdd(ctx, st.PodID, st.ID, messageID, text); errAdd != nil {
		log.Errorf("Could not add text to streaming. streaming_id: %s, err: %v", st.ID, errAdd)
		h.streamingCleanup(ctx, st)
		return errors.Wrap(errAdd, "could not add text to streaming")
	}

	// Step 4: SayFinish — signal no more text (non-blocking, audio continues streaming)
	if _, errFinish := h.reqHandler.TTSV1StreamingSayFinish(ctx, st.PodID, st.ID, messageID); errFinish != nil {
		log.Errorf("Could not finish streaming say. streaming_id: %s, err: %v", st.ID, errFinish)
		h.streamingCleanup(ctx, st)
		return errors.Wrap(errFinish, "could not finish streaming say")
	}

	// Step 5: Wait for all audio to be delivered to Asterisk
	if errWait := h.reqHandler.TTSV1StreamingWaitFinish(ctx, st.PodID, st.ID); errWait != nil {
		log.Errorf("Streaming wait finish failed. streaming_id: %s, err: %v", st.ID, errWait)
		h.streamingCleanup(ctx, st)
		return errors.Wrap(errWait, "streaming wait finish failed")
	}

	// Step 6: Cleanup streaming session
	h.streamingCleanup(ctx, st)

	log.Infof("Streaming talk completed. streaming_id: %s", st.ID)
	return nil
}

// streamingCleanup stops and deletes a streaming session. Best-effort, logs errors but doesn't return them.
func (h *callHandler) streamingCleanup(ctx context.Context, st *tmstreaming.Streaming) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "streamingCleanup",
		"streaming_id": st.ID,
	})

	if _, err := h.reqHandler.TTSV1StreamingDelete(ctx, st.ID); err != nil {
		log.Errorf("Could not delete streaming session. err: %v", err)
	}
}

// Play plays the media to the given call id.
// runNext: if it true, the call will execute the next action after talk.
func (h *callHandler) Play(ctx context.Context, callID uuid.UUID, runNext bool, urls []string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Play",
		"call_id":  callID,
		"run_next": runNext,
		"urls":     urls,
	})

	c, err := h.Get(ctx, callID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "could not get call info")
	}

	// create a media string array
	medias := []string{}
	for _, url := range urls {
		media := fmt.Sprintf("sound:%s", url)
		medias = append(medias, media)
	}

	// get action id
	actionID := c.Action.ID
	if !runNext {
		// we don't want to execut the next action
		actionID = uuid.Nil
	}

	// play
	playbackID := fmt.Sprintf("%s%s", playback.IDPrefixCall, actionID.String())
	if errPlay := h.channelHandler.Play(ctx, c.ChannelID, playbackID, medias, "", 0, 0); errPlay != nil {
		log.Errorf("Could not play the media. media: %v, err: %v", medias, errPlay)
		return errors.Wrap(errPlay, "could not play the media")
	}

	return nil
}

// MediaStop stops the media currently playing on the call.
func (h *callHandler) MediaStop(ctx context.Context, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "MediaStop",
		"call_id": callID,
	})

	c, err := h.Get(ctx, callID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "could not get call info")
	}

	if errStop := h.channelHandler.PlaybackStop(ctx, c.ChannelID); errStop != nil {
		log.Errorf("Could not stop the talk ")
	}

	return nil
}
