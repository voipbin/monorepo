package conferencehandler

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

// Stop is stopping the conference
func (h *conferenceHandler) Stop(id uuid.UUID) error {
	ctx := context.Background()

	log.WithFields(
		log.Fields{
			"conference": id.String(),
		}).Info("Stopping conference.")

	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.WithFields(
			log.Fields{
				"conference": id.String(),
			}).Warnf("Could not get conference for stop. err: %v", err)
		return err
	}

	// if the conference is already terminated or stopping, just return at here
	if cf.Status == conference.StatusTerminated || cf.Status == conference.StatusStopping {
		log.WithFields(
			log.Fields{
				"conference": id.String(),
			}).Infof("The conference is already terminated or being terminated. status: %s", cf.Status)

		return nil
	}

	// set the status to stopping
	if err := h.db.ConferenceSetStatus(ctx, id, conference.StatusStopping); err != nil {
		log.WithFields(
			log.Fields{
				"conference": id.String(),
			}).Warnf("Could not update the status for conference stopping. err: %v", err)
		return err
	}

	// leave out all the call from the bridge
	for _, callID := range cf.CallIDs {
		if err := h.Leave(id, callID); err != nil {
			log.WithFields(
				log.Fields{
					"conference": id.String(),
					"call":       callID.String(),
				}).Errorf("Could not leave out the call from the conference. err: %v", err)

			continue
		}
	}

	return nil
}

// stopConferTypeEcho
func (h *conferenceHandler) stopConfTypeEcho(cf *conference.Conference, c *call.Call) error {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// defer cancel()

	// get

	return nil
}
