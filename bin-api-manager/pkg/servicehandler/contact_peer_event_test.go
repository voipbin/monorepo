package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_PeerEventList_ByPeerType(t *testing.T) {
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
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	expectRes := []*tmpeerevent.PeerEvent{
		{EventType: "call_hangup"},
	}
	peerAddress := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234567"}

	mockReq.EXPECT().TimelineV1PeerEventList(ctx, &tmpeerevent.PeerEventListRequest{
		CustomerID:    a.CustomerID,
		PeerAddresses: []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}},
		PageToken:     "",
		PageSize:      10,
	}).Return(&tmpeerevent.PeerEventListResponse{Result: expectRes, NextPageToken: "next-token"}, nil)

	res, next, err := h.PeerEventList(ctx, a, uuid.Nil, peerAddress, "", 10)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectRes, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
	if next != "next-token" {
		t.Errorf("Wrong next page token. expect: next-token, got: %v", next)
	}
}

func Test_PeerEventList_ByContactID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	contactID := uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseContact := &cmcontact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		Addresses: []cmcontact.Address{
			{Address: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234567"}},
			{Address: commonaddress.Address{Type: commonaddress.TypeEmail, Target: "test@example.com"}},
		},
	}

	mockReq.EXPECT().ContactV1ContactGet(ctx, contactID).Return(responseContact, nil)

	expectRes := []*tmpeerevent.PeerEvent{{EventType: "call_hangup"}}
	mockReq.EXPECT().TimelineV1PeerEventList(ctx, &tmpeerevent.PeerEventListRequest{
		CustomerID: customerID,
		PeerAddresses: []commonaddress.Address{
			{Type: commonaddress.TypeTel, Target: "+15551234567"},
			{Type: commonaddress.TypeEmail, Target: "test@example.com"},
		},
		PageToken: "",
		PageSize:  10,
	}).Return(&tmpeerevent.PeerEventListResponse{Result: expectRes, NextPageToken: ""}, nil)

	res, _, err := h.PeerEventList(ctx, a, contactID, nil, "", 10)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectRes, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_PeerEventList_ByContactID_ZeroAddresses(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	contactID := uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseContact := &cmcontact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		Addresses: []cmcontact.Address{},
	}

	mockReq.EXPECT().ContactV1ContactGet(ctx, contactID).Return(responseContact, nil)
	// No TimelineV1PeerEventList call expected: zero addresses short-circuits.

	res, next, err := h.PeerEventList(ctx, a, contactID, nil, "", 10)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected nil result, got: %v", res)
	}
	if next != "" {
		t.Errorf("Expected empty next page token, got: %v", next)
	}
}

func Test_PeerEventList_ByContactID_CrossTenant(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	// Contact belongs to a DIFFERENT customer.
	responseContact := &cmcontact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("other-customer-11ee-97b2-cfe7337b701c"),
		},
	}

	mockReq.EXPECT().ContactV1ContactGet(ctx, contactID).Return(responseContact, nil)

	_, _, err := h.PeerEventList(ctx, a, contactID, nil, "", 10)
	if err != serviceerrors.ErrNotFound {
		t.Errorf("Expected ErrNotFound (anti-enumeration), got: %v", err)
	}
}

func Test_PeerEventList_NoFilter(t *testing.T) {
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
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	_, _, err := h.PeerEventList(ctx, a, uuid.Nil, nil, "", 10)
	if err == nil {
		t.Fatal("Expected error when no filter is provided, got nil")
	}
}

func Test_PeerEventList_PermissionDenied(t *testing.T) {
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
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})

	peerAddress := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234567"}
	_, _, err := h.PeerEventList(ctx, a, uuid.Nil, peerAddress, "", 10)
	if err != serviceerrors.ErrPermissionDenied {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

func Test_ServiceAgentPeerEventList_ByPeerType(t *testing.T) {
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
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})

	expectRes := []*tmpeerevent.PeerEvent{{EventType: "call_hangup"}}
	peerAddress := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234567"}
	mockReq.EXPECT().TimelineV1PeerEventList(ctx, &tmpeerevent.PeerEventListRequest{
		CustomerID:    a.CustomerID,
		PeerAddresses: []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15551234567"}},
		PageToken:     "",
		PageSize:      10,
	}).Return(&tmpeerevent.PeerEventListResponse{Result: expectRes}, nil)

	res, _, err := h.ServiceAgentPeerEventList(ctx, a, uuid.Nil, peerAddress, "", 10)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectRes, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_ServiceAgentPeerEventList_NotAgent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	// A non-agent identity (e.g. an accesskey identity) should be rejected.
	a := &auth.AuthIdentity{}

	peerAddress := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234567"}
	_, _, err := h.ServiceAgentPeerEventList(ctx, a, uuid.Nil, peerAddress, "", 10)
	if err != serviceerrors.ErrAuthenticationRequired {
		t.Errorf("Expected ErrAuthenticationRequired, got: %v", err)
	}
}
