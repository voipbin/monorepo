package dbhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/talk"
)

func Test_TalkCreate(t *testing.T) {
	tests := []struct {
		name string
		data *talk.Talk
	}{
		{
			name: "normal talk",
			data: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea112"),
					CustomerID: uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9daf4"),
				},
				Type: talk.TypeNormal,
			},
		},
		{
			name: "group talk",
			data: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("686e8e64-e428-11ec-baf2-7b14625ea113"),
					CustomerID: uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9daf5"),
				},
				Type: talk.TypeGroup,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:    dbTest,
				redis: nil,
			}
			ctx := context.Background()

			// Test create
			if err := h.TalkCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Test retrieval
			res, err := h.TalkGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Clear timestamps for comparison
			tt.data.TMCreate = ""
			tt.data.TMUpdate = ""
			tt.data.TMDelete = ""
			res.TMCreate = ""
			res.TMUpdate = ""
			res.TMDelete = ""

			if !reflect.DeepEqual(tt.data, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.data, res)
			}
		})
	}
}

func Test_TalkGet(t *testing.T) {
	tests := []struct {
		name      string
		createTalk *talk.Talk
		getID     uuid.UUID
		expectErr bool
	}{
		{
			name: "existing talk",
			createTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("786e8e64-e428-11ec-baf2-7b14625ea114"),
					CustomerID: uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9daf6"),
				},
				Type: talk.TypeNormal,
			},
			getID:     uuid.FromStringOrNil("786e8e64-e428-11ec-baf2-7b14625ea114"),
			expectErr: false,
		},
		{
			name: "non-existent talk",
			createTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("886e8e64-e428-11ec-baf2-7b14625ea115"),
					CustomerID: uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9daf7"),
				},
				Type: talk.TypeNormal,
			},
			getID:     uuid.FromStringOrNil("999e8e64-e428-11ec-baf2-7b14625ea999"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:    dbTest,
				redis: nil,
			}
			ctx := context.Background()

			// Create talk first
			if err := h.TalkCreate(ctx, tt.createTalk); err != nil {
				t.Errorf("Failed to create talk: %v", err)
			}

			// Test get
			res, err := h.TalkGet(ctx, tt.getID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if res == nil {
					t.Errorf("Expected talk but got nil")
				}
			}
		})
	}
}

func Test_TalkList(t *testing.T) {
	tests := []struct {
		name       string
		createTalks []*talk.Talk
		filters    map[talk.Field]any
		token      string
		size       uint64
		expectLen  int
	}{
		{
			name: "list all talks for customer",
			createTalks: []*talk.Talk{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a86e8e64-e428-11ec-baf2-7b14625ea116"),
						CustomerID: uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9daf8"),
					},
					Type: talk.TypeNormal,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b86e8e64-e428-11ec-baf2-7b14625ea117"),
						CustomerID: uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9daf8"),
					},
					Type: talk.TypeGroup,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c86e8e64-e428-11ec-baf2-7b14625ea118"),
						CustomerID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9daf9"),
					},
					Type: talk.TypeNormal,
				},
			},
			filters: map[talk.Field]any{
				talk.FieldCustomerID: uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9daf8"),
			},
			token:     "",
			size:      100,
			expectLen: 2,
		},
		{
			name: "filter by type",
			createTalks: []*talk.Talk{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d86e8e64-e428-11ec-baf2-7b14625ea119"),
						CustomerID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9daf0"),
					},
					Type: talk.TypeNormal,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e86e8e64-e428-11ec-baf2-7b14625ea120"),
						CustomerID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9daf0"),
					},
					Type: talk.TypeGroup,
				},
			},
			filters: map[talk.Field]any{
				talk.FieldCustomerID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9daf0"),
				talk.FieldType:       talk.TypeGroup,
			},
			token:     "",
			size:      100,
			expectLen: 1,
		},
		{
			name: "pagination limit",
			createTalks: []*talk.Talk{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f86e8e64-e428-11ec-baf2-7b14625ea121"),
						CustomerID: uuid.FromStringOrNil("f922f8c2-e428-11ec-b1a3-4bc67cb9daf1"),
					},
					Type: talk.TypeNormal,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("186e8e64-e428-11ec-baf2-7b14625ea122"),
						CustomerID: uuid.FromStringOrNil("f922f8c2-e428-11ec-b1a3-4bc67cb9daf1"),
					},
					Type: talk.TypeNormal,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("286e8e64-e428-11ec-baf2-7b14625ea123"),
						CustomerID: uuid.FromStringOrNil("f922f8c2-e428-11ec-b1a3-4bc67cb9daf1"),
					},
					Type: talk.TypeNormal,
				},
			},
			filters: map[talk.Field]any{
				talk.FieldCustomerID: uuid.FromStringOrNil("f922f8c2-e428-11ec-b1a3-4bc67cb9daf1"),
			},
			token:     "",
			size:      2,
			expectLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:    dbTest,
				redis: nil,
			}
			ctx := context.Background()

			// Create talks
			for _, talk := range tt.createTalks {
				if err := h.TalkCreate(ctx, talk); err != nil {
					t.Errorf("Failed to create talk: %v", err)
				}
			}

			// Test list
			res, err := h.TalkList(ctx, tt.filters, tt.token, tt.size)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(res) != tt.expectLen {
				t.Errorf("Wrong result count. expect: %d, got: %d", tt.expectLen, len(res))
			}
		})
	}
}

