package bridgehandler

import (
	"context"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
)

func Test_Destroy(t *testing.T) {

	type test struct {
		name string

		id string

		responseBridge *bridge.Bridge
	}

	tests := []test{
		{
			"normal",

			"469a5caf-9f1c-46af-bb2e-ded50fe1c3d0",

			&bridge.Bridge{
				ID: "469a5caf-9f1c-46af-bb2e-ded50fe1c3d0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := bridgeHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).Return(tt.responseBridge, nil)
			mockReq.EXPECT().AstBridgeDelete(ctx, tt.responseBridge.AsteriskID, tt.responseBridge.ID).Return(nil)

			if errDestroy := h.Destroy(ctx, tt.id); errDestroy != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDestroy)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
