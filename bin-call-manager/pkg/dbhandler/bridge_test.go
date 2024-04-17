package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/cachehandler"

	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_BridgeCreate(t *testing.T) {

	type test struct {
		name   string
		bridge *bridge.Bridge

		responseCurTime string
		expectRes       *bridge.Bridge
	}

	tests := []test{
		{
			"test normal",
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
			},

			"2020-04-18 03:22:17.995000",
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
				ChannelIDs: []string{},
				TMCreate:   "2020-04-18 03:22:17.995000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
			},
		},
		{
			"reference type call",
			&bridge.Bridge{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "36d8b0be-9316-11ea-b829-6be92ca1faee",
				ReferenceType: bridge.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("23c83b3e-9316-11ea-91c3-ef8d90e0ec42"),
			},

			"2020-04-18 03:22:17.995000",
			&bridge.Bridge{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "36d8b0be-9316-11ea-b829-6be92ca1faee",
				ChannelIDs:    []string{},
				ReferenceType: bridge.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("23c83b3e-9316-11ea-91c3-ef8d90e0ec42"),
				TMCreate:      "2020-04-18 03:22:17.995000",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
			},
		},
		{
			"reference type conference",
			&bridge.Bridge{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "5149007a-9316-11ea-9de0-5f9cb2e8c235",
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("560448b8-9316-11ea-a651-b78c9ee8e874"),
			},

			"2020-04-18 03:22:17.995000",
			&bridge.Bridge{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "5149007a-9316-11ea-9de0-5f9cb2e8c235",
				ChannelIDs:    []string{},
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("560448b8-9316-11ea-a651-b78c9ee8e874"),
				TMCreate:      "2020-04-18 03:22:17.995000",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BridgeSet(ctx, gomock.Any())
			if err := h.BridgeCreate(ctx, tt.bridge); err != nil {
				t.Errorf("Wrong match. BridgeCreate expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BridgeGet(ctx, tt.bridge.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BridgeSet(ctx, gomock.Any())
			res, err := h.BridgeGet(ctx, tt.bridge.ID)
			if err != nil {
				t.Errorf("Wrong match. BridgeGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BridgeEnd(t *testing.T) {

	tests := []struct {
		name   string
		bridge *bridge.Bridge

		responseCurTime string
		expectRes       *bridge.Bridge
	}{
		{
			"test normal",
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "208a5bbe-8ee3-11ea-b267-174c3bd0a842",
			},

			"2020-04-18 05:22:17.995000",
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "208a5bbe-8ee3-11ea-b267-174c3bd0a842",
				ChannelIDs: []string{},
				TMCreate:   "2020-04-18 05:22:17.995000",
				TMUpdate:   "2020-04-18 05:22:17.995000",
				TMDelete:   "2020-04-18 05:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			if err := h.BridgeCreate(context.Background(), tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			if err := h.BridgeEnd(context.Background(), tt.bridge.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			res, err := h.BridgeGet(context.Background(), tt.bridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
