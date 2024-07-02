package listenhandler

import (
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_ProcessV1AgentsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		pageSize  uint64
		pageToken string

		responseFilters map[string]string
		responseAgents  []*agent.Agent
		expectRes       *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents?page_size=10&page_token=2021-11-23%2017:55:39.712000&filter_customer_id=5fd7f9b8-cb37-11ee-bd29-f30560a6ac86&filter_tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a,08789a66-b236-11ee-8a51-b31bbd98fe91&filter_deleted=false&filter_status=available",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			10,
			"2021-11-23 17:55:39.712000",

			map[string]string{
				"customer_id": "5fd7f9b8-cb37-11ee-bd29-f30560a6ac86",
				"deleted":     "false",
				"status":      string(agent.StatusAvailable),
				"tag_ids":     "f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a,08789a66-b236-11ee-8a51-b31bbd98fe91",
			},
			[]*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
						CustomerID: uuid.FromStringOrNil("5fd7f9b8-cb37-11ee-bd29-f30560a6ac86"),
					},
					Username:     "test1",
					PasswordHash: "password",
					Name:         "test agent1",
					Detail:       "test agent1 detail",
					RingMethod:   "ringall",
					Status:       agent.StatusOffline,
					Permission:   1,
					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("27d3bc3e-4d88-11ec-a61d-af78fdede455")},
					Addresses: []commonaddress.Address{
						{
							Type:   commonaddress.TypeTel,
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
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"5fd7f9b8-cb37-11ee-bd29-f30560a6ac86","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["27d3bc3e-4d88-11ec-a61d-af78fdede455"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
				utilHandler:  mockUtil,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockAgent.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseAgents, nil)

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

// func TestProcessV1AgentsGetTagIDs(t *testing.T) {

// 	tests := []struct {
// 		name    string
// 		request *rabbitmqhandler.Request

// 		customerID uuid.UUID
// 		tagIDs     []uuid.UUID

// 		agents    []*agent.Agent
// 		expectRes *rabbitmqhandler.Response
// 	}{
// 		{
// 			"normal",
// 			&rabbitmqhandler.Request{
// 				URI:      "/v1/agents?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a",
// 				Method:   rabbitmqhandler.RequestMethodGet,
// 				DataType: "application/json",
// 			},

// 			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 			[]uuid.UUID{
// 				uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"),
// 			},

// 			[]*agent.Agent{
// 				{
// 					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
// 					CustomerID:   uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 					Username:     "test1",
// 					PasswordHash: "password",
// 					Name:         "test agent1",
// 					Detail:       "test agent1 detail",
// 					RingMethod:   "ringall",
// 					Status:       agent.StatusOffline,
// 					Permission:   1,
// 					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
// 					Addresses: []commonaddress.Address{
// 						{
// 							Type:   commonaddress.TypeTel,
// 							Target: "+821021656521",
// 						},
// 					},
// 					TMCreate: "2021-11-23 17:55:39.712000",
// 					TMUpdate: "9999-01-01 00:00:00.000000",
// 					TMDelete: "9999-01-01 00:00:00.000000",
// 				},
// 			},
// 			&rabbitmqhandler.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
// 			},
// 		},
// 		{
// 			"have 2 tag ids",
// 			&rabbitmqhandler.Request{
// 				URI:      "/v1/agents?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a,2e5705ea-4d90-11ec-9352-6326ee2dce20",
// 				Method:   rabbitmqhandler.RequestMethodGet,
// 				DataType: "application/json",
// 			},

// 			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 			[]uuid.UUID{
// 				uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"),
// 				uuid.FromStringOrNil("2e5705ea-4d90-11ec-9352-6326ee2dce20"),
// 			},

// 			[]*agent.Agent{
// 				{
// 					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
// 					CustomerID:   uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 					Username:     "test1",
// 					PasswordHash: "password",
// 					Name:         "test agent1",
// 					Detail:       "test agent1 detail",
// 					RingMethod:   "ringall",
// 					Status:       agent.StatusOffline,
// 					Permission:   1,
// 					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
// 					Addresses: []commonaddress.Address{
// 						{
// 							Type:   commonaddress.TypeTel,
// 							Target: "+821021656521",
// 						},
// 					},
// 					TMCreate: "2021-11-23 17:55:39.712000",
// 					TMUpdate: "9999-01-01 00:00:00.000000",
// 					TMDelete: "9999-01-01 00:00:00.000000",
// 				},
// 				{
// 					ID:           uuid.FromStringOrNil("473248a4-4d90-11ec-976a-172883175eb4"),
// 					CustomerID:   uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 					Username:     "test2",
// 					PasswordHash: "password",
// 					Name:         "test agent2",
// 					Detail:       "test agent2 detail",
// 					RingMethod:   "ringall",
// 					Status:       agent.StatusOffline,
// 					Permission:   1,
// 					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("2e5705ea-4d90-11ec-9352-6326ee2dce20")},
// 					Addresses: []commonaddress.Address{
// 						{
// 							Type:   commonaddress.TypeTel,
// 							Target: "+821021656521",
// 						},
// 					},
// 					TMCreate: "2021-11-23 17:55:39.712000",
// 					TMUpdate: "9999-01-01 00:00:00.000000",
// 					TMDelete: "9999-01-01 00:00:00.000000",
// 				},
// 			},
// 			&rabbitmqhandler.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"},{"id":"473248a4-4d90-11ec-976a-172883175eb4","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username":"test2","password_hash":"password","name":"test agent2","detail":"test agent2 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["2e5705ea-4d90-11ec-9352-6326ee2dce20"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 			mockAgent := agenthandler.NewMockAgentHandler(mc)

// 			h := &listenHandler{
// 				rabbitSock:   mockSock,
// 				agentHandler: mockAgent,
// 			}

// 			mockAgent.EXPECT().GetsByTagIDs(gomock.Any(), tt.customerID, tt.tagIDs).Return(tt.agents, nil)

// 			res, err := h.processRequest(tt.request)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(res, tt.expectRes) != true {
// 				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
// 			}

// 		})
// 	}
// }

// func TestProcessV1AgentsGetTagIDsAndStatus(t *testing.T) {

// 	tests := []struct {
// 		name    string
// 		request *rabbitmqhandler.Request

// 		customerID uuid.UUID
// 		tagIDs     []uuid.UUID
// 		status     agent.Status

// 		agents    []*agent.Agent
// 		expectRes *rabbitmqhandler.Response
// 	}{
// 		{
// 			"normal",
// 			&rabbitmqhandler.Request{
// 				URI:      "/v1/agents?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a&status=available",
// 				Method:   rabbitmqhandler.RequestMethodGet,
// 				DataType: "application/json",
// 			},

// 			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 			[]uuid.UUID{
// 				uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"),
// 			},
// 			agent.StatusAvailable,

// 			[]*agent.Agent{
// 				{
// 					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
// 					CustomerID:   uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 					Username:     "test1",
// 					PasswordHash: "password",
// 					Name:         "test agent1",
// 					Detail:       "test agent1 detail",
// 					RingMethod:   "ringall",
// 					Status:       agent.StatusAvailable,
// 					Permission:   1,
// 					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
// 					Addresses: []commonaddress.Address{
// 						{
// 							Type:   commonaddress.TypeTel,
// 							Target: "+821021656521",
// 						},
// 					},
// 					TMCreate: "2021-11-23 17:55:39.712000",
// 					TMUpdate: "9999-01-01 00:00:00.000000",
// 					TMDelete: "9999-01-01 00:00:00.000000",
// 				},
// 			},
// 			&rabbitmqhandler.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"available","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
// 			},
// 		},
// 		{
// 			"have 2 tag ids",
// 			&rabbitmqhandler.Request{
// 				URI:      "/v1/agents?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a,2e5705ea-4d90-11ec-9352-6326ee2dce20&status=available",
// 				Method:   rabbitmqhandler.RequestMethodGet,
// 				DataType: "application/json",
// 			},

// 			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 			[]uuid.UUID{
// 				uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"),
// 				uuid.FromStringOrNil("2e5705ea-4d90-11ec-9352-6326ee2dce20"),
// 			},
// 			agent.StatusAvailable,

// 			[]*agent.Agent{
// 				{
// 					ID:           uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
// 					CustomerID:   uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 					Username:     "test1",
// 					PasswordHash: "password",
// 					Name:         "test agent1",
// 					Detail:       "test agent1 detail",
// 					RingMethod:   "ringall",
// 					Status:       agent.StatusAvailable,
// 					Permission:   1,
// 					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
// 					Addresses: []commonaddress.Address{
// 						{
// 							Type:   commonaddress.TypeTel,
// 							Target: "+821021656521",
// 						},
// 					},
// 					TMCreate: "2021-11-23 17:55:39.712000",
// 					TMUpdate: "9999-01-01 00:00:00.000000",
// 					TMDelete: "9999-01-01 00:00:00.000000",
// 				},
// 				{
// 					ID:           uuid.FromStringOrNil("473248a4-4d90-11ec-976a-172883175eb4"),
// 					CustomerID:   uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
// 					Username:     "test2",
// 					PasswordHash: "password",
// 					Name:         "test agent2",
// 					Detail:       "test agent2 detail",
// 					RingMethod:   "ringall",
// 					Status:       agent.StatusAvailable,
// 					Permission:   1,
// 					TagIDs:       []uuid.UUID{uuid.FromStringOrNil("2e5705ea-4d90-11ec-9352-6326ee2dce20")},
// 					Addresses: []commonaddress.Address{
// 						{
// 							Type:   commonaddress.TypeTel,
// 							Target: "+821021656521",
// 						},
// 					},
// 					TMCreate: "2021-11-23 17:55:39.712000",
// 					TMUpdate: "9999-01-01 00:00:00.000000",
// 					TMDelete: "9999-01-01 00:00:00.000000",
// 				},
// 			},
// 			&rabbitmqhandler.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"available","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"},{"id":"473248a4-4d90-11ec-976a-172883175eb4","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username":"test2","password_hash":"password","name":"test agent2","detail":"test agent2 detail","ring_method":"ringall","status":"available","permission":1,"tag_ids":["2e5705ea-4d90-11ec-9352-6326ee2dce20"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 			mockAgent := agenthandler.NewMockAgentHandler(mc)

// 			h := &listenHandler{
// 				rabbitSock:   mockSock,
// 				agentHandler: mockAgent,
// 			}

// 			mockAgent.EXPECT().GetsByTagIDsAndStatus(gomock.Any(), tt.customerID, tt.tagIDs, tt.status).Return(tt.agents, nil)

// 			res, err := h.processRequest(tt.request)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(res, tt.expectRes) != true {
// 				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
// 			}

// 		})
// 	}
// }

func TestProcessV1AgentsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		customerID uuid.UUID
		username   string
		password   string
		agentName  string
		detail     string
		ringMethod string
		permission uint64
		tagIDs     []uuid.UUID
		addresses  []commonaddress.Address

		agent     *agent.Agent
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username": "test1", "password":"password", "name": "test agent1", "detail": "test agent1 detail", "ring_method": "ringall", "permission": 1, "tag_ids": ["27d3bc3e-4d88-11ec-a61d-af78fdede455"], "addresses":[{"type": "tel", "target":"+821021656521"}]}`),
			},

			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
			"test1",
			"password",
			"test agent1",
			"test agent1 detail",
			"ringall",
			1,
			[]uuid.UUID{uuid.FromStringOrNil("27d3bc3e-4d88-11ec-a61d-af78fdede455")},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				Username:     "test1",
				PasswordHash: "password",
				Name:         "test agent1",
				Detail:       "test agent1 detail",
				RingMethod:   "ringall",
				Status:       agent.StatusOffline,
				Permission:   1,
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("27d3bc3e-4d88-11ec-a61d-af78fdede455")},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
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
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["27d3bc3e-4d88-11ec-a61d-af78fdede455"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
		{
			"have 2 tags",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username": "test1", "password":"password", "name": "test agent1", "detail": "test agent1 detail", "ring_method": "ringall", "permission": 1, "tag_ids": ["159623f0-4d8c-11ec-85da-432863b96d60", "15ec14e0-4d8c-11ec-82e5-cbde7c2e6f84"], "addresses":[{"type": "tel", "target":"+821021656521"}]}`),
			},

			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
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
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("28a63cc8-4d8c-11ec-959e-6bedf5864e94"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
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
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
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
				Data:       []byte(`{"id":"28a63cc8-4d8c-11ec-959e-6bedf5864e94","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["159623f0-4d8c-11ec-85da-432863b96d60","15ec14e0-4d8c-11ec-82e5-cbde7c2e6f84"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
		{
			"have 2 tags and addresses",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","username": "test1", "password":"password", "name": "test agent1", "detail": "test agent1 detail", "ring_method": "ringall", "permission": 1, "tag_ids": ["e7b166ec-4d8c-11ec-8c61-0b9e85603e10", "e82a311c-4d8c-11ec-9411-3382b1284325"], "addresses":[{"type": "tel", "target":"+821021656521"},{"type": "tel", "target":"+821021656522"}]}`),
			},

			uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
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
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821021656522",
				},
			},

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e85d8d78-4d8c-11ec-8a91-1f780097ef8d"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
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
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
					},
					{
						Type:   commonaddress.TypeTel,
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
				Data:       []byte(`{"id":"e85d8d78-4d8c-11ec-8a91-1f780097ef8d","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"offline","permission":1,"tag_ids":["e7b166ec-4d8c-11ec-8c61-0b9e85603e10","e82a311c-4d8c-11ec-9411-3382b1284325"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""},{"type":"tel","target":"+821021656522","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().Create(gomock.Any(), tt.customerID, tt.username, tt.password, tt.agentName, tt.detail, agent.RingMethod(tt.ringMethod), agent.Permission(tt.permission), tt.tagIDs, tt.addresses).Return(tt.agent, nil)

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

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

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
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","password":"password"}`),
			},

			"test1",
			"password",

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				Username:     "test1",
				PasswordHash: "password",
				Name:         "test agent1",
				Detail:       "test agent1 detail",
				RingMethod:   "ringall",
				Status:       agent.StatusAvailable,
				Permission:   1,
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
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
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"test1","password_hash":"password","name":"test agent1","detail":"test agent1 detail","ring_method":"ringall","status":"available","permission":1,"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"],"addresses":[{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""}],"tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().Login(gomock.Any(), tt.username, tt.password).Return(tt.agent, nil)

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

