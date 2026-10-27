package casehandler

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	tmtag "monorepo/bin-tag-manager/models/tag"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

var errPropagationBoom = stderrors.New("injected failure")

var (
	propCustomerID = uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000001")
	propCaseID     = uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000002")
	propTagID      = uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000003")
	propOtherTagID = uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000004")
	propContactID  = uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000005")
)

// newPropagationHandler returns a caseHandler whose collaborators are strict
// mocks, so any call that is not expected fails the test.
func newPropagationHandler(t *testing.T) (*caseHandler, *dbhandler.MockDBHandler, *requesthandler.MockRequestHandler, *notifyhandler.MockNotifyHandler, *utilhandler.MockUtilHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}

	return h, mockDB, mockReq, mockNotify, mockUtil
}

func propCase(tagIDs ...uuid.UUID) *kase.Case {
	return &kase.Case{ID: propCaseID, CustomerID: propCustomerID, TagIDs: tagIDs}
}

func Test_Assign_ErrorPropagation(t *testing.T) {
	ownerID := uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-0000000000a1")

	tests := []struct {
		name string

		firstGetErr  error
		updateErr    error
		secondGetErr error

		expectWrapped bool
	}{
		{name: "first lookup fails", firstGetErr: errPropagationBoom},
		{name: "owner update fails", updateErr: errPropagationBoom, expectWrapped: true},
		{name: "re-read fails", secondGetErr: errPropagationBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, _, _, _ := newPropagationHandler(t)
			ctx := context.Background()

			if tt.firstGetErr != nil {
				mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(nil, tt.firstGetErr)
			} else {
				mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(propCase(), nil)
				if tt.updateErr != nil {
					mockDB.EXPECT().CaseUpdateOwner(ctx, propCustomerID, propCaseID, commonidentity.OwnerTypeAgent, ownerID).Return(tt.updateErr)
				} else {
					mockDB.EXPECT().CaseUpdateOwner(ctx, propCustomerID, propCaseID, commonidentity.OwnerTypeAgent, ownerID).Return(nil)
					mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(nil, tt.secondGetErr)
				}
			}

			res, err := h.Assign(ctx, propCustomerID, propCaseID, commonidentity.OwnerTypeAgent, ownerID)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, errPropagationBoom) {
				t.Fatalf("Wrong match. expect: injected error, got: %v", err)
			}
			if tt.expectWrapped && !strings.Contains(err.Error(), "could not update owner. Assign") {
				t.Errorf("Wrong match. expect: wrapped owner update text, got: %v", err)
			}
		})
	}
}

func Test_CaseList_ErrorPropagation(t *testing.T) {
	h, mockDB, _, _, _ := newPropagationHandler(t)
	ctx := context.Background()

	mockDB.EXPECT().CaseList(ctx, propCustomerID, uint64(defaultCaseListSize+1), "", "", commonidentity.OwnerType(""), uuid.Nil, uuid.Nil, "").Return(nil, errPropagationBoom)

	items, token, err := h.CaseList(ctx, propCustomerID, 0, "", "", "", uuid.Nil, uuid.Nil, "")
	if items != nil || token != "" {
		t.Errorf("Wrong match. expect: nil items and empty token, got: %v, %q", items, token)
	}
	if !stderrors.Is(err, errPropagationBoom) {
		t.Errorf("Wrong match. expect: injected error, got: %v", err)
	}
}

func Test_CaseTagAdd_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name string

		lookupErr error
		updateErr error

		expectWrapped string
	}{
		{name: "ownership lookup fails", lookupErr: errPropagationBoom},
		{name: "tag update fails", updateErr: errPropagationBoom, expectWrapped: "could not update case tag_ids. CaseTagAdd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, mockReq, _, _ := newPropagationHandler(t)
			ctx := context.Background()

			if tt.lookupErr != nil {
				mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(nil, tt.lookupErr)
			} else {
				mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(propCase(propOtherTagID), nil)
				mockReq.EXPECT().TagV1TagGet(ctx, propTagID).Return(&tmtag.Tag{}, nil)
				mockDB.EXPECT().CaseUpdateTagIDs(ctx, propCustomerID, propCaseID, []uuid.UUID{propOtherTagID, propTagID}).Return(tt.updateErr)
			}

			err := h.CaseTagAdd(ctx, propCustomerID, propCaseID, propTagID)
			if !stderrors.Is(err, errPropagationBoom) {
				t.Fatalf("Wrong match. expect: injected error, got: %v", err)
			}
			if tt.expectWrapped != "" && !strings.Contains(err.Error(), tt.expectWrapped) {
				t.Errorf("Wrong match. expect: %q in error, got: %v", tt.expectWrapped, err)
			}
		})
	}
}

