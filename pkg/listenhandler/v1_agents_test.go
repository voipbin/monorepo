package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/agenthandler"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
)

func TestProcessV1AgentsGetNormal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		userID    uint64
		pageSize  uint64
		pageToken string

		agents    []*agent.Agent
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?user_id=1&page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			1,
			10,
			"2021-11-23 17:55:39.712000",

			[]*agent.Agent{
				{
					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
					UserID:       1,
					Username:     "test1",
					PasswordHash: "password",
					Name:         "test agent1",
					Detail:       "test agent1 detail",
					RingMethod:   "ringall",
					Status:       agent.StatusOffline,
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["27d3bc3e-4d88-11ec-a61d-af78fdede455"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentGets(gomock.Any(), tt.userID, tt.pageSize, tt.pageToken).Return(tt.agents, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1AgentsGetTagIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		userID uint64
		tagIDs []uuid.UUID

		agents    []*agent.Agent
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?user_id=1&tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"),
			},

			[]*agent.Agent{
				{
					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
					UserID:       1,
					Username:     "test1",
					PasswordHash: "password",
					Name:         "test agent1",
					Detail:       "test agent1 detail",
					RingMethod:   "ringall",
					Status:       agent.StatusOffline,
					Permission:   1,
					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
		{
			"have 2 tag ids",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?user_id=1&tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a,2e5705ea-4d90-11ec-9352-6326ee2dce20",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"),
				uuid.FromStringOrNil("2e5705ea-4d90-11ec-9352-6326ee2dce20"),
			},

			[]*agent.Agent{
				{
					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
					UserID:       1,
					Username:     "test1",
					PasswordHash: "password",
					Name:         "test agent1",
					Detail:       "test agent1 detail",
					RingMethod:   "ringall",
					Status:       agent.StatusOffline,
					Permission:   1,
					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
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
				{
					ID:           uuid.FromStringOrNil("473248a4-4d90-11ec-976a-172883175eb4"),
					UserID:       1,
					Username:     "test2",
					PasswordHash: "password",
					Name:         "test agent2",
					Detail:       "test agent2 detail",
					RingMethod:   "ringall",
					Status:       agent.StatusOffline,
					Permission:   1,
					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("2e5705ea-4d90-11ec-9352-6326ee2dce20")},
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"},{"id":"473248a4-4d90-11ec-976a-172883175eb4","user_id":1,"username":"test2","password_hash":"password","name":"test agent2","detail":"test agent2 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["2e5705ea-4d90-11ec-9352-6326ee2dce20"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentGetsByTagIDs(gomock.Any(), tt.userID, tt.tagIDs).Return(tt.agents, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1AgentsGetTagIDsAndStatus(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		userID uint64
		tagIDs []uuid.UUID
		status agent.Status

		agents    []*agent.Agent
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?user_id=1&tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a&status=available",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
					UserID:       1,
					Username:     "test1",
					PasswordHash: "password",
					Name:         "test agent1",
					Detail:       "test agent1 detail",
					RingMethod:   "ringall",
					Status:       agent.StatusAvailable,
					Permission:   1,
					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"available","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
		{
			"have 2 tag ids",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?user_id=1&tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a,2e5705ea-4d90-11ec-9352-6326ee2dce20&status=available",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"),
				uuid.FromStringOrNil("2e5705ea-4d90-11ec-9352-6326ee2dce20"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
					UserID:       1,
					Username:     "test1",
					PasswordHash: "password",
					Name:         "test agent1",
					Detail:       "test agent1 detail",
					RingMethod:   "ringall",
					Status:       agent.StatusAvailable,
					Permission:   1,
					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
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
				{
					ID:           uuid.FromStringOrNil("473248a4-4d90-11ec-976a-172883175eb4"),
					UserID:       1,
					Username:     "test2",
					PasswordHash: "password",
					Name:         "test agent2",
					Detail:       "test agent2 detail",
					RingMethod:   "ringall",
					Status:       agent.StatusAvailable,
					Permission:   1,
					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("2e5705ea-4d90-11ec-9352-6326ee2dce20")},
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"available","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"},{"id":"473248a4-4d90-11ec-976a-172883175eb4","user_id":1,"username":"test2","password_hash":"password","name":"test agent2","detail":"test agent2 detail","ring_method":"ringall","status":"available","permission":1,"tag_ids":["2e5705ea-4d90-11ec-9352-6326ee2dce20"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentGetsByTagIDsAndStatus(gomock.Any(), tt.userID, tt.tagIDs, tt.status).Return(tt.agents, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1AgentsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		userID     uint64
		username   string
		password   string
		agentName  string
		detail     string
		ringMethod string
		permission uint64
		tagIDs     []uuid.UUID
		addresses  []cmaddress.Address

		agent     *agent.Agent
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id":1,"username": "test1", "password":"password", "name": "test agent1", "detail": "test agent1 detail", "ring_method": "ringall", "permission": 1, "tag_ids": ["27d3bc3e-4d88-11ec-a61d-af78fdede455"], "addresses":[{"type": "tel", "target":"+821021656521"}]}`),
			},

			1,
			"test1",
			"password",
			"test agent1",
			"test agent1 detail",
			"ringall",
			1,
			[]uuid.UUID{uuid.FromStringOrNil("27d3bc3e-4d88-11ec-a61d-af78fdede455")},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			&agent.Agent{
				ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				UserID:       1,
				Username:     "test1",
				PasswordHash: "password",
				Name:         "test agent1",
				Detail:       "test agent1 detail",
				RingMethod:   "ringall",
				Status:       agent.StatusOffline,
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["27d3bc3e-4d88-11ec-a61d-af78fdede455"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
		{
			"have 2 tags",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id":1,"username": "test1", "password":"password", "name": "test agent1", "detail": "test agent1 detail", "ring_method": "ringall", "permission": 1, "tag_ids": ["159623f0-4d8c-11ec-85da-432863b96d60", "15ec14e0-4d8c-11ec-82e5-cbde7c2e6f84"], "addresses":[{"type": "tel", "target":"+821021656521"}]}`),
			},

			1,
			"test1",
			"password",
			"test agent1",
			"test agent1 detail",
			"ringall",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("159623f0-4d8c-11ec-85da-432863b96d60"),
				uuid.FromStringOrNil("15ec14e0-4d8c-11ec-82e5-cbde7c2e6f84"),
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			&agent.Agent{
				ID:           uuid.FromStringOrNil("28a63cc8-4d8c-11ec-959e-6bedf5864e94"),
				UserID:       1,
				Username:     "test1",
				PasswordHash: "password",
				Name:         "test agent1",
				Detail:       "test agent1 detail",
				RingMethod:   "ringall",
				Status:       agent.StatusOffline,
				Permission:   1,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("159623f0-4d8c-11ec-85da-432863b96d60"),
					uuid.FromStringOrNil("15ec14e0-4d8c-11ec-82e5-cbde7c2e6f84"),
				},
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"28a63cc8-4d8c-11ec-959e-6bedf5864e94","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["159623f0-4d8c-11ec-85da-432863b96d60","15ec14e0-4d8c-11ec-82e5-cbde7c2e6f84"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
		{
			"have 2 tags and addresses",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id":1,"username": "test1", "password":"password", "name": "test agent1", "detail": "test agent1 detail", "ring_method": "ringall", "permission": 1, "tag_ids": ["e7b166ec-4d8c-11ec-8c61-0b9e85603e10", "e82a311c-4d8c-11ec-9411-3382b1284325"], "addresses":[{"type": "tel", "target":"+821021656521"},{"type": "tel", "target":"+821021656522"}]}`),
			},

			1,
			"test1",
			"password",
			"test agent1",
			"test agent1 detail",
			"ringall",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("e7b166ec-4d8c-11ec-8c61-0b9e85603e10"),
				uuid.FromStringOrNil("e82a311c-4d8c-11ec-9411-3382b1284325"),
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+821021656522",
				},
			},

			&agent.Agent{
				ID:           uuid.FromStringOrNil("e85d8d78-4d8c-11ec-8a91-1f780097ef8d"),
				UserID:       1,
				Username:     "test1",
				PasswordHash: "password",
				Name:         "test agent1",
				Detail:       "test agent1 detail",
				RingMethod:   "ringall",
				Status:       agent.StatusOffline,
				Permission:   1,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("e7b166ec-4d8c-11ec-8c61-0b9e85603e10"),
					uuid.FromStringOrNil("e82a311c-4d8c-11ec-9411-3382b1284325"),
				},
				Addresses: []cmaddress.Address{
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656521",
					},
					{
						Type:   cmaddress.TypeTel,
						Target: "+821021656522",
					},
				},
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e85d8d78-4d8c-11ec-8a91-1f780097ef8d","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["e7b166ec-4d8c-11ec-8c61-0b9e85603e10","e82a311c-4d8c-11ec-9411-3382b1284325"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""},{"type":"tel","target":"+821021656522","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentCreate(gomock.Any(), tt.userID, tt.username, tt.password, tt.agentName, tt.detail, agent.RingMethod(tt.ringMethod), agent.Permission(tt.permission), tt.tagIDs, tt.addresses).Return(tt.agent, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1AgentsUsernameLoginPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		userID   uint64
		username string
		password string

		agent     *agent.Agent
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/test1/login",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id":1,"password":"password"}`),
			},

			1,
			"test1",
			"password",

			&agent.Agent{
				ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				UserID:       1,
				Username:     "test1",
				PasswordHash: "password",
				Name:         "test agent1",
				Detail:       "test agent1 detail",
				RingMethod:   "ringall",
				Status:       agent.StatusAvailable,
				Permission:   1,
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","user_id":1,"username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"available","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentLogin(gomock.Any(), tt.userID, tt.username, tt.password).Return(tt.agent, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1AgentsIDAddressesPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id        uuid.UUID
		addresses []cmaddress.Address

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a/addresses",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"addresses":[{"type":"tel","target":"+821021656521"}]}`),
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentUpdateAddresses(gomock.Any(), tt.id, tt.addresses).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1AgentsIDTagIDsPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id     uuid.UUID
		tagIDs []uuid.UUID

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a/tag_ids",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"tag_ids":["f196163c-4da3-11ec-bd6c-27b1ed6735b3"]}`),
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
			[]uuid.UUID{
				uuid.FromStringOrNil("f196163c-4da3-11ec-bd6c-27b1ed6735b3"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentUpdateTagIDs(gomock.Any(), tt.id, tt.tagIDs).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1AgentsIDDialPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id           uuid.UUID
		source       *cmaddress.Address
		confbridgeID uuid.UUID

		agent     *agent.Agent
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a/dial",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"source":{"type":"tel","target":"+821021656521"},"confbridge_id":"4a089d94-4da9-11ec-bac9-07c16f06b600"}`),
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821021656521",
			},
			uuid.FromStringOrNil("4a089d94-4da9-11ec-bac9-07c16f06b600"),

			&agent.Agent{
				ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				UserID:       1,
				Username:     "test1",
				PasswordHash: "password",
				Name:         "test agent1",
				Detail:       "test agent1 detail",
				RingMethod:   "ringall",
				Status:       agent.StatusAvailable,
				Permission:   1,
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f196163c-4da3-11ec-bd6c-27b1ed6735b3")},
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
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentDial(gomock.Any(), tt.id, tt.source, tt.confbridgeID).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestProcessV1AgentsIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		rabbitSock:   mockSock,
		db:           mockDB,
		reqHandler:   mockReq,
		agentHandler: mockAgent,
	}

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id uuid.UUID

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAgent.EXPECT().AgentDelete(gomock.Any(), tt.id).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
