package message

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func TestMetadataStruct(t *testing.T) {
	ownerID := uuid.Must(uuid.NewV4())

	metadata := Metadata{
		Reactions: []Reaction{
			{
				Emoji:     "👍",
				OwnerType: "agent",
				OwnerID:   ownerID,
				TMCreate:  "2023-01-01 00:00:00",
			},
			{
				Emoji:     "❤️",
				OwnerType: "user",
				OwnerID:   uuid.Must(uuid.NewV4()),
				TMCreate:  "2023-01-01 00:01:00",
			},
		},
	}

	if len(metadata.Reactions) != 2 {
		t.Errorf("Metadata.Reactions length = %v, expected %v", len(metadata.Reactions), 2)
	}
	if metadata.Reactions[0].Emoji != "👍" {
		t.Errorf("Metadata.Reactions[0].Emoji = %v, expected %v", metadata.Reactions[0].Emoji, "👍")
	}
	if metadata.Reactions[0].OwnerType != "agent" {
		t.Errorf("Metadata.Reactions[0].OwnerType = %v, expected %v", metadata.Reactions[0].OwnerType, "agent")
	}
	if metadata.Reactions[0].OwnerID != ownerID {
		t.Errorf("Metadata.Reactions[0].OwnerID = %v, expected %v", metadata.Reactions[0].OwnerID, ownerID)
	}
}

func TestReactionStruct(t *testing.T) {
	ownerID := uuid.Must(uuid.NewV4())

	reaction := Reaction{
		Emoji:     "🎉",
		OwnerType: "agent",
		OwnerID:   ownerID,
		TMCreate:  "2023-01-01 00:00:00",
	}

	if reaction.Emoji != "🎉" {
		t.Errorf("Reaction.Emoji = %v, expected %v", reaction.Emoji, "🎉")
	}
	if reaction.OwnerType != "agent" {
		t.Errorf("Reaction.OwnerType = %v, expected %v", reaction.OwnerType, "agent")
	}
	if reaction.OwnerID != ownerID {
		t.Errorf("Reaction.OwnerID = %v, expected %v", reaction.OwnerID, ownerID)
	}
	if reaction.TMCreate != "2023-01-01 00:00:00" {
		t.Errorf("Reaction.TMCreate = %v, expected %v", reaction.TMCreate, "2023-01-01 00:00:00")
	}
}

func TestMetadataMarshalUnmarshal(t *testing.T) {
	ownerID := uuid.Must(uuid.NewV4())

	metadata := Metadata{
		Reactions: []Reaction{
			{
				Emoji:     "👍",
				OwnerType: "agent",
				OwnerID:   ownerID,
				TMCreate:  "2023-01-01 00:00:00",
			},
		},
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		t.Errorf("Failed to marshal metadata: %v", err)
		return
	}

	var result Metadata
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("Failed to unmarshal metadata: %v", err)
		return
	}

	if !reflect.DeepEqual(metadata, result) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", metadata, result)
	}
}

func TestMetadataEmptyReactions(t *testing.T) {
	metadata := Metadata{
		Reactions: []Reaction{},
	}

	if len(metadata.Reactions) != 0 {
		t.Errorf("Metadata.Reactions length = %v, expected %v", len(metadata.Reactions), 0)
	}
}
