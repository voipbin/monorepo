package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/participant"
)

func Test_ParticipantCreate(t *testing.T) {
	tests := []struct {
		name string
		data *participant.Participant
	}{
		{
			name: "normal participant",
			data: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea311"),
					CustomerID: uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9db11"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9db12"),
				},
				ChatID: uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9db13"),
			},
		},
		{
			name: "system participant",
			data: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("686e8e64-e428-11ec-baf2-7b14625ea312"),
					CustomerID: uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9db14"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeNone,
					OwnerID:   uuid.Nil,
				},
				ChatID: uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9db15"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:          dbTest,
				redis:       nil,
			utilHandler: commonutil.NewUtilHandler(),
			}
			ctx := context.Background()

			// Test create
			if err := h.ParticipantCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Test retrieval
			res, err := h.ParticipantGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Clear timestamps for comparison
			tt.data.TMJoined = nil
			res.TMJoined = nil

			if !reflect.DeepEqual(tt.data, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.data, res)
			}
		})
	}
}

func Test_ParticipantCreate_UPSERT(t *testing.T) {
	tests := []struct {
		name              string
		firstParticipant  *participant.Participant
		secondParticipant *participant.Participant
		expectUpsert      bool
	}{
		{
			name: "re-join updates timestamp",
			firstParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("786e8e64-e428-11ec-baf2-7b14625ea313"),
					CustomerID: uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9db16"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9db17"),
				},
				ChatID: uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9db18"),
			},
			secondParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("886e8e64-e428-11ec-baf2-7b14625ea314"), // Different ID
					CustomerID: uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9db16"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9db17"), // Same owner
				},
				ChatID: uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9db18"), // Same chat
			},
			expectUpsert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:          dbTest,
				redis:       nil,
			utilHandler: commonutil.NewUtilHandler(),
			}
			ctx := context.Background()

			// Create first participant
			if err := h.ParticipantCreate(ctx, tt.firstParticipant); err != nil {
				t.Errorf("Failed to create first participant: %v", err)
			}

			// Get first participant's timestamp
			firstRes, err := h.ParticipantGet(ctx, tt.firstParticipant.ID)
			if err != nil {
				t.Errorf("Failed to get first participant: %v", err)
			}
			var firstTimestamp *time.Time
			if firstRes.TMJoined != nil {
				t := *firstRes.TMJoined
				firstTimestamp = &t
			}

			// Create second participant (should UPSERT if same chat + owner)
			if err := h.ParticipantCreate(ctx, tt.secondParticipant); err != nil {
				t.Errorf("Failed to create second participant: %v", err)
			}

			if tt.expectUpsert {
				// List participants for this chat + owner combination
				filters := map[participant.Field]any{
					participant.FieldChatID:    tt.firstParticipant.ChatID,
					participant.FieldOwnerType: tt.firstParticipant.OwnerType,
					participant.FieldOwnerID:   tt.firstParticipant.OwnerID,
				}
				participants, err := h.ParticipantList(ctx, filters)
				if err != nil {
					t.Errorf("Failed to list participants: %v", err)
				}

				// Should only have 1 participant (UPSERT behavior)
				if len(participants) != 1 {
					t.Errorf("Expected 1 participant (UPSERT), got %d", len(participants))
				}

				// Timestamp should be updated (different from first)
				if firstTimestamp != nil && participants[0].TMJoined != nil && participants[0].TMJoined.Equal(*firstTimestamp) {
					t.Errorf("Expected timestamp to be updated, but it's the same: %v", firstTimestamp)
				}
			}
		})
	}
}

func Test_ParticipantGet(t *testing.T) {
	tests := []struct {
		name              string
		createParticipant *participant.Participant
		getID             uuid.UUID
		expectErr         bool
	}{
		{
			name: "existing participant",
			createParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("986e8e64-e428-11ec-baf2-7b14625ea315"),
					CustomerID: uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9db19"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9db20"),
				},
				ChatID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9db21"),
			},
			getID:     uuid.FromStringOrNil("986e8e64-e428-11ec-baf2-7b14625ea315"),
			expectErr: false,
		},
		{
			name: "non-existent participant",
			createParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a86e8e64-e428-11ec-baf2-7b14625ea316"),
					CustomerID: uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9db22"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9db23"),
				},
				ChatID: uuid.FromStringOrNil("c922f8c2-e428-11ec-b1a3-4bc67cb9db24"),
			},
			getID:     uuid.FromStringOrNil("999e8e64-e428-11ec-baf2-7b14625ea999"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:          dbTest,
				redis:       nil,
			utilHandler: commonutil.NewUtilHandler(),
			}
			ctx := context.Background()

			// Create participant first
			if err := h.ParticipantCreate(ctx, tt.createParticipant); err != nil {
				t.Errorf("Failed to create participant: %v", err)
			}

			// Test get
			res, err := h.ParticipantGet(ctx, tt.getID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if res == nil {
					t.Errorf("Expected participant but got nil")
				}
			}
		})
	}
}

