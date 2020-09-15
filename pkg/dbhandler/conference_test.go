package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/cachehandler"
)

func TestConferenceCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		conference       *conference.Conference
		expectConference *conference.Conference
	}

	tests := []test{
		{
			"type conference",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
			},
			&conference.Conference{
				ID:     uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
			},
		},
		{
			"added user ID",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("132d3c9e-f08f-11ea-8ed9-6f27c201eff3"),
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
			},
			&conference.Conference{
				ID:     uuid.FromStringOrNil("132d3c9e-f08f-11ea-8ed9-6f27c201eff3"),
				UserID: 1,
				Type:   conference.TypeConference,
				Name:   "test type conference",
				Detail: "test type conference detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
			if err := h.ConferenceCreate(context.Background(), tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferenceSet(gomock.Any(), gomock.Any())
			res, err := h.ConferenceGet(context.Background(), tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectConference, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectConference, res)
			}
		})
	}
}
