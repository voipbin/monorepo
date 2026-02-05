package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

func Test_MediaCreate(t *testing.T) {

	tests := []struct {
		name string

		media *media.Media

		responseCurTime *time.Time
		expectRes       *media.Media
	}{
		{
			"normal",

			&media.Media{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("77f3825a-eb9c-11ec-9fa6-ef743d81dea8"),
					CustomerID: uuid.FromStringOrNil("7a4fe890-eb9c-11ec-b171-cf5dc7a96ec5"),
				},

				Type:     media.TypeAudio,
				Filename: "testfilename.wav",
			},

			func() *time.Time { t := time.Date(2022, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }(),
			&media.Media{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("77f3825a-eb9c-11ec-9fa6-ef743d81dea8"),
					CustomerID: uuid.FromStringOrNil("7a4fe890-eb9c-11ec-b171-cf5dc7a96ec5"),
				},

				Type:     media.TypeAudio,
				Filename: "testfilename.wav",
				TMCreate: func() *time.Time { t := time.Date(2022, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }(),
				TMUpdate: nil,
				TMDelete: nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().MediaSet(gomock.Any(), gomock.Any())
			if err := h.MediaCreate(ctx, tt.media); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MediaGet(gomock.Any(), tt.media.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MediaSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.MediaGet(ctx, tt.media.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
