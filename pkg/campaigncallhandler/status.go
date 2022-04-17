package campaigncallhandler

import (
	"context"

	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
)

// Done updates the campaigncall status to done and update the outdialtarget's status depends on the result.
func (h *campaigncallHandler) Done(ctx context.Context, id uuid.UUID, status campaigncall.Status, result campaigncall.Result) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "Done",
			"campaigncall_id": id,
		})
	log.Debug("Updating campaigncall status done.")

	// update campaigncall to done
	cc, err := h.UpdateStatus(ctx, id, campaigncall.StatusDone)
	if err != nil {
		log.Errorf("Could not update the campaigncall status to done. err: %v", err)
		return nil, err
	}

	// calculate omoutdialtarget status
	otStatus := calcDialtargetStatus(result)
	log.Debugf("Calculated dialtarget status. status: %s", otStatus)

	// send request to update outdial target
	ot, err := h.reqHandler.OMV1OutdialtargetUpdateStatus(ctx, cc.OutdialTargetID, otStatus)
	if err != nil {
		log.Errorf("Could not update the outdialtarget status correctly. status: %s, err: %v", otStatus, err)
		return nil, err
	}
	log.WithField("otdialtarget", ot).Debugf("Updated outdial target status. status: %s", otStatus)

	return cc, nil
}

// calcDialtargetStatus returns calculated omoutdialtarget status based on campaigncall result
func calcDialtargetStatus(result campaigncall.Result) omoutdialtarget.Status {
	if result == campaigncall.ResultFail {
		return omoutdialtarget.StatusIdle
	}

	return omoutdialtarget.StatusDone
}
