package servicehandler

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
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
		userPerm uint64
	}

	tests := []test{
		{
			"normal",
			"test username",
			"test userpass",
			uint64(user.PermissionNone),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockDB.EXPECT().UserGetByUsername(gomock.Any(), tt.userName).Return(nil, fmt.Errorf("not found"))
			mockDB.EXPECT().UserCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().UserGetByUsername(gomock.Any(), tt.userName)

			_, err := h.UserCreate(tt.userName, tt.userPass, tt.userPerm)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}
