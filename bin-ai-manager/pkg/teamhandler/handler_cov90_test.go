package teamhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

type cov90Mocks struct {
	util   *utilhandler.MockUtilHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler
	db     *dbhandler.MockDBHandler
}

func newCov90Handler(t *testing.T) (*teamHandler, *cov90Mocks) {
	t.Helper()

	mc := gomock.NewController(t)
	m := &cov90Mocks{
		util:   utilhandler.NewMockUtilHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
	}

	return &teamHandler{utilHandler: m.util, reqHandler: m.req, notifyHandler: m.notify, db: m.db}, m
}

var (
	cov90TeamID     = uuid.FromStringOrNil("a2a2a2a2-2222-2222-2222-222222222222")
	cov90CustomerID = uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222")
	cov90DirectID   = uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222")
)

// Test_DirectHashRegenerate_Failures verifies that every failing dependency of
// the regeneration is reported and stops the sequence, so that a failed
// regeneration never leaves a half-updated team behind.
func Test_DirectHashRegenerate_Failures(t *testing.T) {
	errBoom := errors.New("boom")
	ctx := context.Background()

	existing := &team.Team{Identity: commonidentity.Identity{ID: cov90TeamID, CustomerID: cov90CustomerID}, DirectID: cov90DirectID}
	fresh := &team.Team{Identity: commonidentity.Identity{ID: cov90TeamID, CustomerID: cov90CustomerID}}
	direct := &dmdirect.Direct{Identity: commonidentity.Identity{ID: cov90DirectID}, Hash: "newhash456"}

	tests := []struct {
		name  string
		setup func(m *cov90Mocks)
	}{
		{"team cannot be read", func(m *cov90Mocks) {
			m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(nil, errBoom)
		}},
		{"regenerate of the existing direct fails", func(m *cov90Mocks) {
			m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(existing, nil)
			m.req.EXPECT().DirectV1DirectRegenerate(ctx, cov90DirectID).Return(nil, errBoom)
		}},
		{"create of the missing direct fails", func(m *cov90Mocks) {
			m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(fresh, nil)
			m.req.EXPECT().DirectV1DirectCreate(ctx, cov90CustomerID, dmdirect.ResourceTypeAITeam, cov90TeamID).Return(nil, errBoom)
		}},
		{"team update fails", func(m *cov90Mocks) {
			m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(existing, nil)
			m.req.EXPECT().DirectV1DirectRegenerate(ctx, cov90DirectID).Return(direct, nil)
			m.db.EXPECT().TeamUpdate(ctx, cov90TeamID, map[team.Field]any{team.FieldDirectID: cov90DirectID, team.FieldDirectHash: "newhash456"}).Return(errBoom)
		}},
		{"updated team cannot be read", func(m *cov90Mocks) {
			gomock.InOrder(
				m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(existing, nil),
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, cov90DirectID).Return(direct, nil),
				m.db.EXPECT().TeamUpdate(ctx, cov90TeamID, gomock.Any()).Return(nil),
				m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(nil, errBoom),
			)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov90Handler(t)
			tt.setup(m)

			res, err := h.DirectHashRegenerate(ctx, cov90TeamID)
			if !errors.Is(err, errBoom) {
				t.Errorf("expected the dependency error to be wrapped, got: %v", err)
			}
			if res != nil {
				t.Errorf("expected no team with an error, got: %+v", res)
			}
		})
	}
}

// Test_Create_DirectCreateFails verifies that a failed direct creation stops the
// team creation before anything is stored.
func Test_Create_DirectCreateFails(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	errBoom := errors.New("boom")
	ctx := context.Background()

	h, m := newCov90Handler(t)
	m.db.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
	m.util.EXPECT().UUIDCreate().Return(cov90TeamID)
	m.req.EXPECT().DirectV1DirectCreate(ctx, cov90CustomerID, dmdirect.ResourceTypeAITeam, cov90TeamID).Return(nil, errBoom)

	res, err := h.Create(ctx, cov90CustomerID, "t", "d", memberA, []team.Member{{ID: memberA, Name: "A", AIID: aiA}}, nil)
	if !errors.Is(err, errBoom) || res != nil {
		t.Errorf("expected the direct error and no team, got: %v, %+v", err, res)
	}
}