func TestProcessV1AgentsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id    uuid.UUID
		agent *agent.Agent

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
			&agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().Get(gomock.Any(), tt.id).Return(tt.agent, nil)

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

func TestProcessV1AgentsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id         uuid.UUID
		agentName  string
		detail     string
		ringMethod agent.RingMethod

		resonseAgent *agent.Agent
		expectRes    *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"name1","detail":"detail1","ring_method":"ringall"}`),
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
			"name1",
			"detail1",
			agent.RingMethodRingAll,

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().UpdateBasicInfo(gomock.Any(), tt.id, tt.agentName, tt.detail, tt.ringMethod).Return(tt.resonseAgent, nil)
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

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id        uuid.UUID
		addresses []commonaddress.Address

		responseAgent *agent.Agent
		expectRes     *rabbitmqhandler.Response
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
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().UpdateAddresses(gomock.Any(), tt.id, tt.addresses).Return(tt.responseAgent, nil)
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

func TestProcessV1AgentsIDStatusPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id     uuid.UUID
		status agent.Status

		resonseAgent *agent.Agent
		expectRes    *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a/status",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"status":"available"}`),
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
			agent.StatusAvailable,

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().UpdateStatus(gomock.Any(), tt.id, tt.status).Return(tt.resonseAgent, nil)

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

