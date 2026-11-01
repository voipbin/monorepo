package subscribehandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
)

// The sentinel is a fake value that stands for a customer webhook secret.
const logSentinelW = "SENTINEL-W-MUST-NOT-LEAK-3d8a"

// newLogRedactionHook attaches a hook to the standard logger and enables the debug
// level, because the event line is Debug and would otherwise be dropped, which would
// make the assertion pass vacuously. The tests that use it must not run in parallel.
func newLogRedactionHook(t *testing.T) *logrustest.Hook {
	t.Helper()

	hook := logrustest.NewLocal(logrus.StandardLogger())
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetLevel(prev)
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})
	return hook
}

func decimalForm(s string) string {
	parts := make([]string, 0, len(s))
	for _, b := range []byte(s) {
		parts = append(parts, fmt.Sprint(b))
	}
	return strings.Join(parts, " ")
}

// assertNoSecretInEntries requires at least one entry whose message contains wantMsg and
// no entry that carries the secret, in plain text or as decimal bytes, in its message,
// its JSON-marshaled fields (the production formatter) or the %v of each field.
func assertNoSecretInEntries(t *testing.T, hook *logrustest.Hook, wantMsg, secret string) {
	t.Helper()

	found := 0
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, wantMsg) {
			found++
		}

		views := []string{entry.Message}
		if b, err := json.Marshal(entry.Data); err == nil {
			views = append(views, string(b))
		}
		for k, v := range entry.Data {
			views = append(views, k+"="+fmt.Sprintf("%v", v), k+"="+fmt.Sprintf("%+v", v))
		}

		for _, view := range views {
			if strings.Contains(view, secret) || strings.Contains(view, decimalForm(secret)) {
				t.Errorf("secret leaked into a log entry. level: %s, message: %q", entry.Level, entry.Message)
			}
		}
	}
	if found == 0 {
		t.Errorf("no log entry has the message %q, the check would be vacuous", wantMsg)
	}
}

func Test_processEvent_logsDoNotContainBody(t *testing.T) {
	errBoom := errors.New("boom")

	customerData := []byte(`{"id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc","webhook_secret":"` + logSentinelW + `"}`)
	brokenData := []byte(`[1,"` + logSentinelW + `"]`)

	tests := []struct {
		name string

		event    *sock.Event
		mockFunc func(m *agenthandler.MockAgentHandler)

		wantMsgs []string
	}{
		{
			name: "customer deleted",
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      customerData,
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantMsgs: []string{"Received subscribed event."},
		},
		{
			name: "customer created",
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerCreated),
				Data:      customerData,
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventCustomerCreated(gomock.Any(), gomock.Any(), false).Return(nil)
			},
			wantMsgs: []string{"Received subscribed event."},
		},
		{
			name: "customer deleted, handler error",
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      customerData,
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(errBoom)
			},
			wantMsgs: []string{"Could not handle the customer deleted event.", "Could not process the event correctly."},
		},
		{
			name: "customer created, handler error",
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerCreated),
				Data:      customerData,
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventCustomerCreated(gomock.Any(), gomock.Any(), false).Return(errBoom)
			},
			wantMsgs: []string{"Could not handle the customer created event.", "Could not process the event correctly."},
		},
		{
			name: "groupcall progressing, handler error",
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      string(cmgroupcall.EventTypeGroupcallProgressing),
				Data:      []byte(`{"id":"69434cfa-79a4-11ec-a7b1-6ba5b7016d83","note":"` + logSentinelW + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventGroupcallProgressing(gomock.Any(), gomock.Any()).Return(errBoom)
			},
			wantMsgs: []string{"Could not handle the groupcall answered event.", "Could not process the event correctly."},
		},
		{
			name: "customer deleted, broken data",
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      brokenData,
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {},
			wantMsgs: []string{"Could not unmarshal the data.", "Could not process the event correctly."},
		},
		{
			name: "customer created, broken data",
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerCreated),
				Data:      brokenData,
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {},
			wantMsgs: []string{"Could not unmarshal the data.", "Could not process the event correctly."},
		},
		{
			name: "groupcall progressing, broken data",
			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      string(cmgroupcall.EventTypeGroupcallProgressing),
				Data:      brokenData,
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {},
			wantMsgs: []string{"Could not unmarshal the data.", "Could not process the event correctly."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !bytes.Contains(tt.event.Data, []byte(logSentinelW)) {
				t.Fatalf("the event data must carry the sentinel")
			}

			hook := newLogRedactionHook(t)
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &subscribeHandler{agentHandler: mockAgent}
			tt.mockFunc(mockAgent)

			tt.event.DataType = "application/json"
			h.processEvent(tt.event)

			for _, msg := range tt.wantMsgs {
				assertNoSecretInEntries(t, hook, msg, logSentinelW)
			}
		})
	}
}

// Test_processEvent_receivedEventFields checks the values of the allowlisted fields.
func Test_processEvent_receivedEventFields(t *testing.T) {
	hook := newLogRedactionHook(t)
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAgent := agenthandler.NewMockAgentHandler(mc)
	mockAgent.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
	h := &subscribeHandler{agentHandler: mockAgent}

	h.processEvent(&sock.Event{
		Publisher: string(commonoutline.ServiceNameCustomerManager),
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		DataType:  "application/json",
		Data:      []byte(`{"id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc","webhook_secret":"` + logSentinelW + `"}`),
	})

	var got *logrus.Entry
	for _, entry := range hook.AllEntries() {
		if strings.HasPrefix(entry.Message, "Received subscribed event.") {
			got = entry
			break
		}
	}
	if got == nil {
		t.Fatalf("no received event entry")
	}

	expect := map[string]interface{}{
		"publisher": string(commonoutline.ServiceNameCustomerManager),
		"type":      string(cmcustomer.EventTypeCustomerDeleted),
	}
	for k, v := range expect {
		if got.Data[k] != v {
			t.Errorf("Wrong field %s. expect: %v, got: %v", k, v, got.Data[k])
		}
	}
	for _, k := range []string{"event", "message"} {
		if _, ok := got.Data[k]; ok {
			t.Errorf("the %s object must not be a log field", k)
		}
	}
}
