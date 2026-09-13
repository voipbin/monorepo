package teamhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

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

	directID := uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111")

	tests := []struct {
		name string

		customerID    uuid.UUID
		teamName      string
		detail        string
		startMemberID uuid.UUID
		members       []team.Member

		responseUUID   uuid.UUID
		responseDirect *dmdirect.Direct
		responseTeam   *team.Team

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
			responseDirect: &dmdirect.Direct{
				Identity: identity.Identity{
					ID:         directID,
					CustomerID: customerID,
				},
				ResourceType: dmdirect.ResourceTypeAITeam,
				ResourceID:   teamID,
				Hash:         "abc123def456",
			},
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
				DirectID:      directID,
				DirectHash:    "abc123def456",
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
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockReq.EXPECT().DirectV1DirectCreate(ctx, tt.customerID, dmdirect.ResourceTypeAITeam, tt.responseUUID).Return(tt.responseDirect, nil)
			mockDB.EXPECT().TeamCreate(ctx, tt.expectTeam).Return(nil)
			mockDB.EXPECT().TeamGet(ctx, tt.responseUUID).Return(tt.responseTeam, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTeam.CustomerID, team.EventTypeCreated, tt.responseTeam)

			res, err := h.Create(ctx, tt.customerID, tt.teamName, tt.detail, tt.startMemberID, tt.members, nil)
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
	_, err := h.Create(ctx, uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"), "test", "detail", uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), []team.Member{}, nil)
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

	_, err := h.Create(ctx, uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"), "test", "detail", memberA, members, nil)
	if err == nil {
		t.Error("Expected error for non-existent AI, got nil")
	}
}

func Test_Create_insight_member_rejected(t *testing.T) {
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

	// VOIP-1234 §6 v4 item4: an Insight-typed AI must not be admittable as a team member.
	mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{Type: ai.TypeInsight}, nil)

	_, err := h.Create(ctx, uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"), "test", "detail", memberA, members, nil)
	if err == nil {
		t.Error("Expected error for Insight-typed AI member, got nil")
	}
}

func Test_Update_insight_member_rejected(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
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

	// VOIP-1234 §6 v4 item4: same rejection applies to Update, so an existing
	// team cannot be edited to admit an Insight-typed AI either.
	mockDB.EXPECT().TeamGet(ctx, teamID).Return(&team.Team{
		Identity: identity.Identity{ID: teamID},
	}, nil)
	mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{Type: ai.TypeInsight}, nil)

	name := "test"
	detail := "detail"
	_, err := h.Update(ctx, teamID, &name, &detail, &memberA, &members, nil)
	if err == nil {
		t.Error("Expected error for Insight-typed AI member, got nil")
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
				DirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				DirectHash: "test123hash0",
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
			mockReq.EXPECT().DirectV1DirectDelete(ctx, tt.responseTeam.DirectID).Return(nil, nil)
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

	strPtr := func(v string) *string { return &v }

	tests := []struct {
		name          string
		id            uuid.UUID
		teamName      *string
		detail        *string
		startMemberID *uuid.UUID
		members       *[]team.Member

		currentTeam *team.Team

		expectAIGets    bool
		expectNoUpdate  bool
		expectValidFail bool

		responseTeam *team.Team
	}{
		{
			name:          "normal, all fields set",
			id:            teamID,
			teamName:      strPtr("updated team"),
			detail:        strPtr("updated detail"),
			startMemberID: &memberA,
			members:       &members,

			expectAIGets: true,

			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID: teamID,
				},
			},
		},
		{
			name:     "name only, all other fields omitted",
			id:       teamID,
			teamName: strPtr("renamed"),

			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID: teamID,
				},
			},
		},
		{
			name: "all fields omitted, no-op",
			id:   teamID,

			expectNoUpdate: true,

			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID: teamID,
				},
			},
		},
		{
			name:          "start_member_id only, validated against existing members",
			id:            teamID,
			startMemberID: &memberB,

			currentTeam: &team.Team{
				Identity:      identity.Identity{ID: teamID},
				StartMemberID: memberA,
				Members:       members,
			},
			expectAIGets: true,

			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID: teamID,
				},
			},
		},
		{
			// design doc §4 item 9: start_member_id omitted, members provided,
			// and the *existing* start_member_id IS present in the new
			// members list -- success.
			name:    "members only, existing start_member_id present in new members",
			id:      teamID,
			members: &members,

			currentTeam: &team.Team{
				Identity:      identity.Identity{ID: teamID},
				StartMemberID: memberA,
				Members:       []team.Member{{ID: memberB, Name: "old-only-b", AIID: aiB}},
			},
			expectAIGets: true,

			responseTeam: &team.Team{
				Identity: identity.Identity{
					ID: teamID,
				},
			},
		},
		{
			// design doc §4 item 9: start_member_id omitted, members provided,
			// and the *existing* start_member_id is NOT present in the new
			// members list -- expect a validation error.
			name:            "members only, existing start_member_id absent from new members",
			id:              teamID,
			members:         &[]team.Member{{ID: memberB, Name: "B", AIID: aiB}},
			expectValidFail: true,

			currentTeam: &team.Team{
				Identity:      identity.Identity{ID: teamID},
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

			if tt.currentTeam != nil {
				mockDB.EXPECT().TeamGet(ctx, tt.id).Return(tt.currentTeam, nil)
			} else if tt.startMemberID != nil || tt.members != nil {
				// merged-validation Get, triggered whenever either coupled
				// field is being changed and no explicit currentTeam fixture
				// was supplied for this case.
				mockDB.EXPECT().TeamGet(ctx, tt.id).Return(&team.Team{
					Identity:      identity.Identity{ID: tt.id},
					StartMemberID: memberA,
					Members:       members,
				}, nil)
			}

			if tt.expectValidFail {
				// Validation must fail before any AIGet/TeamUpdate/TeamGet
				// (post-write) call; no additional mocks needed here.
				_, err := h.Update(ctx, tt.id, tt.teamName, tt.detail, tt.startMemberID, tt.members, nil)
				if err == nil {
					t.Error("Expected validation error, got nil")
				}
				return
			}

			if tt.expectAIGets {
				mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
				mockDB.EXPECT().AIGet(ctx, aiB).Return(&ai.AI{}, nil)
			}

			if tt.expectNoUpdate {
				mockDB.EXPECT().TeamUpdate(ctx, tt.id, gomock.Any()).Times(0)
				mockDB.EXPECT().TeamGet(ctx, tt.id).Return(tt.responseTeam, nil)
			} else {
				mockDB.EXPECT().TeamUpdate(ctx, tt.id, gomock.Any()).Return(nil)
				mockDB.EXPECT().TeamGet(ctx, tt.id).Return(tt.responseTeam, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTeam.CustomerID, team.EventTypeUpdated, tt.responseTeam)
			}

			res, err := h.Update(ctx, tt.id, tt.teamName, tt.detail, tt.startMemberID, tt.members, nil)
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
	directID := uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222")

	responseTeam := &team.Team{
		Identity: identity.Identity{
			ID: id,
		},
		DirectID:   directID,
		DirectHash: "test123hash0",
	}

	mockDB.EXPECT().TeamGet(ctx, id).Return(responseTeam, nil)
	mockReq.EXPECT().DirectV1DirectDelete(ctx, directID).Return(nil, nil)
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

	// First TeamGet (before delete) succeeds with no direct
	firstTeam := &team.Team{
		Identity: identity.Identity{
			ID: id,
		},
	}
	mockDB.EXPECT().TeamGet(ctx, id).Return(firstTeam, nil)
	mockDB.EXPECT().TeamDelete(ctx, id).Return(nil)
	// Second TeamGet (after delete) fails
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
	name := "test"
	detail := "detail"
	memberID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	emptyMembers := []team.Member{}
	mockDB.EXPECT().TeamGet(ctx, teamID).Return(&team.Team{
		Identity: identity.Identity{ID: teamID},
	}, nil)
	_, err := h.Update(ctx, teamID, &name, &detail, &memberID, &emptyMembers, nil)
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

	mockDB.EXPECT().TeamGet(ctx, teamID).Return(&team.Team{
		Identity: identity.Identity{ID: teamID},
	}, nil)
	mockDB.EXPECT().AIGet(ctx, aiA).Return(nil, fmt.Errorf("not found"))

	name := "test"
	detail := "detail"
	_, err := h.Update(ctx, teamID, &name, &detail, &memberA, &members, nil)
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

	mockDB.EXPECT().TeamGet(ctx, teamID).Return(&team.Team{
		Identity: identity.Identity{ID: teamID},
	}, nil)
	mockDB.EXPECT().AIGet(ctx, aiA).Return(&ai.AI{}, nil)
	mockDB.EXPECT().TeamUpdate(ctx, teamID, gomock.Any()).Return(fmt.Errorf("db error"))

	name := "test"
	detail := "detail"
	_, err := h.Update(ctx, teamID, &name, &detail, &memberA, &members, nil)
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
	mockReq.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAITeam, teamID).Return(&dmdirect.Direct{Hash: "a1b2c3d4e5f6"}, nil)
	mockDB.EXPECT().TeamCreate(ctx, gomock.Any()).Return(fmt.Errorf("db error"))
	mockReq.EXPECT().DirectV1DirectDelete(ctx, gomock.Any()).Return(nil, nil)

	_, err := h.Create(ctx, customerID, "test", "detail", memberA, members, nil)
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
	mockReq.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAITeam, teamID).Return(&dmdirect.Direct{Hash: "a1b2c3d4e5f6"}, nil)
	mockDB.EXPECT().TeamCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().TeamGet(ctx, teamID).Return(nil, fmt.Errorf("db error"))

	_, err := h.Create(ctx, customerID, "test", "detail", memberA, members, nil)
	if err == nil {
		t.Error("Expected error for db get failure after create, got nil")
	}
}

