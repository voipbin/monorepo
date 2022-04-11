package requesthandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/golang/mock/gomock"
	umuser "gitlab.com/voipbin/bin-manager/user-manager.git/models/user"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestUMV1UserGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes []umuser.User
	}{
		{
			"normal",

			"2021-03-02 03:23:20.995000",
			10,

			"bin-manager.user-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/users?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":1,"username":"test1","name":"test user 1","detail":"test user 1 detail","permission":1},{"id":2,"username":"test2","name":"test user 2","detail":"test user 2 detail","permission":1}]`),
			},
			[]umuser.User{
				{
					ID:           1,
					Username:     "test1",
					PasswordHash: "",
					Name:         "test user 1",
					Detail:       "test user 1 detail",
					Permission:   1,
					TMCreate:     "",
					TMUpdate:     "",
					TMDelete:     "",
				},
				{
					ID:           2,
					Username:     "test2",
					PasswordHash: "",
					Name:         "test user 2",
					Detail:       "test user 2 detail",
					Permission:   1,
					TMCreate:     "",
					TMUpdate:     "",
					TMDelete:     "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.UMV1UserGets(ctx, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestUMV1UserGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		userID uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *umuser.User
	}{
		{
			"normal",

			1,

			"bin-manager.user-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/users/1",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":1,"username":"test1","name":"test user 1","detail":"test user 1 detail","permission":1}`),
			},
			&umuser.User{
				ID:           1,
				Username:     "test1",
				PasswordHash: "",
				Name:         "test user 1",
				Detail:       "test user 1 detail",
				Permission:   1,
				TMCreate:     "",
				TMUpdate:     "",
				TMDelete:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.UMV1UserGet(ctx, tt.userID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestUMV1UserDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		userID uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			1,

			"bin-manager.user-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/users/1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.UMV1UserDelete(ctx, tt.userID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestUMV1UserCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		username   string
		password   string
		permission umuser.Permission
		userName   string
		detail     string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *umuser.User
	}{
		{
			"normal",

			"test1",
			"testpassword",
			1,
			"test1",
			"detail1",

			"bin-manager.user-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/users",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"username":"test1","password":"testpassword","name":"test1","detail":"detail1","permission":1}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":1,"username":"test1","name":"test user 1","detail":"test user 1 detail","permission":1}`),
			},
			&umuser.User{
				ID:           1,
				Username:     "test1",
				PasswordHash: "",
				Name:         "test user 1",
				Detail:       "test user 1 detail",
				Permission:   1,
				TMCreate:     "",
				TMUpdate:     "",
				TMDelete:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.UMV1UserCreate(ctx, requestTimeoutDefault, tt.username, tt.password, tt.userName, tt.detail, tt.permission)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestUMV1UserLogin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		username string
		password string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *umuser.User
	}{
		{
			"normal",

			"test1",
			"testpassword",

			"bin-manager.user-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/users/test1/login",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"password":"testpassword"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":1,"username":"test1","name":"test user 1","detail":"test user 1 detail","permission":1}`),
			},
			&umuser.User{
				ID:           1,
				Username:     "test1",
				PasswordHash: "",
				Name:         "test user 1",
				Detail:       "test user 1 detail",
				Permission:   1,
				TMCreate:     "",
				TMUpdate:     "",
				TMDelete:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.UMV1UserLogin(ctx, requestTimeoutDefault, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestUMV1UserUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id       uint64
		userName string
		detail   string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			1,
			"test1",
			"detail1",

			"bin-manager.user-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/users/1",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test1","detail":"detail1"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.UMV1UserUpdateBasicInfo(ctx, tt.id, tt.userName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestUMV1UserUpdatePassword(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id       uint64
		password string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			1,
			"password1",

			"bin-manager.user-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/users/1/password",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"password":"password1"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.UMV1UserUpdatePassword(ctx, requestTimeoutDefault, tt.id, tt.password); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestUMV1UserUpdatePermission(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id         uint64
		permission umuser.Permission

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			1,
			umuser.PermissionNone,

			"bin-manager.user-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/users/1/permission",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"permission":0}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.UMV1UserUpdatePermission(ctx, tt.id, tt.permission); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
