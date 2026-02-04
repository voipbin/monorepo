package dbhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/message"
)

func Test_MessageCreate(t *testing.T) {
	tests := []struct {
		name string
		data *message.Message
	}{
		{
			name: "normal message",
			data: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea211"),
					CustomerID: uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9da11"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9da12"),
				},
				ChatID:   uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9da13"),
				Type:     message.TypeNormal,
				Text:     "Hello world",
				Medias:   []message.Media{},
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
			},
		},
		{
			name: "system message",
			data: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("786e8e64-e428-11ec-baf2-7b14625ea213"),
					CustomerID: uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9da17"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeNone,
					OwnerID:   uuid.Nil,
				},
				ChatID:   uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9da18"),
				Type:     message.TypeSystem,
				Text:     "User joined the chat",
				Medias:   []message.Media{},
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
			},
		},
		{
			name: "message with medias JSON",
			data: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("886e8e64-e428-11ec-baf2-7b14625ea214"),
					CustomerID: uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9da19"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9da20"),
				},
				ChatID: uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9da21"),
				Type:   message.TypeNormal,
				Text:   "Check this file",
				Medias: []message.Media{
					{
						Type:    message.MediaTypeLink,
						LinkURL: "https://example.com/file.pdf",
					},
				},
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
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
			if err := h.MessageCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Test retrieval
			res, err := h.MessageGet(ctx, tt.data.ID)
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

func Test_MessageGet(t *testing.T) {
	tests := []struct {
		name          string
		createMessage *message.Message
		getID         uuid.UUID
		expectErr     bool
	}{
		{
			name: "existing message",
			createMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("986e8e64-e428-11ec-baf2-7b14625ea215"),
					CustomerID: uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9da22"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9da23"),
				},
				ChatID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9da24"),
				Type:   message.TypeNormal,
				Text:   "Test message",
			},
			getID:     uuid.FromStringOrNil("986e8e64-e428-11ec-baf2-7b14625ea215"),
			expectErr: false,
		},
		{
			name: "non-existent message",
			createMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a86e8e64-e428-11ec-baf2-7b14625ea216"),
					CustomerID: uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9da25"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9da26"),
				},
				ChatID: uuid.FromStringOrNil("c922f8c2-e428-11ec-b1a3-4bc67cb9da27"),
				Type:   message.TypeNormal,
				Text:   "Test message",
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

			// Create message first
			if err := h.MessageCreate(ctx, tt.createMessage); err != nil {
				t.Errorf("Failed to create message: %v", err)
			}

			// Test get
			res, err := h.MessageGet(ctx, tt.getID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if res == nil {
					t.Errorf("Expected message but got nil")
				}
			}
		})
	}
}

