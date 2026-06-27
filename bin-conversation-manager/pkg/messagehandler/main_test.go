package messagehandler

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
)

func Test_DeriveEndpoints(t *testing.T) {

	addrA := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+11111"}
	addrB := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+22222"}

	tests := []struct {
		name string

		conversation *conversation.Conversation
		direction    message.Direction

		expectedSource      commonaddress.Address
		expectedDestination commonaddress.Address
	}{
		{
			name: "outgoing maps self to source and peer to destination",

			conversation: &conversation.Conversation{
				Self: addrA,
				Peer: addrB,
			},
			direction: message.DirectionOutgoing,

			expectedSource:      addrA,
			expectedDestination: addrB,
		},
		{
			name: "incoming maps peer to source and self to destination",

			conversation: &conversation.Conversation{
				Self: addrA,
				Peer: addrB,
			},
			direction: message.DirectionIncoming,

			expectedSource:      addrB,
			expectedDestination: addrA,
		},
		{
			name: "unknown direction returns zero endpoints",

			conversation: &conversation.Conversation{
				Self: addrA,
				Peer: addrB,
			},
			direction: message.DirectionNond,

			expectedSource:      commonaddress.Address{},
			expectedDestination: commonaddress.Address{},
		},
		{
			name: "nil conversation returns zero endpoints",

			conversation: nil,
			direction:    message.DirectionOutgoing,

			expectedSource:      commonaddress.Address{},
			expectedDestination: commonaddress.Address{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, destination := DeriveEndpoints(tt.conversation, tt.direction)

			if !reflect.DeepEqual(tt.expectedSource, source) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedSource, source)
			}
			if !reflect.DeepEqual(tt.expectedDestination, destination) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedDestination, destination)
			}
		})
	}
}