func TestProcessV1AgentsIDPasswordPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id       uuid.UUID
		password string

		responseAgent *agent.Agent
		expectRes     *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a/password",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"password":"password1"}`),
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
			"password1",

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().UpdatePassword(gomock.Any(), tt.id, tt.password).Return(tt.responseAgent, nil)
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

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id     uuid.UUID
		tagIDs []uuid.UUID

		responseAgent *agent.Agent
		expectRes     *rabbitmqhandler.Response
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

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().UpdateTagIDs(gomock.Any(), tt.id, tt.tagIDs).Return(tt.responseAgent, nil)
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

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id uuid.UUID

		responseAgent *agent.Agent
		expectRes     *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseAgent, nil)

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

func Test_processV1AgentsIDPermissionPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id         uuid.UUID
		permission agent.Permission

		responseAgent *agent.Agent
		expectRes     *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/agents/f1ba04b0-951e-11ee-a0a2-7b8600a1ee45/permission",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"permission":32}`),
			},

			uuid.FromStringOrNil("f1ba04b0-951e-11ee-a0a2-7b8600a1ee45"),
			agent.PermissionCustomerAdmin,

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1ba04b0-951e-11ee-a0a2-7b8600a1ee45"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f1ba04b0-951e-11ee-a0a2-7b8600a1ee45","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().UpdatePermission(gomock.Any(), tt.id, tt.permission).Return(tt.responseAgent, nil)
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

func Test_processV1AgentsGetByCustomerIDAddressPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseAgent *agent.Agent

		expectCustomerID uuid.UUID
		expectAddress    *commonaddress.Address

		expectRes *rabbitmqhandler.Response
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/agents/get_by_customer_id_address",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"e30eaa4a-2d91-11ef-88e9-bf9e5d130578","address":{"type":"extension","target":"f27b2198-2d91-11ef-9b01-93e01658c814"}}`),
			},

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f2a28abc-2d91-11ef-823f-b37d45f6968f"),
				},
			},

			expectCustomerID: uuid.FromStringOrNil("e30eaa4a-2d91-11ef-88e9-bf9e5d130578"),
			expectAddress: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "f27b2198-2d91-11ef-9b01-93e01658c814",
			},

			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f2a28abc-2d91-11ef-823f-b37d45f6968f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","username":"","password_hash":"","name":"","detail":"","ring_method":"","status":"","permission":0,"tag_ids":null,"addresses":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().GetByCustomerIDAndAddress(gomock.Any(), tt.expectCustomerID, tt.expectAddress).Return(tt.responseAgent, nil)

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
