package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

func (h *conferenceHandler) createEndpointTarget(ctx context.Context, cf *conference.Conference) (string, error) {

	return "", nil

	// // get bridge
	// b, err := h.db.BridgeGet(ctx, cf.BridgeID)
	// if err != nil {
	// 	return "", err
	// }

	// // get bridge asterisk's ip

}

func (h *conferenceHandler) Join(id, callID uuid.UUID) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"conference": id,
			"call":       callID,
		})
	log.Info("Starting to join the call to the conference.")

	// create a bridge

	// put the call's channel into the bridge

	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return err
	}

	// create a dial string
	dialDestination, err := h.createEndpointTarget(ctx, cf)
	if err != nil {
		log.Errorf("Could not create a dial destination. err: %v", err)
		return err
	}
	log.Debugf("Created dial destination. destination: %s", dialDestination)

	// create a another channel with joining context

	return nil
}
