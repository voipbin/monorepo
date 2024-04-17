package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

func Test_MediaCreate(t *testing.T) {

	tests := []struct {
		name string

		media *media.Media
	}{
		{
			"normal",

			&media.Media{
				ID:         uuid.FromStringOrNil("77f3825a-eb9c-11ec-9fa6-ef743d81dea8"),
				CustomerID: uuid.FromStringOrNil("7a4fe890-eb9c-11ec-b171-cf5dc7a96ec5"),
				Type:       media.TypeAudio,
				Filename:   "testfilename.wav",
				TMCreate:   "2022-04-18 03:22:17.995000",
				TMUpdate:   "2022-04-18 03:22:17.995000",
				TMDelete:   DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

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

			if reflect.DeepEqual(res, tt.media) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.media, res)
			}
		})
	}
}
