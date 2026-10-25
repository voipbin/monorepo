package listenhandler

import (
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

func Test_errorResponse_typedNilPointer(t *testing.T) {
	// a typed nil *VoipbinError matches errors.As but can not be converted to a response.
	var ve *cerrors.VoipbinError

	res := errorResponse(ve)
	if res == nil || res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %v", http.StatusInternalServerError, res)
	}
}

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
	mockCampaign := campaignhandler.NewMockCampaignHandler(mc)
	mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)

	res := NewListenHandler(mockSock, mockOutplan, mockCampaign, mockCampaigncall)
	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *listenHandler, got: %T", res)
	}

	if h.sockHandler != mockSock || h.outplanHandler != mockOutplan || h.campaignHandler != mockCampaign || h.campaigncallHandler != mockCampaigncall {
		t.Errorf("Wrong match. the handlers must be wired to the matching fields")
	}
}

func Test_Run(t *testing.T) {
	tests := []struct {
		name string

		queueCreateErr error
		consumeErr     error

		expectErr bool
	}{
		{"normal", nil, nil, false},
		{"consume failure is logged only", nil, fmt.Errorf("consume failed"), false},
		{"queue create failure", fmt.Errorf("queue failed"), nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &listenHandler{sockHandler: mockSock}

			consumed := make(chan struct{})
			mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(tt.queueCreateErr)
			if tt.queueCreateErr == nil {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "campaign-manager", false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
						close(consumed)
						return tt.consumeErr
					})
			}

			err := h.Run("test-queue", "")
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}

			if tt.queueCreateErr == nil {
				<-consumed
			}
		})
	}
}
