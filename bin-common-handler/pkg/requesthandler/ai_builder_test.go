package requesthandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/builder"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

var builderTestCustomerID = uuid.FromStringOrNil("83fec56f-8e28-4356-a50c-7641e39ed2df")

func Test_AIV1BuilderChat(t *testing.T) {
	req := &builder.ChatRequest{
		Messages:     []builder.Message{{Role: builder.RoleUser, Content: "I run a clinic"}},
		CurrentDraft: &builder.Draft{Name: "Clinic desk"},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := requestHandler{sock: mockSock}

	// The customer id is part of the request body so ai-manager charges the
	// daily counter to the right customer; the draft travels with it.
	expectBody := `{"customer_id":"83fec56f-8e28-4356-a50c-7641e39ed2df","messages":[{"role":"user","content":"I run a clinic"}],"current_draft":{"name":"Clinic desk","detail":"","init_prompt":"","tool_names":null}}`
	mockSock.EXPECT().RequestPublish(deadlineNear(t, 55*time.Second), string(outline.QueueNameAIRequest), &sock.Request{
		URI:      "/v1/ai_builder/chat",
		Method:   sock.RequestMethodPost,
		DataType: ContentTypeJSON,
		Data:     []byte(expectBody),
	}).Return(&sock.Response{
		StatusCode: 200,
		DataType:   ContentTypeJSON,
		Data:       []byte(`{"message":"What happens when nobody answers?"}`),
	}, nil)

	res, err := h.AIV1BuilderChat(context.Background(), builderTestCustomerID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(res, &builder.ChatResponse{Message: "What happens when nobody answers?"}) {
		t.Errorf("got %+v", res)
	}
}

// The wait is 55 seconds: ai-manager's LLM deadline is 40, so a normal slow
// answer must not time out on this side first. The value is passed to
// sendRequest in milliseconds.
func Test_AIV1BuilderChat_timeout(t *testing.T) {
	if builderChatTimeout != 55000 {
		t.Errorf("chat timeout: got %d ms, want 55000", builderChatTimeout)
	}
	if builderStatusTimeout != 3000 {
		t.Errorf("status timeout: got %d ms, want 3000", builderStatusTimeout)
	}
}

// The typed error ai-manager sends must survive the trip, reason included: the
// client acts on the reason.
func Test_AIV1BuilderChat_typedErrorKeepsItsReason(t *testing.T) {
	for _, reason := range []string{builder.ReasonDailyLimit, builder.ReasonBusy, builder.ReasonTimeout, builder.ReasonResponseInvalid, builder.ReasonUnavailable, builder.ReasonDisabled} {
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

			_, got := h.AIV1BuilderChat(context.Background(), builderTestCustomerID, &builder.ChatRequest{})
			var out *cerrors.VoipbinError
			if !errors.As(got, &out) || out.Reason != reason {
				t.Errorf("the reason must survive: got %v", got)
			}
		})
	}
}

func Test_AIV1BuilderChat_transportErrorIsReturned(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := requestHandler{sock: mockSock}
	mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, context.DeadlineExceeded)

	_, err := h.AIV1BuilderChat(context.Background(), builderTestCustomerID, &builder.ChatRequest{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("the deadline must stay detectable with errors.Is, got %v", err)
	}
}

func Test_AIV1BuilderStatus(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(deadlineNear(t, 3*time.Second), string(outline.QueueNameAIRequest), &sock.Request{
		URI:    "/v1/ai_builder/status",
		Method: sock.RequestMethodGet,
	}).Return(&sock.Response{
		StatusCode: 200,
		DataType:   ContentTypeJSON,
		Data:       []byte(`{"available":true,"max_messages":40,"max_message_chars":2000}`),
	}, nil)

	res, err := h.AIV1BuilderStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Available || res.MaxMessages != 40 || res.MaxMessageChars != 2000 {
		t.Errorf("got %+v", res)
	}
}

// A body that is not JSON is an error, not an empty success.
func Test_AIV1BuilderStatus_garbageBody(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := requestHandler{sock: mockSock}
	mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(&sock.Response{StatusCode: 200, DataType: ContentTypeJSON, Data: []byte("{not json")}, nil)

	if res, err := h.AIV1BuilderStatus(context.Background()); err == nil {
		t.Errorf("expected an error, got %+v", res)
	}
}

// Nothing the customer wrote may be an error string: a failure to parse the reply, say,
// must not quote the draft.
func Test_AIV1BuilderChat_errorTextHasNoInput(t *testing.T) {
	const secret = "SECRET-INPUT-STRING-do-not-log"
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := requestHandler{sock: mockSock}
	mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(&sock.Response{StatusCode: 200, DataType: ContentTypeJSON, Data: []byte("{" + secret)}, nil)

	_, err := h.AIV1BuilderChat(context.Background(), builderTestCustomerID, &builder.ChatRequest{Messages: []builder.Message{{Role: builder.RoleUser, Content: secret}}})
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("the error text carries the input: %v", err)
	}
}

// deadlineNear matches a context whose deadline is about d from now. It proves
// the timeout constants reach the context handed to the socket, not just exist.
func deadlineNear(t *testing.T, d time.Duration) gomock.Matcher {
	t.Helper()
	return gomock.Cond(func(x any) bool {
		ctx, ok := x.(context.Context)
		if !ok {
			return false
		}
		dl, has := ctx.Deadline()
		if !has {
			return false
		}
		left := time.Until(dl)
		return left > d-2*time.Second && left <= d
	})
}
