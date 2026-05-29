package aipromptproposalhandler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aipromptproposal"
)

// SweepStaleProposals marks any 'progressing' proposal older than staleProposalAgeMinutes as
// failed. Called at service startup to recover records orphaned by pod restarts.
func (h *aipromptproposalHandler) SweepStaleProposals(ctx context.Context) {
	staleBefore := h.utilHandler.TimeGetCurTimeAdd(-staleProposalAgeMinutes * time.Minute)
	filters := map[aipromptproposal.Field]any{
		aipromptproposal.FieldStatus:  aipromptproposal.StatusProgressing,
		aipromptproposal.FieldDeleted: false,
	}

	stale, err := h.db.AIPromptProposalList(ctx, 1000, staleBefore, filters)
	if err != nil {
		logrus.WithError(err).Error("startup stale proposal sweep: list failed")
		return
	}
	if len(stale) == 0 {
		logrus.Infof("startup stale proposal sweep: nothing to do (threshold=%d min)", staleProposalAgeMinutes)
		return
	}

	logrus.Infof("startup stale proposal sweep: marking %d stale proposal(s) as failed", len(stale))
	for _, p := range stale {
		if _, dbErr := h.db.AIPromptProposalUpdateFinal(ctx, p.ID, aipromptproposal.StatusFailed, "", "", string(aipromptproposal.ErrorEvaluatorUnavailable)); dbErr != nil {
			logrus.WithError(dbErr).Errorf("startup stale proposal sweep: could not mark %s as failed", p.ID)
		}
	}
}

// SweepExpiredProposals marks completed proposals as expired when the AI's basis prompt has drifted.
func (h *aipromptproposalHandler) SweepExpiredProposals(ctx context.Context) {
	cutoff := h.utilHandler.TimeGetCurTimeAdd(-proposalExpiryHours * time.Hour)
	filters := map[aipromptproposal.Field]any{
		aipromptproposal.FieldStatus:  aipromptproposal.StatusCompleted,
		aipromptproposal.FieldDeleted: false,
	}

	cand, err := h.db.AIPromptProposalList(ctx, 1000, cutoff, filters)
	if err != nil {
		logrus.WithError(err).Error("expiry sweep: list failed")
		return
	}

	for _, p := range cand {
		ai, gerr := h.db.AIGet(ctx, p.AIID)
		if gerr != nil {
			continue
		}
		if ai.CurrentPromptHistoryID == p.BasisPromptHistoryID {
			continue
		}
		if _, uerr := h.db.AIPromptProposalUpdateExpired(ctx, p.ID, string(aipromptproposal.ErrorPromptVersionDrifted)); uerr != nil {
			logrus.WithError(uerr).Errorf("expiry sweep: could not mark %s expired", p.ID)
		}
	}
}
