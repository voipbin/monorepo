package servicehandler

import (
	"context"
	"errors"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_CaseMessageSend_CaseNotFoundOrCrossTenant(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(nil, serviceerrors.ErrNotFound)

	_, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559876543", "hello")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func Test_CaseMessageSend_CaseClosed(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusClosed,
		}, nil)

	_, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559876543", "hello")
	if !errors.Is(err, serviceerrors.ErrCaseClosed) {
		t.Errorf("Expected ErrCaseClosed, got: %v", err)
	}
}

func Test_CaseMessageSend_DestinationBindingFailure_NoContactID(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  nil,
			PeerTarget: "+15550001111",
		}, nil)

	_, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559999999", "hello")
	if !errors.Is(err, serviceerrors.ErrCaseDestinationNotAssociated) {
		t.Errorf("Expected ErrCaseDestinationNotAssociated, got: %v", err)
	}
}

func Test_CaseMessageSend_DestinationBindingFailure_HasContactID(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  &contactID,
			PeerTarget: "+15550001111",
		}, nil)

	mockReq.EXPECT().
		ContactV1AddressGet(ctx, contactID).
		Return([]cmcontact.Address{
			{Address: commonaddress.Address{Target: "+155****2222"}},
			{Address: commonaddress.Address{Target: "+155****3333"}},
		}, nil)

	_, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559999999", "hello")
	if !errors.Is(err, serviceerrors.ErrCaseDestinationNotAssociated) {
		t.Errorf("Expected ErrCaseDestinationNotAssociated, got: %v", err)
	}
}

// Test_CaseMessageSend_AntiOracle is the explicit anti-oracle test
// (design §4.5 step 2): both destination-binding failure sub-cases MUST
// produce the identical sentinel and identical error string -- a caller
// probing which sub-check failed must not be able to tell them apart.
func Test_CaseMessageSend_AntiOracle(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseIDNoContact := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	caseIDHasContact := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")
	contactID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	// Sub-case 1: no contact_id, destination != peer_target.
	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseIDNoContact).
		Return(&cmkase.Case{
			ID:         caseIDNoContact,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  nil,
			PeerTarget: "+15550001111",
		}, nil)
	_, err1 := h.CaseMessageSend(ctx, a, caseIDNoContact, "+15551234567", "+15559999999", "hello")

	// Sub-case 2: contact_id set, destination not in that contact's addresses.
	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseIDHasContact).
		Return(&cmkase.Case{
			ID:         caseIDHasContact,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  &contactID,
			PeerTarget: "+15550001111",
		}, nil)
	mockReq.EXPECT().
		ContactV1AddressGet(ctx, contactID).
		Return([]cmcontact.Address{
			{Address: commonaddress.Address{Target: "+155****2222"}},
		}, nil)
	_, err2 := h.CaseMessageSend(ctx, a, caseIDHasContact, "+15551234567", "+15559999999", "hello")

	if !errors.Is(err1, serviceerrors.ErrCaseDestinationNotAssociated) {
		t.Errorf("Expected err1 to be ErrCaseDestinationNotAssociated, got: %v", err1)
	}
	if !errors.Is(err2, serviceerrors.ErrCaseDestinationNotAssociated) {
		t.Errorf("Expected err2 to be ErrCaseDestinationNotAssociated, got: %v", err2)
	}
	if err1.Error() != err2.Error() {
		t.Errorf("Anti-oracle violation: err1 (%q) and err2 (%q) have different messages", err1.Error(), err2.Error())
	}
}

func Test_CaseMessageSend_SourceNotOwned(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  nil,
			PeerTarget: "+15559999999",
		}, nil)

	mockReq.EXPECT().
		NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
			nmnumber.FieldCustomerID: customerID,
			nmnumber.FieldNumber:     "+15551234567",
			nmnumber.FieldType:       nmnumber.TypeNormal,
			nmnumber.FieldStatus:     nmnumber.StatusActive,
			nmnumber.FieldDeleted:    false,
		}).
		Return([]nmnumber.Number{}, nil)

	_, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559999999", "hello")
	if !errors.Is(err, serviceerrors.ErrCaseSourceNotOwned) {
		t.Errorf("Expected ErrCaseSourceNotOwned, got: %v", err)
	}
}

