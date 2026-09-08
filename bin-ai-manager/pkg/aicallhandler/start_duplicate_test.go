package aicallhandler

import (
	stderrors "errors"
	"fmt"
	"testing"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// startDuplicateFixture wires just enough of the handler to drive Start down to
// the aicall insert, which is where a pinned-id collision surfaces.
func startDuplicateFixture(mc *gomock.Controller) (*aicallHandler, *dbhandler.MockDBHandler, *utilhandler.MockUtilHandler, *ai.AI) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)

	h := &aicallHandler{
		utilHandler:   mockUtil,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		db:            mockDB,
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		aiHandler:     mockAI,
	}

	a := &ai.AI{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
		},
	}
	mockAI.EXPECT().Get(gomock.Any(), a.ID).Return(a, nil)
	return h, mockDB, mockUtil, a
}

// Test_Start_callerSpecifiedID_duplicate pins the 409 contract for aicalls.
// Without the AlreadyExists mapping the widget cannot tell "your assignment is
// already spent, boot again" from a generic failure.
func Test_Start_callerSpecifiedID_duplicate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockUtil, a := startDuplicateFixture(mc)
	pinnedID := uuid.FromStringOrNil("beefbeef-0000-0000-0000-000000000001")

	// pipecatcall id is still generated; only the aicall id is pinned.
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000003")).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-09-08T00:00:00.000000Z").AnyTimes()
	mockDB.EXPECT().AIcallCreate(gomock.Any(), gomock.Cond(func(c *aicall.AIcall) bool {
		return c.ID == pinnedID
	})).Return(fmt.Errorf("could not execute. err: Error 1062 (23000): Duplicate entry"))

	_, err := h.Start(t.Context(), pinnedID, aicall.AssistanceTypeAI, a.ID, uuid.Nil, aicall.ReferenceTypeNone, uuid.Nil)
	if err == nil {
		t.Fatal("Wrong match. expect: already exists error, got: ok")
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Wrong error type. expect: *cerrors.VoipbinError, got: %T (%v)", err, err)
	}
	if ve.Status != cerrors.StatusAlreadyExists {
		t.Errorf("Wrong status. expect: %v, got: %v", cerrors.StatusAlreadyExists, ve.Status)
	}

	// The sentinel must not carry the driver text. If it did, IsErrDuplicate
	// would match it again and the contact_case retry loop could swallow it.
	if dbhandler.IsErrDuplicate(err) {
		t.Error("The AlreadyExists sentinel still looks like a driver duplicate error")
	}
}

// Test_Start_serverGeneratedID_duplicateIsNotRemapped is the other half: a
// collision on an id the server chose is an internal fault, not a spent
// assignment, and must not be reported as 409.
func Test_Start_serverGeneratedID_duplicateIsNotRemapped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockUtil, a := startDuplicateFixture(mc)

	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000004")).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-09-08T00:00:00.000000Z").AnyTimes()
	mockDB.EXPECT().AIcallCreate(gomock.Any(), gomock.Any()).Return(
		fmt.Errorf("could not execute. err: Error 1062 (23000): Duplicate entry"),
	)

	_, err := h.Start(t.Context(), uuid.Nil, aicall.AssistanceTypeAI, a.ID, uuid.Nil, aicall.ReferenceTypeNone, uuid.Nil)
	if err == nil {
		t.Fatal("Wrong match. expect: error, got: ok")
	}
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) && ve.Status == cerrors.StatusAlreadyExists {
		t.Error("A server-generated id collision must not be reported as AlreadyExists")
	}
}

// Test_Start_callerSpecifiedID_nonDuplicateIsPassedThrough guards against the
// classification swallowing unrelated failures.
func Test_Start_callerSpecifiedID_nonDuplicateIsPassedThrough(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockUtil, a := startDuplicateFixture(mc)
	pinnedID := uuid.FromStringOrNil("beefbeef-0000-0000-0000-000000000002")

	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("eeeeeeee-0000-0000-0000-000000000005")).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-09-08T00:00:00.000000Z").AnyTimes()
	mockDB.EXPECT().AIcallCreate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("connection refused"))

	_, err := h.Start(t.Context(), pinnedID, aicall.AssistanceTypeAI, a.ID, uuid.Nil, aicall.ReferenceTypeNone, uuid.Nil)
	if err == nil {
		t.Fatal("Wrong match. expect: error, got: ok")
	}
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) && ve.Status == cerrors.StatusAlreadyExists {
		t.Error("An unrelated insert failure must not be reported as AlreadyExists")
	}
}

// Test_Create_callerSpecifiedID and its messaging twin cover the branch the
// whole binding depends on: with a pinned id the row must take that id and the
// server must not mint one. gomock enforces the second half -- the only
// UUIDCreate() allowed is the pipecatcall id, so a regression that also
// re-generated the aicall id would exceed the expectation and fail.
func Test_Create_callerSpecifiedID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &aicallHandler{
		utilHandler:   mockUtil,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := t.Context()

	pinnedID := uuid.FromStringOrNil("beefbeef-0000-0000-0000-00000000000a")
	a := &ai.AI{Identity: commonidentity.Identity{
		ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
		CustomerID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
	}}
	created := &aicall.AIcall{Identity: commonidentity.Identity{ID: pinnedID, CustomerID: a.CustomerID}}

	// No UUIDCreate expectation at all: Create must not mint an id.
	mockDB.EXPECT().AIcallCreate(ctx, gomock.Cond(func(c *aicall.AIcall) bool {
		return c.ID == pinnedID
	})).Return(nil)
	mockDB.EXPECT().AIcallGet(ctx, pinnedID).Return(created, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, a.CustomerID, aicall.EventTypeStatusInitializing, created)

	res, err := h.Create(ctx, pinnedID, a, aicall.AssistanceTypeAI, a.ID, uuid.Nil,
		aicall.ReferenceTypeNone, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, nil, nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != pinnedID {
		t.Errorf("Wrong aicall id. expect: %v, got: %v", pinnedID, res.ID)
	}
}

func Test_CreateByMessaging_callerSpecifiedID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &aicallHandler{
		utilHandler:   mockUtil,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := t.Context()

	pinnedID := uuid.FromStringOrNil("beefbeef-0000-0000-0000-00000000000b")
	a := &ai.AI{Identity: commonidentity.Identity{
		ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
		CustomerID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
	}}
	created := &aicall.AIcall{Identity: commonidentity.Identity{ID: pinnedID, CustomerID: a.CustomerID}}

	mockDB.EXPECT().AIcallCreate(ctx, gomock.Cond(func(c *aicall.AIcall) bool {
		return c.ID == pinnedID
	})).Return(nil)
	mockDB.EXPECT().AIcallGet(ctx, pinnedID).Return(created, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, a.CustomerID, aicall.EventTypeStatusInitializing, created)

	res, err := h.CreateByMessaging(ctx, pinnedID, a, aicall.AssistanceTypeAI, a.ID, uuid.Nil,
		aicall.ReferenceTypeNone, uuid.Nil, uuid.Nil, uuid.Nil, nil, nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != pinnedID {
		t.Errorf("Wrong aicall id. expect: %v, got: %v", pinnedID, res.ID)
	}
}
