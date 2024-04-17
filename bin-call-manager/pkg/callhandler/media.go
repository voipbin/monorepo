package callhandler

import (
	"context"
	"fmt"

	tmtts "monorepo/bin-tts-manager/models/tts"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
)

// Talk plays the tts to the given call id.
// runNext: if it true, the call will execute the next action after talk.
func (h *callHandler) Talk(ctx context.Context, callID uuid.UUID, runNext bool, text string, gender string, language string) error {
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
	tts, err := h.reqHandler.TTSV1SpeecheCreate(ctx, c.ID, text, tmtts.Gender(gender), language, 10000)
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
		// we don't want to execut the next action
		actionID = uuid.Nil
	}

	// play
	if errPlay := h.channelHandler.Play(ctx, c.ChannelID, actionID, medias, ""); errPlay != nil {
		log.Errorf("Could not play the media for tts. medias: %v, err: %v", medias, errPlay)
		return errors.Wrap(errPlay, "could not play the media for tts")
	}

	return nil
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
	if errPlay := h.channelHandler.Play(ctx, c.ChannelID, actionID, medias, ""); errPlay != nil {
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
