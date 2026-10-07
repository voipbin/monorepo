package subscribehandler

import (
	"context"
	"fmt"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
)

func Test_processEventCMCallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		callID uuid.UUID

		responseCampaigncall *campaigncall.Campaigncall
	}{
		{
			"normal",
			&sock.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"62a54c96-c46c-11ec-aff0-ebddfa5d9bc4"}`),
			},

			uuid.FromStringOrNil("62a54c96-c46c-11ec-aff0-ebddfa5d9bc4"),

			&campaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e6cca6a4-c46c-11ec-8175-3fd04df5a0dc"),
				},
				CampaignID: uuid.FromStringOrNil("f4f81330-c46c-11ec-845b-634ec638de76"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := subscribeHandler{
				campaignHandler:     mockCampaign,
				campaigncallHandler: mockCampaigncall,
			}

			mockCampaigncall.EXPECT().GetByReferenceID(gomock.Any(), tt.callID).Return(tt.responseCampaigncall, nil)
			mockCampaigncall.EXPECT().EventHandleReferenceCallHungup(gomock.Any(), gomock.Any(), tt.responseCampaigncall).Return(tt.responseCampaigncall, nil)
			mockCampaign.EXPECT().EventHandleReferenceCallHungup(gomock.Any(), tt.responseCampaigncall.CampaignID).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func Test_processEventCMCallHangup_errorAndSkip(t *testing.T) {

	callID := uuid.FromStringOrNil("62a54c96-c46c-11ec-aff0-ebddfa5d9bc4")
	campaignID := uuid.FromStringOrNil("f4f81330-c46c-11ec-845b-634ec638de76")

	event := &sock.Event{
		Publisher: "call-manager",
		Type:      cmcall.EventTypeCallHangup,
		DataType:  "application/json",
		Data:      []byte(`{"id":"62a54c96-c46c-11ec-aff0-ebddfa5d9bc4"}`),
	}

	newCampaigncall := func() *campaigncall.Campaigncall {
		return &campaigncall.Campaigncall{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("e6cca6a4-c46c-11ec-8175-3fd04df5a0dc"),
			},
			CampaignID: campaignID,
		}
	}

	tests := []struct {
		name string

		event *sock.Event

		// the expectation setup of one case: the loaded campaigncall, and what the handlers return.
		getErr           error
		handleReturnsCC  bool
		handleErr        error
		expectCampaign   bool
		campaignErr      error
		expectErr        bool
		expectHandleCall bool
	}{
		{
			name:  "campaigncall lookup fails, nothing else is called",
			event: event,

			getErr: fmt.Errorf("not found"),
		},
		{
			name:  "campaigncall handler fails, the campaign handler still runs with the loaded campaign id",
			event: event,

			expectHandleCall: true,
			handleErr:        fmt.Errorf("could not done"),
			expectCampaign:   true,
		},
		{
			name:  "campaigncall is skipped as already done, the campaign handler still runs",
			event: event,

			expectHandleCall: true,
			handleReturnsCC:  true,
			expectCampaign:   true,
		},
		{
			name:  "campaign handler fails, the event is still handled",
			event: event,

			expectHandleCall: true,
			handleReturnsCC:  true,
			expectCampaign:   true,
			campaignErr:      fmt.Errorf("could not stop"),
		},
		{
			name: "the data is not valid json",
			event: &sock.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{bad`),
			},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := subscribeHandler{
				campaignHandler:     mockCampaign,
				campaigncallHandler: mockCampaigncall,
			}

			ctx := context.Background()
			cc := newCampaigncall()

			if tt.expectErr {
				// the data is not valid: no handler is called.
			} else if tt.getErr != nil {
				mockCampaigncall.EXPECT().GetByReferenceID(ctx, callID).Return(nil, tt.getErr)
			} else {
				mockCampaigncall.EXPECT().GetByReferenceID(ctx, callID).Return(cc, nil)
			}

			if tt.expectHandleCall {
				var handleRes *campaigncall.Campaigncall
				if tt.handleReturnsCC {
					handleRes = cc
				}
				mockCampaigncall.EXPECT().EventHandleReferenceCallHungup(ctx, gomock.Any(), cc).Return(handleRes, tt.handleErr)
			}
			if tt.expectCampaign {
				mockCampaign.EXPECT().EventHandleReferenceCallHungup(ctx, campaignID).Return(tt.campaignErr)
			}

			err := h.processEventCMCallHungup(ctx, tt.event)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
