// Package settings holds the notification settings of an agent.
package settings

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-push-manager/models/notification"
)

// Settings is the notification settings of an agent: which notification kinds are on and which chats are muted.
// Everything is on and nothing is muted until the agent changes it, so a new kind is on for everybody.
type Settings struct {
	// Kinds has every kind the agent can switch (notification.SettingsKinds): true is on.
	Kinds map[notification.Kind]bool `json:"kinds"`

	// MutedChatIDs is the chats the agent muted. It is never nil, so it is a JSON list even when empty.
	MutedChatIDs []uuid.UUID `json:"muted_chat_ids"`
}

// New builds the settings from what is stored: the kinds that are off and the muted chats. A kind that the agent
// can not switch is ignored.
func New(disabled []notification.Kind, muted []uuid.UUID) *Settings {
	res := &Settings{
		Kinds:        make(map[notification.Kind]bool, len(notification.SettingsKinds())),
		MutedChatIDs: []uuid.UUID{},
	}

	for _, k := range notification.SettingsKinds() {
		res.Kinds[k] = true
	}
	for _, k := range disabled {
		if notification.IsSettingsKind(k) {
			res.Kinds[k] = false
		}
	}

	res.MutedChatIDs = append(res.MutedChatIDs, muted...)

	return res
}
