package teamhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	memberB := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiB := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	customerID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc")
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")

	members := []team.Member{
		{
			ID:   memberA,
			Name: "Greeter",
			AIID: aiA,
			Transitions: []team.Transition{
				{FunctionName: "transfer_to_b", Description: "Go to B", NextMemberID: memberB},
			},
		},
		{
			ID:   memberB,
			Name: "Specialist",
			AIID: aiB,
		},
	}

	tests := []struct {
		name string

		customerID    uuid.UUID
		teamName      string
		detail        string
		startMemberID uuid.UUID
		members       []team.Member

		responseUUID uuid.UUID
		responseTeam *team.Team

		expectTeam *team.Team
	}{
		{
			name: "normal",

			customerID:    customerID,
			teamName:      "test team",
			detail:        "test detail",
			startMemberID: memberA,
			members:       members,

			responseUUID: teamID,
			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID:         teamID,
					CustomerID: customerID,
				},
			},

			expectTeam: &team.Team{
				Identity: identity.Identity{
					ID:         teamID,
					CustomerID: customerID,
				},
				Name:          "test team",
				Detail:        "test detail",
				StartMemberID: memberA,
				Members:       members,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &teamHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
			mockDB.EXPECT().AIGet(ctx, aiB).Return(&ai.AI{}, nil)
			mockDB.EXPECT().TeamCreate(ctx, tt.expectTeam).Return(nil)
			mockDB.EXPECT().TeamGet(ctx, tt.responseUUID).Return(tt.responseTeam, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTeam.CustomerID, team.EventTypeCreated, tt.responseTeam)

			res, err := h.Create(ctx, tt.customerID, tt.teamName, tt.detail, tt.startMemberID, tt.members)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseTeam) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTeam, res)
			}
		})
	}
}

func Test_Create_validation_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()

	// Empty members — should fail validation before any DB call
	_, err := h.Create(ctx, uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"), "test", "detail", uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), []team.Member{})
	if err == nil {
		t.Error("Expected error for empty members, got nil")
	}
}

func Test_Create_ai_not_found(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()

	members := []team.Member{
		{ID: memberA, Name: "A", AIID: aiA},
	}

	mockDB.EXPECT().AIGet(ctx, aiA).Return(nil, fmt.Errorf("not found"))

	_, err := h.Create(ctx, uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"), "test", "detail", memberA, members)
	if err == nil {
		t.Error("Expected error for non-existent AI, got nil")
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string
		id   uuid.UUID

		responseTeam *team.Team
	}{
		{
			"normal",

			uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd"),

			&team.Team{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &teamHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().TeamGet(ctx, tt.id).Return(tt.responseTeam, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseTeam) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTeam, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name    string
		size    uint64
		token   string
		filters map[team.Field]any

		responseTeams []*team.Team
	}{
		{
			name:  "normal",
			size:  10,
			token: "2023-01-03T21:35:02.809Z",
			filters: map[team.Field]any{
				team.FieldDeleted:    false,
				team.FieldCustomerID: uuid.FromStringOrNil("132be434-f839-11ed-ae95-efa657af10fb"),
			},

			responseTeams: []*team.Team{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("31b00c64-f839-11ed-8f59-ab874a16ee9c"),
					},
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &teamHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().TeamList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseTeams, nil)

			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseTeams) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTeams, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string
		id   uuid.UUID

		responseTeam *team.Team
	}{
		{
			"normal",

			uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995"),

			&team.Team{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &teamHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().TeamDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().TeamGet(ctx, tt.id).Return(tt.responseTeam, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTeam.CustomerID, team.EventTypeDeleted, tt.responseTeam)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseTeam) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTeam, res)
			}
		})
	}
}

