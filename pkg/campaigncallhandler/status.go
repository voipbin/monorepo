package campaigncallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
)

func (h *campaigncallHandler) Done(ctx context.Context, id uuid.UUID, status campaigncall.Status, reason string) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "Done",
			"campaigncall_id": id,
		})
	log.Debug("Campaigncall done.")

	// // get campaigncall
	// cc, err := h.db.CampaigncallGet(ctx, id)
	// if err != nil {
	// 	log.Errorf("Could not get campaigncall. err: %v", err)
	// 	return nil, err
	// }

	// update campaigncall to done
	cc, err := h.UpdateStatus(ctx, id, campaigncall.StatusDone)
	if err != nil {
		log.Errorf("Could not update the campaigncall status to done. err: %v", err)
	}

	// send request to update outdial target
	h.reqHandler.OMV1OutdialtargetGet()
}