func Test_ParticipantList(t *testing.T) {
	tests := []struct {
		name               string
		createParticipants []*participant.Participant
		filters            map[participant.Field]any
		expectLen          int
	}{
		{
			name: "list all participants for chat",
			createParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b86e8e64-e428-11ec-baf2-7b14625ea317"),
						CustomerID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9db25"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("c922f8c2-e428-11ec-b1a3-4bc67cb9db26"),
					},
					ChatID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9db27"),
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c86e8e64-e428-11ec-baf2-7b14625ea318"),
						CustomerID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9db25"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("e922f8c2-e428-11ec-b1a3-4bc67cb9db28"),
					},
					ChatID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9db27"),
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d86e8e64-e428-11ec-baf2-7b14625ea319"),
						CustomerID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9db25"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("f922f8c2-e428-11ec-b1a3-4bc67cb9db29"),
					},
					ChatID: uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9db30"), // Different chat
				},
			},
			filters: map[participant.Field]any{
				participant.FieldChatID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9db27"),
			},
			expectLen: 2,
		},
		{
			name: "filter by owner type",
			createParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e86e8e64-e428-11ec-baf2-7b14625ea320"),
						CustomerID: uuid.FromStringOrNil("e922f8c2-e428-11ec-b1a3-4bc67cb9db31"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("f922f8c2-e428-11ec-b1a3-4bc67cb9db32"),
					},
					ChatID: uuid.FromStringOrNil("2922f8c2-e428-11ec-b1a3-4bc67cb9db33"),
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f86e8e64-e428-11ec-baf2-7b14625ea321"),
						CustomerID: uuid.FromStringOrNil("e922f8c2-e428-11ec-b1a3-4bc67cb9db31"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeNone,
						OwnerID:   uuid.Nil,
					},
					ChatID: uuid.FromStringOrNil("2922f8c2-e428-11ec-b1a3-4bc67cb9db33"),
				},
			},
			filters: map[participant.Field]any{
				participant.FieldChatID:    uuid.FromStringOrNil("2922f8c2-e428-11ec-b1a3-4bc67cb9db33"),
				participant.FieldOwnerType: commonidentity.OwnerTypeAgent,
			},
			expectLen: 1,
		},
		{
			name: "filter by specific owner",
			createParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("186e8e64-e428-11ec-baf2-7b14625ea322"),
						CustomerID: uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9db34"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("2922f8c2-e428-11ec-b1a3-4bc67cb9db35"),
					},
					ChatID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9db36"),
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("286e8e64-e428-11ec-baf2-7b14625ea323"),
						CustomerID: uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9db34"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("4922f8c2-e428-11ec-b1a3-4bc67cb9db37"),
					},
					ChatID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9db36"),
				},
			},
			filters: map[participant.Field]any{
				participant.FieldChatID:  uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9db36"),
				participant.FieldOwnerID: uuid.FromStringOrNil("2922f8c2-e428-11ec-b1a3-4bc67cb9db35"),
			},
			expectLen: 1,
		},
		{
			name: "list by customer",
			createParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("386e8e64-e428-11ec-baf2-7b14625ea324"),
						CustomerID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9db38"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("4922f8c2-e428-11ec-b1a3-4bc67cb9db39"),
					},
					ChatID: uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9db40"),
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("486e8e64-e428-11ec-baf2-7b14625ea325"),
						CustomerID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9db38"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9db41"),
					},
					ChatID: uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9db42"),
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea326"),
						CustomerID: uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9db43"), // Different customer
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9db44"),
					},
					ChatID: uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9db45"),
				},
			},
			filters: map[participant.Field]any{
				participant.FieldCustomerID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9db38"),
			},
			expectLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:          dbTest,
				redis:       nil,
			utilHandler: commonutil.NewUtilHandler(),
			}
			ctx := context.Background()

			// Create participants
			for _, p := range tt.createParticipants {
				if err := h.ParticipantCreate(ctx, p); err != nil {
					t.Errorf("Failed to create participant: %v", err)
				}
			}

			// Test list
			res, err := h.ParticipantList(ctx, tt.filters)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(res) != tt.expectLen {
				t.Errorf("Wrong result count. expect: %d, got: %d", tt.expectLen, len(res))
			}
		})
	}
}

func Test_ParticipantDelete(t *testing.T) {
	tests := []struct {
		name              string
		createParticipant *participant.Participant
		deleteID          uuid.UUID
	}{
		{
			name: "hard delete participant",
			createParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("686e8e64-e428-11ec-baf2-7b14625ea327"),
					CustomerID: uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9db46"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9db47"),
				},
				ChatID: uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9db48"),
			},
			deleteID: uuid.FromStringOrNil("686e8e64-e428-11ec-baf2-7b14625ea327"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:          dbTest,
				redis:       nil,
			utilHandler: commonutil.NewUtilHandler(),
			}
			ctx := context.Background()

			// Create participant
			if err := h.ParticipantCreate(ctx, tt.createParticipant); err != nil {
				t.Errorf("Failed to create participant: %v", err)
			}

			// Delete participant (hard delete)
			if err := h.ParticipantDelete(ctx, tt.deleteID); err != nil {
				t.Errorf("Failed to delete participant: %v", err)
			}

			// Verify hard delete - should return error
			_, err := h.ParticipantGet(ctx, tt.deleteID)
			if err == nil {
				t.Errorf("Expected error for deleted participant, got none")
			}

			// Verify participant is not in list
			filterRes, err := h.ParticipantList(ctx, map[participant.Field]any{
				participant.FieldID: tt.deleteID,
			})
			if err != nil {
				t.Errorf("Failed to list participants: %v", err)
			}
			if len(filterRes) != 0 {
				t.Errorf("Expected deleted participant to be gone, but got %d results", len(filterRes))
			}
		})
	}
}
