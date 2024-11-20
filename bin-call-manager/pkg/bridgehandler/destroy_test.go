package bridgehandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/dbhandler"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"
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