func Test_CaseTagRemove_KeepsOtherTags(t *testing.T) {
	h, mockDB, _, mockNotify, _ := newPropagationHandler(t)
	ctx := context.Background()

	mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(propCase(propOtherTagID, propTagID), nil)
	mockDB.EXPECT().CaseUpdateTagIDs(ctx, propCustomerID, propCaseID, []uuid.UUID{propOtherTagID}).Return(nil)
	mockNotify.EXPECT().PublishEvent(ctx, kase.EventTypeCaseTagRemoved, &kase.CaseTagEvent{CaseID: propCaseID, TagID: propTagID})

	if err := h.CaseTagRemove(ctx, propCustomerID, propCaseID, propTagID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_CaseTagRemove_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name string

		lookupErr error
		updateErr error

		expectWrapped string
	}{
		{name: "ownership lookup fails", lookupErr: errPropagationBoom},
		{name: "tag update fails", updateErr: errPropagationBoom, expectWrapped: "could not update case tag_ids. CaseTagRemove"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, _, _, _ := newPropagationHandler(t)
			ctx := context.Background()

			if tt.lookupErr != nil {
				mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(nil, tt.lookupErr)
			} else {
				mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(propCase(propTagID), nil)
				mockDB.EXPECT().CaseUpdateTagIDs(ctx, propCustomerID, propCaseID, []uuid.UUID{}).Return(tt.updateErr)
			}

			err := h.CaseTagRemove(ctx, propCustomerID, propCaseID, propTagID)
			if !stderrors.Is(err, errPropagationBoom) {
				t.Fatalf("Wrong match. expect: injected error, got: %v", err)
			}
			if tt.expectWrapped != "" && !strings.Contains(err.Error(), tt.expectWrapped) {
				t.Errorf("Wrong match. expect: %q in error, got: %v", tt.expectWrapped, err)
			}
		})
	}
}

func Test_CaseNoteCreate_DBError(t *testing.T) {
	h, mockDB, _, _, mockUtil := newPropagationHandler(t)
	ctx := context.Background()

	mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(propCase(), nil)
	mockUtil.EXPECT().TimeNow().Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-0000000000b1"))
	mockDB.EXPECT().CaseNoteCreate(ctx, gomock.Any()).Return(errPropagationBoom)

	res, err := h.CaseNoteCreate(ctx, propCustomerID, propCaseID, "agent", nil, "synthetic note")
	if res != nil {
		t.Errorf("Wrong match. expect: nil note, got: %v", res)
	}
	if !stderrors.Is(err, errPropagationBoom) || !strings.Contains(err.Error(), "could not create case note. CaseNoteCreate") {
		t.Errorf("Wrong match. expect: wrapped injected error, got: %v", err)
	}
}

func Test_CaseNoteDelete_DBError(t *testing.T) {
	h, mockDB, _, _, _ := newPropagationHandler(t)
	ctx := context.Background()
	noteID := uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-0000000000b2")

	mockDB.EXPECT().CaseNoteDelete(ctx, propCustomerID, propCaseID, noteID).Return(errPropagationBoom)

	err := h.CaseNoteDelete(ctx, propCustomerID, propCaseID, noteID)
	if err != errPropagationBoom { //nolint:errorlint // the handler must return the db error unchanged
		t.Errorf("Wrong match. expect: injected error unchanged, got: %v", err)
	}
}

