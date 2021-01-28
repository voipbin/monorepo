package servicehandler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/cmrecording"
)

func TestRecordingfileGet(t *testing.T) {
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
		id   string

		response   *cmrecording.Recording
		responseST string
		expectRes  string
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			"59a394e4-610e-11eb-b8c6-aff7333845f1",

			&cmrecording.Recording{
				ID:     "call_25b4a290-0f25-4b50-87bd-7174638ac906_2021-01-26T02:17:05Z",
				UserID: 1,
			},
			"test.com/downloadlink.wav",
			"test.com/downloadlink.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().CMRecordingGet(tt.id).Return(tt.response, nil)
			mockReq.EXPECT().STRecordingGet(tt.id+".wav").Return(tt.responseST, nil)

			res, err := h.RecordingfileGet(tt.user, tt.id)

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
