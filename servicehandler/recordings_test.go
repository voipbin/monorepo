package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/cmrecording"
)

func TestRecordingGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User

		size  uint64
		token string

		// response  *fmflow.Flow
		response  []cmrecording.Recording
		expectRes []*recording.Recording
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			10,
			"2020-10-20 01:00:00.995000",

			[]cmrecording.Recording{
				cmrecording.Recording{
					ID:     "call_25b4a290-0f25-4b50-87bd-7174638ac906_2021-01-26T02:17:05Z",
					UserID: 1,
				},
				cmrecording.Recording{
					ID:     "call_2f167946-b2b4-4370-94fa-d6c2c57c84da_2020-12-04T18:48:03Z",
					UserID: 1,
				},
			},

			[]*recording.Recording{
				&recording.Recording{
					ID:          "call_25b4a290-0f25-4b50-87bd-7174638ac906_2021-01-26T02:17:05Z",
					UserID:      1,
					ReferenceID: uuid.Nil,
				},
				&recording.Recording{
					ID:          "call_2f167946-b2b4-4370-94fa-d6c2c57c84da_2020-12-04T18:48:03Z",
					UserID:      1,
					ReferenceID: uuid.Nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().CMRecordingGets(tt.user.ID, tt.size, tt.token).Return(tt.response, nil)

			res, err := h.RecordingGets(tt.user, tt.size, tt.token)

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res[0])
			}
		})
	}
}
