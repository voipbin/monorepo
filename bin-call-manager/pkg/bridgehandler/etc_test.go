package bridgehandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_IsExist(t *testing.T) {

	type test struct {
		name string

		id string

		responseBridge *bridge.Bridge
		responseAstErr error
		responseEndErr error

		expectRes bool
	}

	tests := []test{
		{
			name: "normal",

			id: "8f1c7a4e-a7d1-11f0-9b3c-7f2d1e6a9c01",

			responseBridge: &bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "8f1c7a4e-a7d1-11f0-9b3c-7f2d1e6a9c01",
			},
			responseAstErr: nil,
			responseEndErr: nil,

			expectRes: true,
		},
		{
			name: "asterisk bridge does not exist",

			id: "9a2d8b5f-a7d1-11f0-8c4d-3b6e2f7a1d02",

			responseBridge: &bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "9a2d8b5f-a7d1-11f0-8c4d-3b6e2f7a1d02",
			},
			responseAstErr: fmt.Errorf("bridge not found"),
			responseEndErr: nil,

			expectRes: false,
		},
		{
			name: "asterisk bridge does not exist and delete failed",

			id: "a43e9c60-a7d1-11f0-a5e8-5c7f3a8b2e03",

			responseBridge: &bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "a43e9c60-a7d1-11f0-a5e8-5c7f3a8b2e03",
			},
			responseAstErr: fmt.Errorf("bridge not found"),
			responseEndErr: fmt.Errorf("could not end the bridge"),

			expectRes: false,
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

			// Get reads the bridge once. When the asterisk bridge is gone and
			// BridgeEnd succeeds, Delete reads it once more.
			getTimes := 1
			if tt.responseAstErr != nil && tt.responseEndErr == nil {
				getTimes = 2
			}
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).Return(tt.responseBridge, nil).Times(getTimes)

			if tt.responseAstErr != nil {
				mockReq.EXPECT().AstBridgeGet(ctx, tt.responseBridge.AsteriskID, tt.responseBridge.ID).Return(nil, tt.responseAstErr)
				mockDB.EXPECT().BridgeEnd(ctx, tt.responseBridge.ID).Return(tt.responseEndErr)
			} else {
				mockReq.EXPECT().AstBridgeGet(ctx, tt.responseBridge.AsteriskID, tt.responseBridge.ID).Return(&bridge.Bridge{}, nil)
			}

			res := h.IsExist(ctx, tt.id)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_IsExist_getFailed(t *testing.T) {

	type test struct {
		name string

		id string

		expectRes bool
	}

	tests := []test{
		{
			name: "bridge does not exist in the database",

			id: "b85fad71-a7d1-11f0-b7f9-8d1a4c9e3f04",

			expectRes: false,
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

			// A short deadline keeps the test fast.
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()

			// Block until the lookup context expires, then fail. After its
			// defaultDelayTimeout sleep the retry goroutine sees the expired
			// context and exits without another lookup, so BridgeGet is
			// called exactly once.
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).DoAndReturn(func(c context.Context, _ string) (*bridge.Bridge, error) {
				<-c.Done()
				return nil, fmt.Errorf("bridge not found")
			}).Times(1)

			res := h.IsExist(ctx, tt.id)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