func Test_TalkUpdate(t *testing.T) {
	tests := []struct {
		name       string
		createTalk *talk.Talk
		updateID   uuid.UUID
		fields     map[talk.Field]any
		expectType talk.Type
	}{
		{
			name: "update type to group",
			createTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("386e8e64-e428-11ec-baf2-7b14625ea124"),
					CustomerID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9daf2"),
				},
				Type: talk.TypeNormal,
			},
			updateID: uuid.FromStringOrNil("386e8e64-e428-11ec-baf2-7b14625ea124"),
			fields: map[talk.Field]any{
				talk.FieldType: talk.TypeGroup,
			},
			expectType: talk.TypeGroup,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:    dbTest,
				redis: nil,
			}
			ctx := context.Background()

			// Create talk
			if err := h.TalkCreate(ctx, tt.createTalk); err != nil {
				t.Errorf("Failed to create talk: %v", err)
			}

			// Update talk
			if err := h.TalkUpdate(ctx, tt.updateID, tt.fields); err != nil {
				t.Errorf("Failed to update talk: %v", err)
			}

			// Verify update
			res, err := h.TalkGet(ctx, tt.updateID)
			if err != nil {
				t.Errorf("Failed to get talk: %v", err)
			}

			if res.Type != tt.expectType {
				t.Errorf("Wrong type. expect: %v, got: %v", tt.expectType, res.Type)
			}
		})
	}
}

func Test_TalkDelete(t *testing.T) {
	tests := []struct {
		name       string
		createTalk *talk.Talk
		deleteID   uuid.UUID
	}{
		{
			name: "soft delete talk",
			createTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("486e8e64-e428-11ec-baf2-7b14625ea125"),
					CustomerID: uuid.FromStringOrNil("4922f8c2-e428-11ec-b1a3-4bc67cb9daf3"),
				},
				Type: talk.TypeNormal,
			},
			deleteID: uuid.FromStringOrNil("486e8e64-e428-11ec-baf2-7b14625ea125"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{
				db:    dbTest,
				redis: nil,
			}
			ctx := context.Background()

			// Create talk
			if err := h.TalkCreate(ctx, tt.createTalk); err != nil {
				t.Errorf("Failed to create talk: %v", err)
			}

			// Delete talk (soft delete)
			if err := h.TalkDelete(ctx, tt.deleteID); err != nil {
				t.Errorf("Failed to delete talk: %v", err)
			}

			// Verify soft delete - should return error when getting deleted talk
			res, err := h.TalkGet(ctx, tt.deleteID)
			if err == nil {
				// If no error, verify tm_delete is not default
				if res.TMDelete == commondatabasehandler.DefaultTimeStamp {
					t.Errorf("Expected soft delete timestamp, got default timestamp")
				}
			}

			// Verify talk is excluded from list with deleted=false filter
			filterRes, err := h.TalkList(ctx, map[talk.Field]any{
				talk.FieldID:      tt.deleteID,
				talk.FieldDeleted: false,
			}, "", 100)
			if err != nil {
				t.Errorf("Failed to list talks: %v", err)
			}
			if len(filterRes) != 0 {
				t.Errorf("Expected deleted talk to be filtered out, but got %d results", len(filterRes))
			}
		})
	}
}