// Test_Create_OrphanCleanupFails verifies that the original storage error is
// still returned when the cleanup of the orphaned direct fails as well.
func Test_Create_OrphanCleanupFails(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	errStore := errors.New("store failed")
	ctx := context.Background()

	h, m := newCov90Handler(t)
	m.db.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
	m.util.EXPECT().UUIDCreate().Return(cov90TeamID)
	m.req.EXPECT().DirectV1DirectCreate(ctx, cov90CustomerID, dmdirect.ResourceTypeAITeam, cov90TeamID).Return(&dmdirect.Direct{Identity: commonidentity.Identity{ID: cov90DirectID}, Hash: "h"}, nil)
	m.db.EXPECT().TeamCreate(ctx, gomock.Any()).Return(errStore)
	m.req.EXPECT().DirectV1DirectDelete(ctx, cov90DirectID).Return(nil, errors.New("cleanup failed"))

	res, err := h.Create(ctx, cov90CustomerID, "t", "d", memberA, []team.Member{{ID: memberA, Name: "A", AIID: aiA}}, nil)
	if !errors.Is(err, errStore) || res != nil {
		t.Errorf("expected the storage error and no team, got: %v, %+v", err, res)
	}
}

// Test_Delete_DirectDeleteFails verifies that a failing direct deletion does not
// block the deletion of the team (it is best effort).
func Test_Delete_DirectDeleteFails(t *testing.T) {
	ctx := context.Background()
	stored := &team.Team{Identity: commonidentity.Identity{ID: cov90TeamID, CustomerID: cov90CustomerID}, DirectID: cov90DirectID}

	h, m := newCov90Handler(t)
	gomock.InOrder(
		m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(stored, nil),
		m.req.EXPECT().DirectV1DirectDelete(ctx, cov90DirectID).Return(nil, errors.New("direct delete failed")),
		m.db.EXPECT().TeamDelete(ctx, cov90TeamID).Return(nil),
		m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(stored, nil),
		m.notify.EXPECT().PublishWebhookEvent(ctx, cov90CustomerID, team.EventTypeDeleted, stored),
	)

	res, err := h.Delete(ctx, cov90TeamID)
	if err != nil || res != stored {
		t.Errorf("expected the deleted team, got: %v, %+v", err, res)
	}
}

// Test_Update_ParameterAndReadBackFailure verifies that the parameter is written
// as given, and that a failure to read the updated team is reported.
func Test_Update_ParameterAndReadBackFailure(t *testing.T) {
	ctx := context.Background()
	param := map[string]any{"key": "value"}
	errRead := errors.New("read failed")

	h, m := newCov90Handler(t)
	m.db.EXPECT().TeamUpdate(ctx, cov90TeamID, map[team.Field]any{team.FieldParameter: param}).Return(nil)
	m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(nil, errRead)

	res, err := h.Update(ctx, cov90TeamID, nil, nil, nil, nil, &param)
	if !errors.Is(err, errRead) || res != nil {
		t.Errorf("expected the read error and no team, got: %v, %+v", err, res)
	}
}

// Test_Delete_TeamCannotBeRead verifies that a team that cannot be read is not deleted.
func Test_Delete_TeamCannotBeRead(t *testing.T) {
	ctx := context.Background()
	errRead := errors.New("read failed")

	h, m := newCov90Handler(t)
	m.db.EXPECT().TeamGet(ctx, cov90TeamID).Return(nil, errRead)

	res, err := h.Delete(ctx, cov90TeamID)
	if !errors.Is(err, errRead) || res != nil {
		t.Errorf("expected the read error and no team, got: %v, %+v", err, res)
	}
}

// Test_NewTeamHandler verifies that the constructor wires the given dependencies.
func Test_NewTeamHandler(t *testing.T) {
	mc := gomock.NewController(t)
	req := requesthandler.NewMockRequestHandler(mc)
	notify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewMockDBHandler(mc)

	h, ok := NewTeamHandler(req, notify, db).(*teamHandler)
	if !ok {
		t.Fatalf("expected a *teamHandler")
	}
	if h.reqHandler != req || h.notifyHandler != notify || h.db != db || h.utilHandler == nil {
		t.Errorf("the constructor did not wire the given dependencies")
	}
}
