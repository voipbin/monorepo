package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestAMV1AgentCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		username   string
		password   string
		agentName  string
		deail      string
		ringMethod amagent.RingMethod
		permission amagent.Permission
		tagIDs     []uuid.UUID
		addresses  []cmaddress.Address

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *amagent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			"test1",
			"password1",
			"test agent1",
			"test agent1 detail",
			amagent.RingMethodRingAll,
			amagent.PermissionNone,
			[]uuid.UUID{
				uuid.FromStringOrNil("ce0c4b4a-4e76-11ec-b6fe-9b57b172471a"),
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"7fdb8e66-7fe7-11ec-ac90-878b581c2615","username":"test1","password":"password1","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","permission":0,"tag_ids":["ce0c4b4a-4e76-11ec-b6fe-9b57b172471a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}]}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"7fdb8e66-7fe7-11ec-ac90-878b581c2615","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["27d3bc3e-4d88-11ec-a61d-af78fdede455"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
			&amagent.Agent{

				ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				CustomerID:   uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
				Username:     "test1",
				PasswordHash: "password",
				Name:         "test agent1",
				Detail:       "test agent1 detail",
				RingMethod:   "ringall",
				Status:       amagent.StatusOffline,
				Permission:   1,
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("27d3bc3e-4d88-11ec-a61d-af78fdede455")},
				Addresses: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656521",
					},
				},
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentCreate(ctx, requestTimeoutDefault, tt.customerID, tt.username, tt.password, tt.agentName, tt.deail, tt.ringMethod, tt.permission, tt.tagIDs, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		agentID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *amagent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("7ab80df4-4c72-11ec-b095-17146a0e7e4c"),

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/7ab80df4-4c72-11ec-b095-17146a0e7e4c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7ab80df4-4c72-11ec-b095-17146a0e7e4c"}`),
			},
			&amagent.Agent{
				ID: uuid.FromStringOrNil("7ab80df4-4c72-11ec-b095-17146a0e7e4c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentGet(ctx, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []amagent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=7fdb8e66-7fe7-11ec-ac90-878b581c2615",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d3ce27ac-4c72-11ec-b790-6b79445cbb01"}]`),
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("d3ce27ac-4c72-11ec-b790-6b79445cbb01"),
				},
			},
		},
		{
			"2 agents",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=7fdb8e66-7fe7-11ec-ac90-878b581c2615",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"11cfd8e8-4c73-11ec-8f06-b73cd86fc9ae"},{"id":"12237ce6-4c73-11ec-8a2a-57b7a8d6a6f4"}]`),
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("11cfd8e8-4c73-11ec-8f06-b73cd86fc9ae"),
				},
				{
					ID: uuid.FromStringOrNil("12237ce6-4c73-11ec-8a2a-57b7a8d6a6f4"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentGets(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentGetsByTagIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		tagIDs     []uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []amagent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			[]uuid.UUID{
				uuid.FromStringOrNil("ef626c46-4e78-11ec-bb14-6fbde14856d4"),
			},

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?customer_id=7fdb8e66-7fe7-11ec-ac90-878b581c2615&tag_ids=ef626c46-4e78-11ec-bb14-6fbde14856d4",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ef626c46-4e78-11ec-bb14-6fbde14856d4"}]`),
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("ef626c46-4e78-11ec-bb14-6fbde14856d4"),
				},
			},
		},
		{
			"2 agents",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			[]uuid.UUID{
				uuid.FromStringOrNil("36a057ee-4e79-11ec-a0c6-5fc332a14527"),
				uuid.FromStringOrNil("36c77248-4e79-11ec-8aa9-93ecdefec6c9"),
			},

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?customer_id=7fdb8e66-7fe7-11ec-ac90-878b581c2615&tag_ids=36a057ee-4e79-11ec-a0c6-5fc332a14527,36c77248-4e79-11ec-8aa9-93ecdefec6c9",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"36a057ee-4e79-11ec-a0c6-5fc332a14527"},{"id":"36c77248-4e79-11ec-8aa9-93ecdefec6c9"}]`),
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("36a057ee-4e79-11ec-a0c6-5fc332a14527"),
				},
				{
					ID: uuid.FromStringOrNil("36c77248-4e79-11ec-8aa9-93ecdefec6c9"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentGetsByTagIDs(ctx, tt.customerID, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentGetsByTagIDsAndStatus(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		tagIDs     []uuid.UUID
		status     amagent.Status

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []amagent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			[]uuid.UUID{
				uuid.FromStringOrNil("a23822ac-4e79-11ec-935d-335a1fd132e8"),
			},
			amagent.StatusAvailable,

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?customer_id=7fdb8e66-7fe7-11ec-ac90-878b581c2615&tag_ids=a23822ac-4e79-11ec-935d-335a1fd132e8&status=available",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"a23822ac-4e79-11ec-935d-335a1fd132e8"}]`),
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("a23822ac-4e79-11ec-935d-335a1fd132e8"),
				},
			},
		},
		{
			"2 agents",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			[]uuid.UUID{
				uuid.FromStringOrNil("bcde4bea-4e79-11ec-bbc2-4b92f6f04b6a"),
				uuid.FromStringOrNil("bd0786ea-4e79-11ec-8ecc-3bc59c72be3b"),
			},
			amagent.StatusAvailable,

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?customer_id=7fdb8e66-7fe7-11ec-ac90-878b581c2615&tag_ids=bcde4bea-4e79-11ec-bbc2-4b92f6f04b6a,bd0786ea-4e79-11ec-8ecc-3bc59c72be3b&status=available",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bcde4bea-4e79-11ec-bbc2-4b92f6f04b6a"},{"id":"bd0786ea-4e79-11ec-8ecc-3bc59c72be3b"}]`),
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("bcde4bea-4e79-11ec-bbc2-4b92f6f04b6a"),
				},
				{
					ID: uuid.FromStringOrNil("bd0786ea-4e79-11ec-8ecc-3bc59c72be3b"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentGetsByTagIDsAndStatus(ctx, tt.customerID, tt.tagIDs, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMAgentDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("f4b44b28-4e79-11ec-be3c-73450ec23a51"),

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/f4b44b28-4e79-11ec-be3c-73450ec23a51",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f4b44b28-4e79-11ec-be3c-73450ec23a51"}`),
			},
			&amagent.Agent{
				ID: uuid.FromStringOrNil("f4b44b28-4e79-11ec-be3c-73450ec23a51"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentLogin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		username   string
		password   string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *amagent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			"test1",
			"password1",

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/test1/login",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"7fdb8e66-7fe7-11ec-ac90-878b581c2615","password":"password1"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7ab80df4-4c72-11ec-b095-17146a0e7e4c"}`),
			},

			&amagent.Agent{
				ID: uuid.FromStringOrNil("7ab80df4-4c72-11ec-b095-17146a0e7e4c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentLogin(ctx, requestTimeoutDefault, tt.customerID, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentUpdateAddresses(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id        uuid.UUID
		addresses []cmaddress.Address

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/1e60cb12-4e7b-11ec-9d7b-532466c1faf1/addresses",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}]}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1e60cb12-4e7b-11ec-9d7b-532466c1faf1"}`),
			},

			&amagent.Agent{
				ID: uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentUpdateAddresses(ctx, tt.id, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentUpdatePassword(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id       uuid.UUID
		password string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			"password1",

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/1e60cb12-4e7b-11ec-9d7b-532466c1faf1/password",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"password":"password1"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1e60cb12-4e7b-11ec-9d7b-532466c1faf1"}`),
			},
			&amagent.Agent{
				ID: uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentUpdatePassword(ctx, requestTimeoutDefault, tt.id, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id         uuid.UUID
		agentName  string
		detail     string
		ringMethod amagent.RingMethod

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			"update name",
			"update detail",
			amagent.RingMethodRingAll,

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/1e60cb12-4e7b-11ec-9d7b-532466c1faf1",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail","ring_method":"ringall"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1e60cb12-4e7b-11ec-9d7b-532466c1faf1"}`),
			},
			&amagent.Agent{
				ID: uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentUpdate(ctx, tt.id, tt.agentName, tt.detail, string(tt.ringMethod))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func TestAMV1AgentUpdateTagIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id     uuid.UUID
		tagIDs []uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			[]uuid.UUID{
				uuid.FromStringOrNil("000c4a82-4e7c-11ec-a7e0-fff54f4ae71d"),
			},

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/1e60cb12-4e7b-11ec-9d7b-532466c1faf1/tag_ids",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"tag_ids":["000c4a82-4e7c-11ec-a7e0-fff54f4ae71d"]}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1e60cb12-4e7b-11ec-9d7b-532466c1faf1"}`),
			},
			&amagent.Agent{
				ID: uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentUpdateTagIDs(ctx, tt.id, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentUpdateStatus(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id     uuid.UUID
		status amagent.Status

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			amagent.StatusAvailable,

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/1e60cb12-4e7b-11ec-9d7b-532466c1faf1/status",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"status":"available"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1e60cb12-4e7b-11ec-9d7b-532466c1faf1"}`),
			},
			&amagent.Agent{
				ID: uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AMV1AgentUpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAMV1AgentDial(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id           uuid.UUID
		source       *cmaddress.Address
		confbridgeID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821021656521",
			},
			uuid.FromStringOrNil("719c27a8-4e7c-11ec-bbc9-3b81de0e9e0a"),

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/1e60cb12-4e7b-11ec-9d7b-532466c1faf1/dial",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"source":{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""},"confbridge_id":"719c27a8-4e7c-11ec-bbc9-3b81de0e9e0a"}`),
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

			if err := reqHandler.AMV1AgentDial(ctx, tt.id, tt.source, tt.confbridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
