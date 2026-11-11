package settings

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-push-manager/models/notification"
)

func Test_New(t *testing.T) {
	chatA := uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11")

	t.Run("nothing stored means everything on and no mute", func(t *testing.T) {
		res := New(nil, nil)
		if len(res.Kinds) != 5 {
			t.Fatalf("Wrong match. expect: 5 kinds, got: %v", res.Kinds)
		}
		for _, k := range notification.SettingsKinds() {
			if !res.Kinds[k] {
				t.Errorf("Wrong match. %s must be on", k)
			}
		}
		if res.MutedChatIDs == nil || len(res.MutedChatIDs) != 0 {
			t.Errorf("Wrong match. expect: an empty, non nil list, got: %#v", res.MutedChatIDs)
		}

		m, err := json.Marshal(res)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		expect := `{"kinds":{"case_assigned":true,"case_note":true,"chat_mentioned":true,"chat_message":true,"conversation_message":true},"muted_chat_ids":[]}`
		if string(m) != expect {
			t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, m)
		}
	})

	t.Run("a disabled kind is off and a call kind is ignored", func(t *testing.T) {
		res := New([]notification.Kind{notification.KindChatMessage, notification.KindCallIncoming}, []uuid.UUID{chatA})
		if res.Kinds[notification.KindChatMessage] {
			t.Errorf("Wrong match. chat_message must be off")
		}
		if !res.Kinds[notification.KindCaseNote] {
			t.Errorf("Wrong match. case_note must stay on")
		}
		if _, ok := res.Kinds[notification.KindCallIncoming]; ok {
			t.Errorf("Wrong match. a call kind is not a setting")
		}
		if len(res.MutedChatIDs) != 1 || res.MutedChatIDs[0] != chatA {
			t.Errorf("Wrong match. got: %v", res.MutedChatIDs)
		}
	})
}
