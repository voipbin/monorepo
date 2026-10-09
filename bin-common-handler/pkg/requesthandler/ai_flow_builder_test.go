package requesthandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_AIV1FlowBuilderChat(t *testing.T) {
	req := &flowbuilder.ChatRequest{
		Messages:             []flowbuilder.Message{{Role: flowbuilder.RoleUser, Content: "greeting flow"}},
		SupportedActionTypes: []string{"talk", "hangup"},
		CurrentDraft: &flowbuilder.Draft{
			Actions: []map[string]any{{"id": "a", "type": "talk"}},
			Labels:  map[string]string{"a": "greet"},
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := requestHandler{sock: mockSock}

	// The marshaled wire shape: the customer id is part of the body (the
	// daily counter is charged to it), and the supported types and the draft
	// travel with it.
	expectBody := `{"customer_id":"83fec56f-8e28-4356-a50c-7641e39ed2df","messages":[{"role":"user","content":"greeting flow"}],"current_draft":{"actions":[{"id":"a","type":"talk"}],"labels":{"a":"greet"}},"supported_action_types":["talk","hangup"]}`
	mockSock.EXPECT().RequestPublish(deadlineNear(t, 55*time.Second), string(outline.QueueNameAIRequest), &sock.Request{
		URI:      "/v1/flow_builder/chat",
		Method:   sock.RequestMethodPost,
		DataType: ContentTypeJSON,
		Data:     []byte(expectBody),
	}).Return(&sock.Response{
		StatusCode: 200,
		DataType:   ContentTypeJSON,
		Data:       []byte(`{"message":"Which channel?"}`),
	}, nil)

	res, err := h.AIV1FlowBuilderChat(context.Background(), builderTestCustomerID, req)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, &flowbuilder.ChatResponse{Message: "Which channel?"}) {
		t.Errorf("Wrong match. got %+v", res)
	}
}

func Test_AIV1FlowBuilderChat_responseCarriesDraftAndSensitiveNodes(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(&sock.Response{
		StatusCode: 200,
		DataType:   ContentTypeJSON,
		Data:       []byte(`{"message":"m","draft":{"actions":[{"id":"6c73ff34-7f4c-11ec-b4d5-5b94d40e4071","type":"message_send"}],"labels":{}},"draft_warnings":["open_end: a"],"sensitive_nodes":["6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"]}`),
	}, nil)

	res, err := h.AIV1FlowBuilderChat(context.Background(), builderTestCustomerID, &flowbuilder.ChatRequest{})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Draft == nil || len(res.Draft.Actions) != 1 || len(res.SensitiveNodes) != 1 || len(res.DraftWarnings) != 1 {
		t.Errorf("Wrong match. got %+v", res)
	}
}

func Test_AIV1FlowBuilderChat_typedErrorKeepsItsReason(t *testing.T) {
	for _, reason := range []string{builder.ReasonDailyLimit, builder.ReasonBusy, builder.ReasonTimeout, builder.ReasonResponseInvalid, builder.ReasonUnavailable} {
		t.Run(reason, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{sock: mockSock}

			ve := cerrors.Unavailable(outline.ServiceNameAIManager, reason, "x")
			wire, err := cerrors.ToResponse(ve)
			if err != nil {
				t.Fatal(err)
			}
			mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(wire, nil)

			_, got := h.AIV1FlowBuilderChat(context.Background(), builderTestCustomerID, &flowbuilder.ChatRequest{})
			var out *cerrors.VoipbinError
			if !errors.As(got, &out) || out.Reason != reason {
				t.Errorf("Wrong match. the reason must survive: got %v", got)
			}
		})
	}
}

func Test_AIV1FlowBuilderChat_transportErrorIsReturnedWithoutInput(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := requestHandler{sock: mockSock}
	mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, context.DeadlineExceeded)

	req := &flowbuilder.ChatRequest{Messages: []flowbuilder.Message{{Role: flowbuilder.RoleUser, Content: "SECRET-INPUT"}}}
	_, err := h.AIV1FlowBuilderChat(context.Background(), builderTestCustomerID, req)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Wrong match. the deadline must stay detectable with errors.Is, got %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "SECRET-INPUT") {
		t.Errorf("Wrong match. the input reached the error: %v", err)
	}
}
