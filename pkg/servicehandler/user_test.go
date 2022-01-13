package servicehandler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
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

		user *user.User

		username   string
		password   string
		userName   string
		detail     string
		permission user.Permission

		res *umuser.User
	}{
		{
			"normal",

			&user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},

			"test1",
			"test userpass 1",
			"name1",
			"detail1",
			user.PermissionNone,

			&umuser.User{
				Username:   "test1",
				Permission: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().UMV1UserCreate(gomock.Any(), 30, tt.username, tt.password, tt.userName, tt.detail, umuser.Permission(tt.permission)).Return(tt.res, nil)

			_, err := h.UserCreate(tt.user, tt.username, tt.password, tt.userName, tt.detail, tt.permission)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestUserDelete(t *testing.T) {
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

		user *user.User

		id uint64

		res *umuser.User
	}{
		{
			"normal",

			&user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},

			1,

			&umuser.User{
				Username:   "test1",
				Permission: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().UMV1UserDelete(gomock.Any(), tt.id).Return(nil)

			if err := h.UserDelete(tt.user, tt.id); err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestUserGet(t *testing.T) {
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

		user *user.User

		id uint64

		res *umuser.User
	}{
		{
			"normal",

			&user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},

			1,

			&umuser.User{
				Username:   "test1",
				Permission: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().UMV1UserGet(gomock.Any(), tt.id).Return(tt.res, nil)

			_, err := h.UserGet(tt.user, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestUserGets(t *testing.T) {
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

		user *user.User

		size  uint64
		token string

		res []umuser.User
	}{
		{
			"normal",

			&user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},

			10,
			"2021-03-02%2003%3A23%3A20.995000",

			[]umuser.User{
				{
					Username:   "test1",
					Permission: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().UMV1UserGets(gomock.Any(), tt.token, tt.size).Return(tt.res, nil)

			_, err := h.UserGets(tt.user, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestUserUpdate(t *testing.T) {
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

		user *user.User

		id       uint64
		userName string
		detail   string

		res []umuser.User
	}{
		{
			"normal",

			&user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},

			1,
			"name1",
			"detail1",

			[]umuser.User{
				{
					Username:   "test1",
					Permission: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().UMV1UserGet(gomock.Any(), tt.user.ID).Return(&umuser.User{Permission: umuser.PermissionAdmin}, nil)
			mockReq.EXPECT().UMV1UserUpdateBasicInfo(gomock.Any(), tt.id, tt.userName, tt.detail).Return(nil)

			if err := h.UserUpdate(tt.user, tt.id, tt.userName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestUserUpdatePassword(t *testing.T) {
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

		user *user.User

		id       uint64
		password string

		res []umuser.User
	}{
		{
			"normal",

			&user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},

			1,
			"password1",

			[]umuser.User{
				{
					Username:   "test1",
					Permission: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().UMV1UserGet(gomock.Any(), tt.user.ID).Return(&umuser.User{Permission: umuser.PermissionAdmin}, nil)
			mockReq.EXPECT().UMV1UserUpdatePassword(gomock.Any(), 30, tt.id, tt.password).Return(nil)

			if err := h.UserUpdatePassword(tt.user, tt.id, tt.password); err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestUserUpdatePermission(t *testing.T) {
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

		user *user.User

		id         uint64
		permission user.Permission

		res []umuser.User
	}{
		{
			"normal",

			&user.User{
				ID:         1,
				Permission: user.PermissionAdmin,
			},

			1,
			user.PermissionAdmin,

			[]umuser.User{
				{
					Username:   "test1",
					Permission: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().UMV1UserUpdatePermission(gomock.Any(), tt.id, umuser.Permission(tt.permission)).Return(nil)

			if err := h.UserUpdatePermission(tt.user, tt.id, tt.permission); err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}