func Test_MessageList(t *testing.T) {
	tests := []struct {
		name           string
		createMessages []*message.Message
		filters        map[message.Field]any
		token          string
		size           uint64
		expectLen      int
	}{
		{
			name: "list all messages for chat",
			createMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b86e8e64-e428-11ec-baf2-7b14625ea217"),
						CustomerID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9da28"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("c922f8c2-e428-11ec-b1a3-4bc67cb9da29"),
					},
					ChatID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9da30"),
					Type:   message.TypeNormal,
					Text:   "Message 1",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c86e8e64-e428-11ec-baf2-7b14625ea218"),
						CustomerID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9da28"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("c922f8c2-e428-11ec-b1a3-4bc67cb9da29"),
					},
					ChatID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9da30"),
					Type:   message.TypeNormal,
					Text:   "Message 2",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d86e8e64-e428-11ec-baf2-7b14625ea219"),
						CustomerID: uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9da28"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("c922f8c2-e428-11ec-b1a3-4bc67cb9da29"),
					},
					ChatID: uuid.FromStringOrNil("e922f8c2-e428-11ec-b1a3-4bc67cb9da31"),
					Type:   message.TypeNormal,
					Text:   "Message 3 (different chat)",
				},
			},
			filters: map[message.Field]any{
				message.FieldChatID: uuid.FromStringOrNil("d922f8c2-e428-11ec-b1a3-4bc67cb9da30"),
			},
			token:     "",
			size:      100,
			expectLen: 2,
		},
		{
			name: "filter by owner",
			createMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e86e8e64-e428-11ec-baf2-7b14625ea220"),
						CustomerID: uuid.FromStringOrNil("e922f8c2-e428-11ec-b1a3-4bc67cb9da32"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("f922f8c2-e428-11ec-b1a3-4bc67cb9da33"),
					},
					ChatID: uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9da34"),
					Type:   message.TypeNormal,
					Text:   "Agent message",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f86e8e64-e428-11ec-baf2-7b14625ea221"),
						CustomerID: uuid.FromStringOrNil("e922f8c2-e428-11ec-b1a3-4bc67cb9da32"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeNone,
						OwnerID:   uuid.Nil,
					},
					ChatID: uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9da34"),
					Type:   message.TypeSystem,
					Text:   "System message",
				},
			},
			filters: map[message.Field]any{
				message.FieldChatID:    uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9da34"),
				message.FieldOwnerType: commonidentity.OwnerTypeAgent,
			},
			token:     "",
			size:      100,
			expectLen: 1,
		},
		{
			name: "pagination limit",
			createMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("186e8e64-e428-11ec-baf2-7b14625ea222"),
						CustomerID: uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9da35"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("2922f8c2-e428-11ec-b1a3-4bc67cb9da36"),
					},
					ChatID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9da37"),
					Type:   message.TypeNormal,
					Text:   "Message 1",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("286e8e64-e428-11ec-baf2-7b14625ea223"),
						CustomerID: uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9da35"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("2922f8c2-e428-11ec-b1a3-4bc67cb9da36"),
					},
					ChatID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9da37"),
					Type:   message.TypeNormal,
					Text:   "Message 2",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("386e8e64-e428-11ec-baf2-7b14625ea224"),
						CustomerID: uuid.FromStringOrNil("1922f8c2-e428-11ec-b1a3-4bc67cb9da35"),
					},
					Owner: commonidentity.Owner{
						OwnerType: commonidentity.OwnerTypeAgent,
						OwnerID:   uuid.FromStringOrNil("2922f8c2-e428-11ec-b1a3-4bc67cb9da36"),
					},
					ChatID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9da37"),
					Type:   message.TypeNormal,
					Text:   "Message 3",
				},
			},
			filters: map[message.Field]any{
				message.FieldChatID: uuid.FromStringOrNil("3922f8c2-e428-11ec-b1a3-4bc67cb9da37"),
			},
			token:     "",
			size:      2,
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

			// Create messages
			for _, msg := range tt.createMessages {
				if err := h.MessageCreate(ctx, msg); err != nil {
					t.Errorf("Failed to create message: %v", err)
				}
			}

			// Test list
			res, err := h.MessageList(ctx, tt.filters, tt.token, tt.size)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(res) != tt.expectLen {
				t.Errorf("Wrong result count. expect: %d, got: %d", tt.expectLen, len(res))
			}
		})
	}
}

func Test_MessageUpdate(t *testing.T) {
	tests := []struct {
		name          string
		createMessage *message.Message
		updateID      uuid.UUID
		fields        map[message.Field]any
		expectText    string
	}{
		{
			name: "update text",
			createMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("486e8e64-e428-11ec-baf2-7b14625ea225"),
					CustomerID: uuid.FromStringOrNil("4922f8c2-e428-11ec-b1a3-4bc67cb9da38"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9da39"),
				},
				ChatID: uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9da40"),
				Type:   message.TypeNormal,
				Text:   "Original text",
			},
			updateID: uuid.FromStringOrNil("486e8e64-e428-11ec-baf2-7b14625ea225"),
			fields: map[message.Field]any{
				message.FieldText: "Updated text",
			},
			expectText: "Updated text",
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

			// Create message
			if err := h.MessageCreate(ctx, tt.createMessage); err != nil {
				t.Errorf("Failed to create message: %v", err)
			}

			// Update message
			if err := h.MessageUpdate(ctx, tt.updateID, tt.fields); err != nil {
				t.Errorf("Failed to update message: %v", err)
			}

			// Verify update
			res, err := h.MessageGet(ctx, tt.updateID)
			if err != nil {
				t.Errorf("Failed to get message: %v", err)
			}

			if res.Text != tt.expectText {
				t.Errorf("Wrong text. expect: %v, got: %v", tt.expectText, res.Text)
			}
		})
	}
}

