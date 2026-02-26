package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/teamhandler"
)

func Test_processV1TeamsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseTeams []*team.Team

		expectPageSize  uint64
		expectPageToken string
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/teams?page_size=10&page_token=2020-05-03T21:35:02.809Z&filter_customer_id=24676972-7f49-11ec-bc89-b7d33e9d3ea8&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			responseTeams: []*team.Team{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("0b61dcbe-a770-11ed-bab4-2fc1dac66672"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("0bbe1dee-a770-11ed-b455-cbb60d5dd90b"),
					},
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-05-03T21:35:02.809Z",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0b61dcbe-a770-11ed-bab4-2fc1dac66672","customer_id":"00000000-0000-0000-0000-000000000000","start_member_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null},{"id":"0bbe1dee-a770-11ed-b455-cbb60d5dd90b","customer_id":"00000000-0000-0000-0000-000000000000","start_member_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				teamHandler: mockTeam,
			}

			mockTeam.EXPECT().List(gomock.Any(), tt.expectPageSize, tt.expectPageToken, gomock.Any()).Return(tt.responseTeams, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TeamsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseTeam *team.Team

		expectCustomerID    uuid.UUID
		expectName          string
		expectDetail        string
		expectStartMemberID uuid.UUID
		expectMembers       []team.Member
		expectRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/teams",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847","name":"test team","detail":"test detail","start_member_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","members":[{"id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","name":"Greeter","ai_id":"11111111-1111-1111-1111-111111111111","transitions":[{"function_name":"transfer_to_b","description":"Go to B","next_member_id":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}]},{"id":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","name":"Specialist","ai_id":"22222222-2222-2222-2222-222222222222"}]}`),
			},

			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("59230ca2-a770-11ed-b5dd-2783587ed477"),
				},
			},

			expectCustomerID:    uuid.FromStringOrNil("58e7502c-a770-11ed-9b86-7fabe2dba847"),
			expectName:          "test team",
			expectDetail:        "test detail",
			expectStartMemberID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			expectMembers: []team.Member{
				{
					ID:   uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					Name: "Greeter",
					AIID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					Transitions: []team.Transition{
						{FunctionName: "transfer_to_b", Description: "Go to B", NextMemberID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")},
					},
				},
				{
					ID:   uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
					Name: "Specialist",
					AIID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"59230ca2-a770-11ed-b5dd-2783587ed477","customer_id":"00000000-0000-0000-0000-000000000000","start_member_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				teamHandler: mockTeam,
			}

			mockTeam.EXPECT().Create(
				gomock.Any(),
				tt.expectCustomerID,
				tt.expectName,
				tt.expectDetail,
				tt.expectStartMemberID,
				tt.expectMembers,
			).Return(tt.responseTeam, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TeamsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseTeam *team.Team

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/teams/de740384-a770-11ed-afab-5f9c8a447889",
				Method: sock.RequestMethodGet,
			},

			&team.Team{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("de740384-a770-11ed-afab-5f9c8a447889"),
				},
			},

			uuid.FromStringOrNil("de740384-a770-11ed-afab-5f9c8a447889"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"de740384-a770-11ed-afab-5f9c8a447889","customer_id":"00000000-0000-0000-0000-000000000000","start_member_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				teamHandler: mockTeam,
			}

			mockTeam.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseTeam, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TeamsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseTeam *team.Team

		expectID            uuid.UUID
		expectName          string
		expectDetail        string
		expectStartMemberID uuid.UUID
		expectMembers       []team.Member
		expectRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/teams/fa4d3b6a-f82f-11ed-9176-d32f5705e10c",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"updated team","detail":"updated detail","start_member_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","members":[{"id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","name":"A","ai_id":"11111111-1111-1111-1111-111111111111"}]}`),
			},

			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
				},
			},

			expectID:            uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
			expectName:          "updated team",
			expectDetail:        "updated detail",
			expectStartMemberID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			expectMembers: []team.Member{
				{
					ID:   uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					Name: "A",
					AIID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa4d3b6a-f82f-11ed-9176-d32f5705e10c","customer_id":"00000000-0000-0000-0000-000000000000","start_member_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				teamHandler: mockTeam,
			}

			mockTeam.EXPECT().Update(
				gomock.Any(),
				tt.expectID,
				tt.expectName,
				tt.expectDetail,
				tt.expectStartMemberID,
				tt.expectMembers,
			).Return(tt.responseTeam, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TeamsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseTeam *team.Team

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/teams/de99e522-a770-11ed-a0ab-5b39ee2db203",
				Method: sock.RequestMethodDelete,
			},

			&team.Team{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203"),
				},
			},

			uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"de99e522-a770-11ed-a0ab-5b39ee2db203","customer_id":"00000000-0000-0000-0000-000000000000","start_member_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				teamHandler: mockTeam,
			}

			mockTeam.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseTeam, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
