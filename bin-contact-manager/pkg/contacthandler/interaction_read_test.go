package contacthandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

func Test_InteractionList_ByPeer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := contactHandler{
		utilHandler: mockUtil,
		db:          mockDB,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6")

	expectedRes := &tmpeerevent.PeerEventListResponse{
		Result: []*tmpeerevent.PeerEvent{
			{
				Timestamp:  time.Now(),
				CustomerID: customerID,
				Publisher:  "call",
			},
		},
		NextPageToken: "",
	}

	mockReq.EXPECT().TimelineV1PeerEventList(ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *tmpeerevent.PeerEventListRequest) (*tmpeerevent.PeerEventListResponse, error) {
			if req.CustomerID != customerID {
				t.Errorf("wrong customer id. expect: %v, got: %v", customerID, req.CustomerID)
			}
			if len(req.PeerAddresses) != 1 || req.PeerAddresses[0].Type != commonaddress.TypeTel || req.PeerAddresses[0].Target != "+821100000001" {
				t.Errorf("wrong peer addresses. got: %v", req.PeerAddresses)
			}
			return expectedRes, nil
		},
	)

	res, nextToken, err := h.InteractionList(ctx, customerID, 10, "", string(commonaddress.TypeTel), "+821100000001", uuid.Nil, uuid.Nil, time.Time{})
	if err != nil {
		t.Errorf("Test_InteractionList_ByPeer got error: %v", err)
	}
	if len(res) != 1 {
		t.Errorf("Test_InteractionList_ByPeer wrong result count. expect: 1, got: %d", len(res))
	}
	if nextToken != "" {
		t.Errorf("Test_InteractionList_ByPeer wrong next token. expect: empty, got: %s", nextToken)
	}
}

func Test_InteractionList_ByContact(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := contactHandler{
		utilHandler: mockUtil,
		db:          mockDB,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6")
	contactID := uuid.FromStringOrNil("b082d59c-2a00-11ee-8fb1-8bbf141432f6")

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		Addresses: []contact.Address{
			{
				Address: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},
			{
				Address: commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "test@test.com",
				},
			},
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockReq.EXPECT().TimelineV1PeerEventList(ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *tmpeerevent.PeerEventListRequest) (*tmpeerevent.PeerEventListResponse, error) {
			if len(req.PeerAddresses) != 2 {
				t.Errorf("wrong peer addresses count. expect: 2, got: %d", len(req.PeerAddresses))
			}
			return &tmpeerevent.PeerEventListResponse{Result: []*tmpeerevent.PeerEvent{}}, nil
		},
	)

	_, _, err := h.InteractionList(ctx, customerID, 10, "", "", "", contactID, uuid.Nil, time.Time{})
	if err != nil {
		t.Errorf("Test_InteractionList_ByContact got error: %v", err)
	}
}

func Test_InteractionList_ByAddress(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := contactHandler{
		utilHandler: mockUtil,
		db:          mockDB,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6")
	addressID := uuid.FromStringOrNil("c082d59c-2a00-11ee-8fb1-8bbf141432f6")

	responseAddress := &contact.Address{
		Address: commonaddress.Address{
			Type:   commonaddress.TypeTel,
			Target: "+821100000001",
		},
		ID:         addressID,
		CustomerID: customerID,
	}

	mockDB.EXPECT().AddressGet(ctx, customerID, addressID).Return(responseAddress, nil)
	mockReq.EXPECT().TimelineV1PeerEventList(ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *tmpeerevent.PeerEventListRequest) (*tmpeerevent.PeerEventListResponse, error) {
			if len(req.PeerAddresses) != 1 || req.PeerAddresses[0].Target != "+821100000001" {
				t.Errorf("wrong peer addresses. got: %v", req.PeerAddresses)
			}
			return &tmpeerevent.PeerEventListResponse{Result: []*tmpeerevent.PeerEvent{}}, nil
		},
	)

	_, _, err := h.InteractionList(ctx, customerID, 10, "", "", "", uuid.Nil, addressID, time.Time{})
	if err != nil {
		t.Errorf("Test_InteractionList_ByAddress got error: %v", err)
	}
}

func Test_InteractionList_NoFilter_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := contactHandler{
		utilHandler: mockUtil,
		db:          mockDB,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6")

	// No filters, and since is also zero -- peer_events requires >=1 address filter.
	_, _, err := h.InteractionList(ctx, customerID, 10, "", "", "", uuid.Nil, uuid.Nil, time.Time{})
	if err == nil {
		t.Errorf("Test_InteractionList_NoFilter_Error expected error, got nil")
	}
}

func Test_InteractionList_SinceOnly_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := contactHandler{
		utilHandler: mockUtil,
		db:          mockDB,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6")

	// since-only (unfiltered mode) is no longer supported -- peer_events has
	// no unfiltered read path.
	_, _, err := h.InteractionList(ctx, customerID, 10, "", "", "", uuid.Nil, uuid.Nil, time.Now().Add(-24*time.Hour))
	if err == nil {
		t.Errorf("Test_InteractionList_SinceOnly_Error expected error, got nil")
	}
}
