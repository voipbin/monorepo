package campaigncallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
)

// Done updates the campaigncall status to done and update the outdialtarget's status depends on the result.
func (h *campaigncallHandler) Done(ctx context.Context, id uuid.UUID, result campaigncall.Result) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Done",
		"campaigncall_id": id,
		"result":          result,
	})
	log.Debugf("Updating campaigncall status done. campagincall_id: %s, result: %s", id, result)

	// update campaigncall to done
	cc, err := h.updateStatusDone(ctx, id, result)
	if err != nil {
		log.Errorf("Could not update the campaigncall status to done. err: %v", err)
		return nil, err
	}

	// calculate omoutdialtarget status
	otStatus, err := calcDialtargetStatus(result)
	if err != nil {
		log.Errorf("Could not calculate the dialtarget status. err: %v", err)
		return nil, err
	}
	log.Debugf("Calculated dialtarget status. status: %s", otStatus)

	// send request to update outdial target
	ot, err := h.reqHandler.OutdialV1OutdialtargetUpdateStatus(ctx, cc.OutdialTargetID, otStatus)
	if err != nil {
		log.Errorf("Could not update the outdialtarget status correctly. status: %s, err: %v", otStatus, err)
		return nil, err
	}
	log.WithField("otdialtarget", ot).Debugf("Updated outdial target status. status: %s", otStatus)

	return cc, nil
}

// Progressing updates the campaigncall status to progressing and update the outdialtarget's status depends on the result.
func (h *campaigncallHandler) Progressing(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Progress",
		"campaigncall_id": id,
	})
	log.Debug("Updating campaigncall status progress.")

	// update campaigncall to done
	cc, err := h.updateStatus(ctx, id, campaigncall.StatusProgressing)
	if err != nil {
		log.Errorf("Could not update the campaigncall status to done. err: %v", err)
		return nil, err
	}

	return cc, nil
}

// calcDialtargetStatus returns calculated omoutdialtarget status based on campaigncall result
func calcDialtargetStatus(result campaigncall.Result) (omoutdialtarget.Status, error) {

	mapStatus := map[campaigncall.Result]omoutdialtarget.Status{
		campaigncall.ResultNone:    omoutdialtarget.StatusIdle,
		campaigncall.ResultSuccess: omoutdialtarget.StatusDone,
		campaigncall.ResultFail:    omoutdialtarget.StatusIdle,
	}

	res, ok := mapStatus[result]
	if !ok {
		return omoutdialtarget.StatusIdle, fmt.Errorf("status not found")
	}

	return res, nil
}