func Test_UpdateContact_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID

		contactGetErr error
		updateErr     error
		clearErr      error
		rereadErr     error

		expectText      string
		expectIsWrapped bool
	}{
		{name: "contact lookup fails", contactID: propContactID, contactGetErr: errPropagationBoom, expectText: "could not get contact. UpdateContact", expectIsWrapped: true},
		{name: "attach update fails", contactID: propContactID, updateErr: errPropagationBoom, expectText: "could not update case contact_id. UpdateContact", expectIsWrapped: true},
		{name: "detach clear fails", contactID: uuid.Nil, clearErr: errPropagationBoom, expectText: "could not clear case contact_id. UpdateContact", expectIsWrapped: true},
		{name: "re-read fails after attach", contactID: propContactID, rereadErr: errPropagationBoom, expectText: "could not get updated case. UpdateContact", expectIsWrapped: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, _, _, _ := newPropagationHandler(t)
			ctx := context.Background()

			mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(propCase(), nil)

			switch {
			case tt.contactGetErr != nil:
				mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(nil, tt.contactGetErr)
			case tt.contactID == uuid.Nil:
				mockDB.EXPECT().CaseClearContactID(ctx, propCustomerID, propCaseID).Return(tt.clearErr)
			default:
				ct := &contact.Contact{Identity: commonidentity.Identity{ID: tt.contactID, CustomerID: propCustomerID}}
				mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(ct, nil)
				mockDB.EXPECT().CaseUpdateContactID(ctx, propCustomerID, propCaseID, tt.contactID).Return(tt.updateErr)
				if tt.updateErr == nil {
					mockDB.EXPECT().CaseGetByID(ctx, propCaseID).Return(nil, tt.rereadErr)
				}
			}

			res, err := h.UpdateContact(ctx, propCustomerID, propCaseID, tt.contactID)
			if res != nil {
				t.Errorf("Wrong match. expect: nil case, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectText) {
				t.Fatalf("Wrong match. expect: %q in error, got: %v", tt.expectText, err)
			}
			if got := stderrors.Is(err, errPropagationBoom); got != tt.expectIsWrapped {
				t.Errorf("Wrong match. expect: errors.Is %v, got: %v", tt.expectIsWrapped, got)
			}
		})
	}
}

func Test_Create_PeerRequired(t *testing.T) {
	tests := []struct {
		name string
		peer commonaddress.Address
	}{
		{name: "empty type", peer: commonaddress.Address{Target: "+155****0001"}},
		{name: "empty target", peer: commonaddress.Address{Type: commonaddress.TypeTel}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, _, _, _ := newPropagationHandler(t)

			res, err := h.Create(context.Background(), propCustomerID, commonaddress.Address{}, tt.peer, "call", "", "", "")
			if res != nil {
				t.Errorf("Wrong match. expect: nil case, got: %v", res)
			}

			var ve *cerrors.VoipbinError
			if !stderrors.As(err, &ve) {
				t.Fatalf("Wrong match. expect: *VoipbinError, got: %T (%v)", err, err)
			}
			if ve.Status != cerrors.StatusInvalidArgument || ve.Reason != "CASE_PEER_REQUIRED" {
				t.Errorf("Wrong match. expect: InvalidArgument CASE_PEER_REQUIRED, got: %v %s", ve.Status, ve.Reason)
			}
		})
	}
}

func Test_Create_InsertError(t *testing.T) {
	h, mockDB, _, _, mockUtil := newPropagationHandler(t)
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-0000000000c1"))
	mockDB.EXPECT().CaseInsert(ctx, gomock.Any()).Return(errPropagationBoom)

	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"}
	res, err := h.Create(ctx, propCustomerID, commonaddress.Address{}, peer, "call", "", "", "")
	if res != nil {
		t.Errorf("Wrong match. expect: nil case, got: %v", res)
	}
	if err != errPropagationBoom { //nolint:errorlint // an unclassified insert failure must be returned unchanged
		t.Errorf("Wrong match. expect: injected error unchanged, got: %v", err)
	}
}

func Test_NewCaseHandler_InjectsDependencies(t *testing.T) {
	mc := gomock.NewController(t)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewCaseHandler(mockReq, mockDB, mockNotify)

	h, ok := res.(*caseHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *caseHandler, got: %T", res)
	}
	if h.db != dbhandler.DBHandler(mockDB) || h.reqHandler != requesthandler.RequestHandler(mockReq) || h.notifyHandler != notifyhandler.NotifyHandler(mockNotify) {
		t.Errorf("Wrong match. expect: injected dependencies, got: %+v", h)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. expect: non-nil utilHandler")
	}
}