// Test_CaseMessageSend_SourceFilterMismatch_WrongCustomer verifies the
// filter map itself does the right query -- the mock returns empty when
// invoked with the case's customer_id, proving a wrong-customer's number
// would be filtered out (round-17 correction: "wrong customer's number ...
// rejected" -- verified via the filter map, not a separate scenario, per
// the task's explicit allowance to do so as long as it's verified with a
// real test rather than assumed).
func Test_CaseMessageSend_SourceFilterMismatch_WrongCustomer(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  nil,
			PeerTarget: "+15559999999",
		}, nil)

	// The RPC is expected to be called with FieldCustomerID=customerID
	// (the case's own customer). If the code accidentally queried with a
	// different customer_id, gomock's exact-match EXPECT() below would
	// fail to match and the test would report an unexpected call, proving
	// the filter is correctly scoped to this case's customer.
	mockReq.EXPECT().
		NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
			nmnumber.FieldCustomerID: customerID,
			nmnumber.FieldNumber:     "+15551234567",
			nmnumber.FieldType:       nmnumber.TypeNormal,
			nmnumber.FieldStatus:     nmnumber.StatusActive,
			nmnumber.FieldDeleted:    false,
		}).
		Return([]nmnumber.Number{}, nil)

	_, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559999999", "hello")
	if !errors.Is(err, serviceerrors.ErrCaseSourceNotOwned) {
		t.Errorf("Expected ErrCaseSourceNotOwned, got: %v", err)
	}
}

// Test_CaseMessageSend_FailOpen is the explicit fail-open test (design
// §4.5 step 2 / "Metadata write, FAIL-OPEN"): mocking
// ConversationV1ConversationUpdateMetadata to error must NOT abort the
// send -- the message still gets sent and the overall call still
// succeeds.
func Test_CaseMessageSend_FailOpen(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	conversationID := uuid.FromStringOrNil("55555555-0000-0000-0000-000000000005")
	messageID := uuid.FromStringOrNil("66666666-0000-0000-0000-000000000006")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  nil,
			PeerTarget: "+15559999999",
			PeerType:   commonaddress.TypeTel,
		}, nil)

	mockReq.EXPECT().
		NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
			nmnumber.FieldCustomerID: customerID,
			nmnumber.FieldNumber:     "+15551234567",
			nmnumber.FieldType:       nmnumber.TypeNormal,
			nmnumber.FieldStatus:     nmnumber.StatusActive,
			nmnumber.FieldDeleted:    false,
		}).
		Return([]nmnumber.Number{{}}, nil)

	mockReq.EXPECT().
		ConversationV1ConversationGetOrCreateBySelfAndPeer(
			ctx, customerID, cvconversation.TypeMessage, "",
			commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234567"},
			commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15559999999"},
		).
		Return(&cvconversation.Conversation{
			Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		}, nil)

	mockReq.EXPECT().
		ConversationV1ConversationUpdateMetadata(ctx, conversationID, cvconversation.Metadata{ContactCaseID: &caseID}).
		Return(nil, errors.New("metadata write failed"))

	mockReq.EXPECT().
		ConversationV1MessageSend(ctx, conversationID, "hello", nil).
		Return(&cvmessage.Message{
			Identity:       commonidentity.Identity{ID: messageID, CustomerID: customerID},
			ConversationID: conversationID,
			Text:           "hello",
		}, nil)

	res, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559999999", "hello")
	if err != nil {
		t.Errorf("Expected success despite metadata-write failure (fail-open), got err: %v", err)
	}
	if res == nil {
		t.Errorf("Expected a result but got nil")
	}
}

// Test_CaseMessageSend_HappyPath and the case_id echo-read test share the
// same setup as fail-open above, but assert the metadata write succeeds
// and that it was called with THIS case's own ID -- the practical proof
// that Phase 1's read path would pick up a correctly-written hint.
func Test_CaseMessageSend_HappyPath_And_CaseIDEchoRead(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	conversationID := uuid.FromStringOrNil("55555555-0000-0000-0000-000000000005")
	messageID := uuid.FromStringOrNil("66666666-0000-0000-0000-000000000006")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  nil,
			PeerTarget: "+15559999999",
			PeerType:   commonaddress.TypeTel,
		}, nil)

	mockReq.EXPECT().
		NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
			nmnumber.FieldCustomerID: customerID,
			nmnumber.FieldNumber:     "+15551234567",
			nmnumber.FieldType:       nmnumber.TypeNormal,
			nmnumber.FieldStatus:     nmnumber.StatusActive,
			nmnumber.FieldDeleted:    false,
		}).
		Return([]nmnumber.Number{{}}, nil)

	mockReq.EXPECT().
		ConversationV1ConversationGetOrCreateBySelfAndPeer(
			ctx, customerID, cvconversation.TypeMessage, "",
			commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234567"},
			commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15559999999"},
		).
		Return(&cvconversation.Conversation{
			Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		}, nil)

	// This is the assertion that proves the write uses THIS case's own
	// ID as the metadata hint -- Metadata{ContactCaseID: &caseID} where
	// caseID is exactly the case_id this send request targeted, not any
	// other value. gomock's exact-value match on the caseID pointer
	// dereference (via reflect.DeepEqual on the pointee) fails the test
	// if a different case ID were ever passed here.
	mockReq.EXPECT().
		ConversationV1ConversationUpdateMetadata(ctx, conversationID, cvconversation.Metadata{ContactCaseID: &caseID}).
		Return(&cvconversation.Conversation{
			Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		}, nil)

	mockReq.EXPECT().
		ConversationV1MessageSend(ctx, conversationID, "hello", nil).
		Return(&cvmessage.Message{
			Identity:       commonidentity.Identity{ID: messageID, CustomerID: customerID},
			ConversationID: conversationID,
			Text:           "hello",
		}, nil)

	res, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559999999", "hello")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if res == nil {
		t.Fatalf("Expected result but got nil")
	}
	if res.ID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, res.ID)
	}
	if res.ConversationID != conversationID {
		t.Errorf("Expected conversation ID %s, got %s", conversationID, res.ConversationID)
	}
	if res.Text != "hello" {
		t.Errorf("Expected text %q, got %q", "hello", res.Text)
	}
}