func Test_Update(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	memberB := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiB := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")

	members := []team.Member{
		{ID: memberA, Name: "A", AIID: aiA},
		{ID: memberB, Name: "B", AIID: aiB},
	}

	tests := []struct {
		name          string
		id            uuid.UUID
		teamName      string
		detail        string
		startMemberID uuid.UUID
		members       []team.Member

		responseTeam *team.Team
	}{
		{
			name:          "normal",
			id:            teamID,
			teamName:      "updated team",
			detail:        "updated detail",
			startMemberID: memberA,
			members:       members,

			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID: teamID,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &teamHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
			mockDB.EXPECT().AIGet(ctx, aiB).Return(&ai.AI{}, nil)
			mockDB.EXPECT().TeamUpdate(ctx, tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().TeamGet(ctx, tt.id).Return(tt.responseTeam, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTeam.CustomerID, team.EventTypeUpdated, tt.responseTeam)

			res, err := h.Update(ctx, tt.id, tt.teamName, tt.detail, tt.startMemberID, tt.members)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseTeam) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseTeam, res)
			}
		})
	}
}

func Test_Get_db_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd")

	mockDB.EXPECT().TeamGet(ctx, id).Return(nil, fmt.Errorf("db error"))

	_, err := h.Get(ctx, id)
	if err == nil {
		t.Error("Expected error for db failure, got nil")
	}
}

func Test_List_db_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()
	filters := map[team.Field]any{
		team.FieldDeleted: false,
	}

	mockDB.EXPECT().TeamList(ctx, uint64(10), "2023-01-03T21:35:02.809Z", filters).Return(nil, fmt.Errorf("db error"))

	_, err := h.List(ctx, 10, "2023-01-03T21:35:02.809Z", filters)
	if err == nil {
		t.Error("Expected error for db failure, got nil")
	}
}

func Test_Delete_db_delete_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995")

	mockDB.EXPECT().TeamDelete(ctx, id).Return(fmt.Errorf("db error"))

	_, err := h.Delete(ctx, id)
	if err == nil {
		t.Error("Expected error for db delete failure, got nil")
	}
}

func Test_Delete_db_get_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995")

	mockDB.EXPECT().TeamDelete(ctx, id).Return(nil)
	mockDB.EXPECT().TeamGet(ctx, id).Return(nil, fmt.Errorf("db error"))

	_, err := h.Delete(ctx, id)
	if err == nil {
		t.Error("Expected error for db get failure after delete, got nil")
	}
}

func Test_Update_validation_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")

	// Empty members — should fail validation before any DB call
	_, err := h.Update(ctx, teamID, "test", "detail", uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), []team.Member{})
	if err == nil {
		t.Error("Expected error for empty members, got nil")
	}
}

func Test_Update_ai_not_found(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")

	members := []team.Member{
		{ID: memberA, Name: "A", AIID: aiA},
	}

	mockDB.EXPECT().AIGet(ctx, aiA).Return(nil, fmt.Errorf("not found"))

	_, err := h.Update(ctx, teamID, "test", "detail", memberA, members)
	if err == nil {
		t.Error("Expected error for non-existent AI, got nil")
	}
}

func Test_Update_db_error(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")

	members := []team.Member{
		{ID: memberA, Name: "A", AIID: aiA},
	}

	mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
	mockDB.EXPECT().TeamUpdate(ctx, teamID, gomock.Any()).Return(fmt.Errorf("db error"))

	_, err := h.Update(ctx, teamID, "test", "detail", memberA, members)
	if err == nil {
		t.Error("Expected error for db update failure, got nil")
	}
}

func Test_Create_db_create_error(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	customerID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc")
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()

	members := []team.Member{
		{ID: memberA, Name: "A", AIID: aiA},
	}

	mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
	mockUtil.EXPECT().UUIDCreate().Return(teamID)
	mockDB.EXPECT().TeamCreate(ctx, gomock.Any()).Return(fmt.Errorf("db error"))

	_, err := h.Create(ctx, customerID, "test", "detail", memberA, members)
	if err == nil {
		t.Error("Expected error for db create failure, got nil")
	}
}

func Test_Create_db_get_error(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	customerID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc")
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &teamHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	ctx := context.Background()

	members := []team.Member{
		{ID: memberA, Name: "A", AIID: aiA},
	}

	mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
	mockUtil.EXPECT().UUIDCreate().Return(teamID)
	mockDB.EXPECT().TeamCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().TeamGet(ctx, teamID).Return(nil, fmt.Errorf("db error"))

	_, err := h.Create(ctx, customerID, "test", "detail", memberA, members)
	if err == nil {
		t.Error("Expected error for db get failure after create, got nil")
	}
}
