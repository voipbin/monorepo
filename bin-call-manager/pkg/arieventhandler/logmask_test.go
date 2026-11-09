package arieventhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/channelhandler"
)

const testTicketBody = "0123456789abcdef0123456789abcdef"

func Test_maskPushRingTicket(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		expect string
	}{
		{"ticket", "direct." + testTicketBody, "direct.0123****"},
		{"ticket inside text", "exten=direct." + testTicketBody + ";x", "exten=direct.0123****;x"},
		{"direct hash is not a ticket", "direct.0123456789abcdef", "direct.0123456789abcdef"},
		{"plain number", "011441332323027", "011441332323027"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskPushRingTicket(tt.in); got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

// The ticket reaches the channel info untouched and never reaches a log field or message.
func Test_EventHandlerChannelCreated_ticketNotLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := logrustest.NewLocal(logrus.StandardLogger())
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetLevel(prev)
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})

	mockChannel := channelhandler.NewMockChannelHandler(mc)
	h := eventHandler{channelHandler: mockChannel}

	ticket := "direct." + testTicketBody
	evt := &ari.ChannelCreated{
		Event: ari.Event{Type: ari.EventTypeChannelCreated, AsteriskID: "42:01:0a:a4:00:05"},
		Channel: ari.Channel{
			ID:       "1587307080.49",
			Name:     "PJSIP/in-voipbin-00000030",
			Dialplan: ari.DialplanCEP{Context: "in-voipbin", Exten: ticket, Priority: 1},
		},
	}

	mockChannel.EXPECT().Create(
		gomock.Any(), evt.Channel.ID, evt.AsteriskID, evt.Channel.Name, channel.TypeNone, gomock.Any(),
		gomock.Any(), gomock.Any(), "", ticket, gomock.Any(),
	).Return(&channel.Channel{ID: evt.Channel.ID, DestinationNumber: ticket}, nil)

	if err := h.EventHandlerChannelCreated(context.Background(), evt); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	found := false
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "Created a channel info") {
			found = true
		}
		b, err := json.Marshal(entry.Data)
		if err != nil {
			t.Fatalf("could not marshal the log fields: %v", err)
		}
		for _, s := range []string{string(b), fmt.Sprintf("%v", entry.Data) + entry.Message} {
			if strings.Contains(s, testTicketBody) {
				t.Errorf("the ticket leaked into the log entry %q: %s", entry.Message, s)
			}
		}
	}
	if !found {
		t.Errorf("the debug log under test was not collected")
	}
	if evt.Channel.Dialplan.Exten != ticket {
		t.Errorf("the event must not be changed: %s", evt.Channel.Dialplan.Exten)
	}
}