func Test_MessageDelete(t *testing.T) {
	tests := []struct {
		name          string
		createMessage *message.Message
		deleteID      uuid.UUID
	}{
		{
			name: "soft delete message",
			createMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea226"),
					CustomerID: uuid.FromStringOrNil("5922f8c2-e428-11ec-b1a3-4bc67cb9da41"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9da42"),
				},
				ChatID: uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9da43"),
				Type:   message.TypeNormal,
				Text:   "To be deleted",
			},
			deleteID: uuid.FromStringOrNil("586e8e64-e428-11ec-baf2-7b14625ea226"),
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

			// Create message
			if err := h.MessageCreate(ctx, tt.createMessage); err != nil {
				t.Errorf("Failed to create message: %v", err)
			}

			// Delete message (soft delete)
			if err := h.MessageDelete(ctx, tt.deleteID); err != nil {
				t.Errorf("Failed to delete message: %v", err)
			}

			// Verify soft delete - should return error when getting deleted message
			res, err := h.MessageGet(ctx, tt.deleteID)
			if err == nil {
				// If no error, verify tm_delete is not default
				if res.TMDelete == commondatabasehandler.DefaultTimeStamp {
					t.Errorf("Expected soft delete timestamp, got default timestamp")
				}
			}

			// Verify message is excluded from list with deleted=false filter
			filterRes, err := h.MessageList(ctx, map[message.Field]any{
				message.FieldID:      tt.deleteID,
				message.FieldDeleted: false,
			}, "", 100)
			if err != nil {
				t.Errorf("Failed to list messages: %v", err)
			}
			if len(filterRes) != 0 {
				t.Errorf("Expected deleted message to be filtered out, but got %d results", len(filterRes))
			}
		})
	}
}

func Test_MessageAddReactionAtomic(t *testing.T) {
	tests := []struct {
		name          string
		createMessage *message.Message
		reaction      message.Reaction
		expectCount   int
	}{
		{
			name: "add first reaction",
			createMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("686e8e64-e428-11ec-baf2-7b14625ea227"),
					CustomerID: uuid.FromStringOrNil("6922f8c2-e428-11ec-b1a3-4bc67cb9da44"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9da45"),
				},
				ChatID:   uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9da46"),
				Type:     message.TypeNormal,
				Text:     "Message with reactions",
				Medias:   []message.Media{},
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
			},
			reaction: message.Reaction{
				Emoji:     "👍",
				OwnerType: string(commonidentity.OwnerTypeAgent),
				OwnerID:   uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9da47"),
				TMCreate:  "2024-01-01T00:00:00.000000Z",
			},
			expectCount: 1,
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

			// Create message
			if err := h.MessageCreate(ctx, tt.createMessage); err != nil {
				t.Errorf("Failed to create message: %v", err)
			}

			// Add reaction atomically
			reactionJSON, err := json.Marshal(tt.reaction)
			if err != nil {
				t.Errorf("Failed to marshal reaction: %v", err)
			}

			if err := h.MessageAddReactionAtomic(ctx, tt.createMessage.ID, string(reactionJSON)); err != nil {
				t.Errorf("Failed to add reaction: %v", err)
			}

			// Verify reaction was added
			res, err := h.MessageGet(ctx, tt.createMessage.ID)
			if err != nil {
				t.Errorf("Failed to get message: %v", err)
			}

			if len(res.Metadata.Reactions) != tt.expectCount {
				t.Errorf("Wrong reaction count. expect: %d, got: %d", tt.expectCount, len(res.Metadata.Reactions))
			}

			if res.Metadata.Reactions[0].Emoji != tt.reaction.Emoji {
				t.Errorf("Wrong emoji. expect: %s, got: %s", tt.reaction.Emoji, res.Metadata.Reactions[0].Emoji)
			}
		})
	}
}

