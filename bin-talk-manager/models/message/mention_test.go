package message

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_ExtractAgentMentions(t *testing.T) {
	a1 := "11111111-1111-4111-8111-111111111111"
	a2 := "22222222-2222-4222-8222-222222222222"
	author := "99999999-9999-4999-8999-999999999999"

	manyText := ""
	manyExpect := []uuid.UUID{}
	for i := 0; i < 60; i++ {
		id := uuid.FromStringOrNil(fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1))
		manyText += fmt.Sprintf("[@a%d](mention:%s) ", i, id)
		if i < maxMentionsPerMessage {
			manyExpect = append(manyExpect, id)
		}
	}

	tests := []struct {
		name   string
		text   string
		expect []uuid.UUID
	}{
		{"no mention", "hello world", nil},
		{"empty", "", nil},
		{"one agent mention", "hi [@Alice](mention:" + a1 + ")", []uuid.UUID{uuid.FromStringOrNil(a1)}},
		{"two agents keep the order of first appearance", "[@B](mention:" + a2 + ") and [@A](mention:" + a1 + ")", []uuid.UUID{uuid.FromStringOrNil(a2), uuid.FromStringOrNil(a1)}},
		{"contact mention is excluded", "[@Bob](mention:contact:" + a1 + ")", nil},
		{"contact and agent mention", "[@Bob](mention:contact:" + a1 + ") [@A](mention:" + a2 + ")", []uuid.UUID{uuid.FromStringOrNil(a2)}},
		{"duplicate mention is recorded once", "[@A](mention:" + a1 + ") [@A again](mention:" + a1 + ")", []uuid.UUID{uuid.FromStringOrNil(a1)}},
		{"duplicate with different case is recorded once", "[@A](mention:" + a1 + ") [@A](mention:" + strings.ToUpper(a1) + ")", []uuid.UUID{uuid.FromStringOrNil(a1)}},
		{"self mention is excluded", "[@me](mention:" + author + ")", nil},
		{"self mention excluded, other kept", "[@me](mention:" + author + ") [@A](mention:" + a1 + ")", []uuid.UUID{uuid.FromStringOrNil(a1)}},
		{"name containing a bracket is not a mention", "[@A]B](mention:" + a1 + ")", nil},
		{"invalid uuid", "[@A](mention:not-a-uuid)", nil},
		{"uuid too short", "[@A](mention:1111-2222)", nil},
		{"uuid with trailing characters", "[@A](mention:" + a1 + "x)", nil},
		{"missing at sign", "[A](mention:" + a1 + ")", nil},
		{"empty display name", "[@](mention:" + a1 + ")", nil},
		{"nil uuid is accepted by shape but not a valid agent", "[@A](mention:00000000-0000-0000-0000-000000000000)", nil},
		{"capped at 50 distinct agents", manyText, manyExpect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractAgentMentions(tt.text, uuid.FromStringOrNil(author))
			if len(got) == 0 && len(tt.expect) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, got)
			}
		})
	}
}