func Test_CaseMessageSend_DirectAccessDenied(t *testing.T) {
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")})

	_, err := h.CaseMessageSend(ctx, a, caseID, "+15551234567", "+15559999999", "hello")
	if err != serviceerrors.ErrDirectAccessNotSupported {
		t.Errorf("Expected ErrDirectAccessNotSupported, got: %v", err)
	}
}

func Test_CaseMessageSend_PermissionDenied(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAgent,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			PeerTarget: "+155****9999",
		}, nil)

	_, err := h.CaseMessageSend(ctx, a, caseID, "+155****4567", "+155****9999", "hello")
	if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

// Test_CaseMessageSend_SelfAndPeerTypeMatch_WhatsApp is a regression test
// for a round-1 Phase 5 review defect: selfAddr.Type must match the
// case's PeerType (not be hardcoded to TypeTel), otherwise
// ConversationGetOrCreateBySelfAndPeer would never find an existing
// WhatsApp/LINE conversation (self.type/peer.type mismatch) and would
// create a spurious duplicate on every send.
func Test_CaseMessageSend_SelfAndPeerTypeMatch_WhatsApp(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	conversationID := uuid.FromStringOrNil("55555555-0000-0000-0000-000000000005")
	messageID := uuid.FromStringOrNil("66666666-0000-0000-0000-000000000006")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
			ContactID:  nil,
			PeerTarget: "whatsapp-peer-id-9999",
			PeerType:   commonaddress.TypeWhatsApp,
		}, nil)

	mockReq.EXPECT().
		NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
			nmnumber.FieldCustomerID: customerID,
			nmnumber.FieldNumber:     "whatsapp-business-id-4567",
			nmnumber.FieldType:       nmnumber.TypeNormal,
			nmnumber.FieldStatus:     nmnumber.StatusActive,
			nmnumber.FieldDeleted:    false,
		}).
		Return([]nmnumber.Number{{}}, nil)

	// The key assertion: BOTH self and peer addresses must carry
	// TypeWhatsApp (matching c.PeerType), not a hardcoded TypeTel on the
	// self side. gomock's exact-match EXPECT() below fails the test if
	// selfAddr.Type reverts to TypeTel.
	mockReq.EXPECT().
		ConversationV1ConversationGetOrCreateBySelfAndPeer(
			ctx, customerID, cvconversation.TypeWhatsApp, "",
			commonaddress.Address{Type: commonaddress.TypeWhatsApp, Target: "whatsapp-business-id-4567"},
			commonaddress.Address{Type: commonaddress.TypeWhatsApp, Target: "whatsapp-peer-id-9999"},
		).
		Return(&cvconversation.Conversation{
			Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		}, nil)

	mockReq.EXPECT().
		ConversationV1ConversationUpdateMetadata(ctx, conversationID, cvconversation.Metadata{ContactCaseID: &caseID}).
		Return(&cvconversation.Conversation{Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID}}, nil)

	mockReq.EXPECT().
		ConversationV1MessageSend(ctx, conversationID, "hello", nil).
		Return(&cvmessage.Message{
			Identity:       commonidentity.Identity{ID: messageID, CustomerID: customerID},
			ConversationID: conversationID,
			Text:           "hello",
		}, nil)

	res, err := h.CaseMessageSend(ctx, a, caseID, "whatsapp-business-id-4567", "whatsapp-peer-id-9999", "hello")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if res == nil {
		t.Errorf("Expected a result but got nil")
	}
}
