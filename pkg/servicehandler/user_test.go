package servicehandler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"
	umuser "gitlab.com/voipbin/bin-manager/user-manager.git/models/user"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestUserCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	tests := []struct {
		name string

		userName string
		userPass string
		userPerm uint64

		res *umuser.User
	}{
		{
			"normal",
			"test1",
			"test userpass 1",
			uint64(user.PermissionNone),

			&umuser.User{
				Username:   "test1",
				Permission: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().UMV1UserCreate(gomock.Any(), tt.userName, tt.userPass, tt.userPerm).Return(tt.res, nil)

			_, err := h.UserCreate(tt.userName, tt.userPass, tt.userPerm)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}
