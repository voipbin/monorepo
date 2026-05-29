package aipromptproposalhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestSweepStaleProposals_NoStale_NoUpdates(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	mdb.EXPECT().AIPromptProposalList(gomock.Any(), uint64(1000), gomock.Any(), gomock.Any()).Return(nil, nil)
	h.SweepStaleProposals(context.Background())
}

func TestSweepStaleProposals_MarksOldProgressingAsFailed(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	pid := uuid.Must(uuid.NewV4())
	mdb.EXPECT().AIPromptProposalList(gomock.Any(), uint64(1000), gomock.Any(), gomock.Any()).Return([]*aipromptproposal.AIPromptProposal{{
		Identity: commonidentity.Identity{ID: pid},
		Status:   aipromptproposal.StatusProgressing,
	}}, nil)
	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), pid, aipromptproposal.StatusFailed, "", "", string(aipromptproposal.ErrorEvaluatorUnavailable)).Return(int64(1), nil)

	h.SweepStaleProposals(context.Background())
}

func TestSweepExpiredProposals_DriftedOnly(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	pid := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	basis := uuid.Must(uuid.NewV4())
	current := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIPromptProposalList(gomock.Any(), uint64(1000), gomock.Any(), gomock.Any()).Return([]*aipromptproposal.AIPromptProposal{{
		Identity:             commonidentity.Identity{ID: pid},
		AIID:                 aiID,
		BasisPromptHistoryID: basis,
		Status:               aipromptproposal.StatusCompleted,
	}}, nil)
	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{CurrentPromptHistoryID: current}, nil)
	mdb.EXPECT().AIPromptProposalUpdateExpired(gomock.Any(), pid, string(aipromptproposal.ErrorPromptVersionDrifted)).Return(int64(1), nil)

	h.SweepExpiredProposals(context.Background())
}

func TestSweepExpiredProposals_NotDrifted_LeftAlone(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	pid := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	hist := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIPromptProposalList(gomock.Any(), uint64(1000), gomock.Any(), gomock.Any()).Return([]*aipromptproposal.AIPromptProposal{{
		Identity:             commonidentity.Identity{ID: pid},
		AIID:                 aiID,
		BasisPromptHistoryID: hist,
		Status:               aipromptproposal.StatusCompleted,
	}}, nil)
	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{CurrentPromptHistoryID: hist}, nil)

	h.SweepExpiredProposals(context.Background())
}
