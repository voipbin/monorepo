package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AuthLogin(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		responseAgent   *amagent.Agent
		responseCurTime string

		expectedRes string
	}{
		{
			name: "normal",

			username: "test@test.com",
			password: "testpassword",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6bc342d0-8aed-11ee-a07d-7bc7fee5a336"),
					CustomerID: uuid.FromStringOrNil("6c0ff198-8aed-11ee-8a04-474584947e03"),
				},
			},
			responseCurTime: "2023-11-19 09:29:11.763331118",
			expectedRes:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhZ2VudCI6eyJpZCI6IjZiYzM0MmQwLThhZWQtMTFlZS1hMDdkLTdiYzdmZWU1YTMzNiIsImN1c3RvbWVyX2lkIjoiNmMwZmYxOTgtOGFlZC0xMWVlLThhMDQtNDc0NTg0OTQ3ZTAzIiwidXNlcm5hbWUiOiIiLCJwYXNzd29yZF9oYXNoIjoiIiwibmFtZSI6IiIsImRldGFpbCI6IiIsInJpbmdfbWV0aG9kIjoiIiwic3RhdHVzIjoiIiwicGVybWlzc2lvbiI6MCwidGFnX2lkcyI6bnVsbCwiYWRkcmVzc2VzIjpudWxsfSwiZXhwaXJlIjoiMjAyMy0xMS0xOSAwOToyOToxMS43NjMzMzExMTgifQ.wfDWIwHRqZvu3JD0Cq-MbFzunJ41SFK3Qc21IQlma8c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1Login(ctx, gomock.Any(), tt.username, tt.password).Return(tt.responseAgent, nil)
			mockUtil.EXPECT().TimeGetCurTimeAdd(TokenExpiration).Return(tt.responseCurTime)

			res, err := h.AuthLogin(ctx, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectedRes {
				t.Errorf("Wrong match. expected: %v, got: %v", res, tt.expectedRes)
			}
		})
	}
}

func Test_AuthJWTGenerate(t *testing.T) {

	tests := []struct {
		name string

		data map[string]interface{}

		responseCurTime string

		expectRes map[string]interface{}
	}{
		{
			name: "normal",

			data: map[string]interface{}{
				"key1": "val1",
				"key2": "val2",
			},

			responseCurTime: "2023-11-19 09:29:11.763331118",
			expectRes: map[string]interface{}{
				"key1":   "val1",
				"key2":   "val2",
				"expire": "2023-11-19 09:29:11.763331118",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTimeAdd(TokenExpiration).Return(tt.responseCurTime)
			token, err := h.AuthJWTGenerate(tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			res, err := h.AuthJWTParse(ctx, token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AuthAccesskeyParse(t *testing.T) {

	tests := []struct {
		name string

		accesskey string

		responseAccesskeys []csaccesskey.Accesskey
		responseCurTime    string

		expectRes map[string]interface{}
	}{
		{
			name: "normal",

			accesskey: "test_accesskey",

			responseAccesskeys: []csaccesskey.Accesskey{
				{
					ID:         uuid.FromStringOrNil("42ea8504-af3f-11ef-8714-d3091c69e57f"),
					CustomerID: uuid.FromStringOrNil("a68a0422-af3f-11ef-a15e-5f3bf088a0e3"),
					Name:       "test key name",
					Detail:     "test key detail",
					TMExpire: nil,
					TMDelete: nil,
				},
			},
			responseCurTime: "2023-11-19 09:29:11.763331118",
			expectRes: map[string]interface{}{
				"agent": &amagent.Agent{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("42ea8504-af3f-11ef-8714-d3091c69e57f"),
						CustomerID: uuid.FromStringOrNil("a68a0422-af3f-11ef-a15e-5f3bf088a0e3"),
					},
					Name:   "test key name",
					Detail: "test key detail",

					Permission: amagent.PermissionCustomerAdmin,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockUtil.EXPECT().HashSHA256Hex(tt.accesskey).Return("hashed_" + tt.accesskey)
			mockReq.EXPECT().CustomerV1AccesskeyList(ctx, "", gomock.Any(), gomock.Any()).Return(tt.responseAccesskeys, nil)

			res, err := h.AuthAccesskeyParse(ctx, tt.accesskey)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AuthAccesskeyParse_error(t *testing.T) {

	tests := []struct {
		name string

		accesskey string

		responseAccesskeys []csaccesskey.Accesskey
		responseCurTime    string
	}{
		{
			name: "accesskey has expired",

			accesskey: "test_accesskey",

			responseAccesskeys: []csaccesskey.Accesskey{
				{
					ID:         uuid.FromStringOrNil("42ea8504-af3f-11ef-8714-d3091c69e57f"),
					CustomerID: uuid.FromStringOrNil("a68a0422-af3f-11ef-a15e-5f3bf088a0e3"),
					Name:       "test key name",
					Detail:     "test key detail",
					TMExpire:   timePtr("2023-11-19 09:29:11.763331117"),
					TMDelete: nil,
				},
			},
			responseCurTime: "2023-11-19 09:29:11.763331118",
		},
		{
			name: "token has deleted",

			accesskey: "test_accesskey",

			responseAccesskeys: []csaccesskey.Accesskey{
				{
					ID:         uuid.FromStringOrNil("42ea8504-af3f-11ef-8714-d3091c69e57f"),
					CustomerID: uuid.FromStringOrNil("a68a0422-af3f-11ef-a15e-5f3bf088a0e3"),
					Name:       "test key name",
					Detail:     "test key detail",
					TMExpire: nil,
					TMDelete:   timePtr("2023-11-19 09:29:11.763331117"),
				},
			},
			responseCurTime: "2023-11-19 09:29:11.763331118",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			mockUtil.EXPECT().HashSHA256Hex(tt.accesskey).Return("hashed_" + tt.accesskey)
			mockReq.EXPECT().CustomerV1AccesskeyList(ctx, "", gomock.Any(), gomock.Any()).Return(tt.responseAccesskeys, nil)

			_, err := h.AuthAccesskeyParse(ctx, tt.accesskey)
			if err == nil {
				t.Errorf("Wrong match. expected: error, got: ok")
			}
		})
	}
}