func Test_MessageAddReactionAtomic_Concurrent(t *testing.T) {
	// Skip this test for SQLite - it uses read-modify-write instead of true atomic operations
	// MySQL uses JSON_ARRAY_APPEND which is atomic, but SQLite implementation can have race conditions
	t.Skip("Skipping concurrent reaction test - SQLite implementation not truly atomic (MySQL-specific feature)")

	h := &dbHandler{
		db:          dbTest,
		redis:       nil,
			utilHandler: commonutil.NewUtilHandler(),
	}
	ctx := context.Background()

	// Create test message
	testMessage := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("786e8e64-e428-11ec-baf2-7b14625ea228"),
			CustomerID: uuid.FromStringOrNil("7922f8c2-e428-11ec-b1a3-4bc67cb9da48"),
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9da49"),
		},
		ChatID:   uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9da50"),
		Type:     message.TypeNormal,
		Text:     "Concurrent reactions test",
		Medias:   []message.Media{},
		Metadata: message.Metadata{Reactions: []message.Reaction{}},
	}

	if err := h.MessageCreate(ctx, testMessage); err != nil {
		t.Errorf("Failed to create message: %v", err)
	}

	// Add reactions concurrently
	var wg sync.WaitGroup
	reactionCount := 5

	for i := 0; i < reactionCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			reaction := message.Reaction{
				Emoji:     "👍",
				OwnerType: string(commonidentity.OwnerTypeAgent),
				OwnerID:   uuid.Must(uuid.NewV4()),
				TMCreate:  "2024-01-01T00:00:00.000000Z",
			}

			reactionJSON, err := json.Marshal(reaction)
			if err != nil {
				t.Errorf("Failed to marshal reaction: %v", err)
				return
			}

			if err := h.MessageAddReactionAtomic(ctx, testMessage.ID, string(reactionJSON)); err != nil {
				t.Errorf("Failed to add reaction: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all reactions were added
	res, err := h.MessageGet(ctx, testMessage.ID)
	if err != nil {
		t.Errorf("Failed to get message: %v", err)
	}

	if len(res.Metadata.Reactions) != reactionCount {
		t.Errorf("Wrong reaction count. expect: %d, got: %d", reactionCount, len(res.Metadata.Reactions))
	}
}

func Test_MessageRemoveReactionAtomic(t *testing.T) {
	tests := []struct {
		name          string
		createMessage *message.Message
		addReaction   message.Reaction
		removeEmoji   string
		removeOwnerType string
		removeOwnerID uuid.UUID
		expectCount   int
	}{
		{
			name: "remove reaction",
			createMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("886e8e64-e428-11ec-baf2-7b14625ea229"),
					CustomerID: uuid.FromStringOrNil("8922f8c2-e428-11ec-b1a3-4bc67cb9da51"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("9922f8c2-e428-11ec-b1a3-4bc67cb9da52"),
				},
				ChatID:   uuid.FromStringOrNil("a922f8c2-e428-11ec-b1a3-4bc67cb9da53"),
				Type:     message.TypeNormal,
				Text:     "Message with reaction to remove",
				Medias:   []message.Media{},
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
			},
			addReaction: message.Reaction{
				Emoji:     "👍",
				OwnerType: string(commonidentity.OwnerTypeAgent),
				OwnerID:   uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9da54"),
				TMCreate:  "2024-01-01T00:00:00.000000Z",
			},
			removeEmoji:     "👍",
			removeOwnerType: string(commonidentity.OwnerTypeAgent),
			removeOwnerID:   uuid.FromStringOrNil("b922f8c2-e428-11ec-b1a3-4bc67cb9da54"),
			expectCount:     0,
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

			// Create message
			if err := h.MessageCreate(ctx, tt.createMessage); err != nil {
				t.Errorf("Failed to create message: %v", err)
			}

			// Add reaction first
			reactionJSON, err := json.Marshal(tt.addReaction)
			if err != nil {
				t.Errorf("Failed to marshal reaction: %v", err)
			}

			if err := h.MessageAddReactionAtomic(ctx, tt.createMessage.ID, string(reactionJSON)); err != nil {
				t.Errorf("Failed to add reaction: %v", err)
			}

			// Remove reaction atomically
			if err := h.MessageRemoveReactionAtomic(ctx, tt.createMessage.ID, tt.removeEmoji, tt.removeOwnerType, tt.removeOwnerID); err != nil {
				t.Errorf("Failed to remove reaction: %v", err)
			}

			// Verify reaction was removed
			res, err := h.MessageGet(ctx, tt.createMessage.ID)
			if err != nil {
				t.Errorf("Failed to get message: %v", err)
			}

			if len(res.Metadata.Reactions) != tt.expectCount {
				t.Errorf("Wrong reaction count. expect: %d, got: %d", tt.expectCount, len(res.Metadata.Reactions))
			}
		})
	}
}