// Test_Update_NotFoundDuringMergedValidation pins the behavior that a
// dbhandler.ErrNotFound surfaced by the merged-validation Get (triggered
// whenever start_member_id or members is being changed) propagates as a
// typed *cerrors.VoipbinError with Status=StatusNotFound and
// Reason="TEAM_NOT_FOUND", exercising the exact
// errors.Wrap(err, "could not get current team for validation") line in
// teamHandler.Update -- not merely that *some* error is returned. This
// verifies the unwrap-transparency of github.com/pkg/errors.Wrap (its
// withMessage/withStack wrappers implement Unwrap(), so errors.As can walk
// through them to find the underlying *cerrors.VoipbinError set by
// teamHandler.Get).
func Test_Update_NotFoundDuringMergedValidation(t *testing.T) {
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

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

	// startMemberID is non-nil, so Update must trigger the merged-validation
	// Get; simulate the team having been deleted concurrently.
	mockDB.EXPECT().TeamGet(ctx, teamID).Return(nil, dbhandler.ErrNotFound)

	_, err := h.Update(ctx, teamID, nil, nil, &memberA, nil, nil)
	if err == nil {
		t.Fatal("Expected error for not-found during merged validation, got nil")
	}

	var voipbinErr *cerrors.VoipbinError
	if !stderrors.As(err, &voipbinErr) {
		t.Fatalf("Expected error to unwrap to *cerrors.VoipbinError, got: %v (%T)", err, err)
	}
	if voipbinErr.Status != cerrors.StatusNotFound {
		t.Errorf("Wrong status.\nexpect: %v\ngot: %v", cerrors.StatusNotFound, voipbinErr.Status)
	}
	if voipbinErr.Reason != "TEAM_NOT_FOUND" {
		t.Errorf("Wrong reason.\nexpect: %v\ngot: %v", "TEAM_NOT_FOUND", voipbinErr.Reason)
	}
}

// Test_Update_StartMemberIDOnly_EmptyStoredMembers pins the intentional
// "confusing 400" behavior documented in the Phase 6b design doc §3: a
// start_member_id-only PUT can still fail validation if the team's
// currently-stored members is empty, since validateTeam's rule 10 runs
// against the merged (existing members + new start_member_id) state, not
// just the request body.
func Test_Update_StartMemberIDOnly_EmptyStoredMembers(t *testing.T) {
	teamID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

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

	// Currently-stored members is empty.
	mockDB.EXPECT().TeamGet(ctx, teamID).Return(&team.Team{
		Identity: identity.Identity{ID: teamID},
		Members:  []team.Member{},
	}, nil)

	_, err := h.Update(ctx, teamID, nil, nil, &memberA, nil, nil)
	if err == nil {
		t.Error("Expected validation error for start_member_id-only update against empty stored members, got nil")
	}
}
