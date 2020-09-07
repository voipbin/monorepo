package servicehandler

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
)

func TestUserCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	type test struct {
		name string

		userName string
		userPass string
	}

	tests := []test{
		{
			"normal",
			"test username",
			"test userpass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := servicHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockDB.EXPECT().UserGetByUsername(gomock.Any(), tt.userName).Return(nil, fmt.Errorf("not found"))
			mockDB.EXPECT().UserCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().UserGetByUsername(gomock.Any(), tt.userName)

			_, err := h.UserCreate(tt.userName, tt.userPass)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}
