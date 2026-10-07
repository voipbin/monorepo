package servicehandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
)

func flowBuilderChatReq(content string) *flowbuilder.ChatRequest {
	return &flowbuilder.ChatRequest{
		Messages:             []flowbuilder.Message{{Role: flowbuilder.RoleUser, Content: content}},
		SupportedActionTypes: []string{"talk", "hangup"},
	}
}

func Test_FlowBuilderChat_success(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)

	// The customer id sent to ai-manager is the AUTHENTICATED customer.
	mockReq.EXPECT().AIV1FlowBuilderChat(gomock.Any(), uuid.FromStringOrNil(builderCustomer), gomock.Any()).
		Return(&flowbuilder.ChatResponse{Message: "hi"}, nil)

	res, err := h.FlowBuilderChat(context.Background(), a, flowBuilderChatReq("hello"))
	if err != nil || res.Message != "hi" {
		t.Fatalf("Wrong match. got %+v / %v", res, err)
	}
}

func Test_FlowBuilderChat_permissions(t *testing.T) {
	tests := []struct {
		name   string
		a      *auth.AuthIdentity
		expect error
	}{
		{"admin", builderAgent(amagent.PermissionCustomerAdmin, builderCustomer), nil},
		{"manager", builderAgent(amagent.PermissionCustomerManager, builderCustomer), nil},
		{"plain agent", builderAgent(amagent.PermissionNone, builderCustomer), serviceerrors.ErrPermissionDenied},
		{"accesskey", builderAccesskey(), serviceerrors.ErrPermissionDenied},
		{"direct", auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil(builderCustomer)}), serviceerrors.ErrPermissionDenied},
		{"delegate", auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: uuid.FromStringOrNil(builderCustomer)}), serviceerrors.ErrPermissionDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockReq := newBuilderServiceHandler(t)
			if tt.expect == nil {
				mockReq.EXPECT().AIV1FlowBuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(&flowbuilder.ChatResponse{Message: "ok"}, nil)
			}
			_, err := h.FlowBuilderChat(context.Background(), tt.a, flowBuilderChatReq("hello")) // strict mock: no RPC when denied
			if (tt.expect == nil && err != nil) || (tt.expect != nil && !errors.Is(err, tt.expect)) {
				t.Errorf("Wrong match. got %v, want %v", err, tt.expect)
			}
		})
	}
}

func Test_FlowBuilderChat_validatesBeforeTheRPC(t *testing.T) {
	h, _ := newBuilderServiceHandler(t) // strict mock
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)

	for name, req := range map[string]*flowbuilder.ChatRequest{
		"empty":               {},
		"no supported types":  {Messages: []flowbuilder.Message{{Role: flowbuilder.RoleUser, Content: "x"}}},
		"too many node draft": {Messages: []flowbuilder.Message{{Role: flowbuilder.RoleUser, Content: "x"}}, SupportedActionTypes: []string{"talk"}, CurrentDraft: &flowbuilder.Draft{Actions: make([]map[string]any, flowbuilder.MaxFlowNodes+1)}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := h.FlowBuilderChat(context.Background(), a, req)
			var ve *cerrors.VoipbinError
			if !errors.As(err, &ve) || ve.Reason != builder.ReasonInvalidArgument {
				t.Errorf("Wrong match. got %v", err)
			}
		})
	}
}

// The two transport counters are the Flow Builder's own: the Assistant
// Builder's series must not move for a Flow turn.
func Test_FlowBuilderChat_timeoutIsCountedInTheFlowSeriesOnly(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	mockReq.EXPECT().AIV1FlowBuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.Join(errors.New("rpc"), context.DeadlineExceeded))

	flowBefore, assistantBefore := testutil.ToFloat64(metricFlowBuilderTimeout), testutil.ToFloat64(metricBuilderTimeout)
	_, err := h.FlowBuilderChat(context.Background(), a, flowBuilderChatReq("hello"))

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Reason != builder.ReasonTimeout || ve.Status != cerrors.StatusUnavailable {
		t.Fatalf("Wrong match. got %v", err)
	}
	if !strings.Contains(ve.Message, "flow builder") {
		t.Errorf("Wrong match. expect the flow builder wording, got: %s", ve.Message)
	}
	if testutil.ToFloat64(metricFlowBuilderTimeout)-flowBefore != 1 {
		t.Error("Wrong match. the flow timeout counter must rise by one")
	}
	if testutil.ToFloat64(metricBuilderTimeout) != assistantBefore {
		t.Error("Wrong match. the assistant timeout counter must not move")
	}
}

func Test_FlowBuilderChat_circuitOpenIsCountedInTheFlowSeriesOnly(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	mockReq.EXPECT().AIV1FlowBuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.Join(errors.New("rpc"), circuitbreakerhandler.ErrCircuitOpen))

	flowBefore, assistantBefore := testutil.ToFloat64(metricFlowBuilderCircuitOpen), testutil.ToFloat64(metricBuilderCircuitOpen)
	_, err := h.FlowBuilderChat(context.Background(), a, flowBuilderChatReq("hello"))

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Reason != "SERVICE_UNAVAILABLE" {
		t.Fatalf("Wrong match. got %v", err)
	}
	if testutil.ToFloat64(metricFlowBuilderCircuitOpen)-flowBefore != 1 {
		t.Error("Wrong match. the flow circuit counter must rise by one")
	}
	if testutil.ToFloat64(metricBuilderCircuitOpen) != assistantBefore {
		t.Error("Wrong match. the assistant circuit counter must not move")
	}
}

func Test_FlowBuilderChat_typedErrorPassesThroughAndCancelIsNotCounted(t *testing.T) {
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)

	t.Run("typed", func(t *testing.T) {
		h, mockReq := newBuilderServiceHandler(t)
		typed := cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonDailyLimit, "x")
		mockReq.EXPECT().AIV1FlowBuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, typed)

		_, err := h.FlowBuilderChat(context.Background(), a, flowBuilderChatReq("hello"))
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) || ve.Reason != builder.ReasonDailyLimit {
			t.Errorf("Wrong match. got %v", err)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		h, mockReq := newBuilderServiceHandler(t)
		mockReq.EXPECT().AIV1FlowBuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, context.Canceled)

		before := testutil.ToFloat64(metricFlowBuilderTimeout)
		_, err := h.FlowBuilderChat(context.Background(), a, flowBuilderChatReq("hello"))
		if err == nil || testutil.ToFloat64(metricFlowBuilderTimeout) != before {
			t.Errorf("Wrong match. a cancel is an error and is not counted: %v", err)
		}
	})
}

func Test_FlowBuilderChat_neverLogsTheInput(t *testing.T) {
	var buf strings.Builder
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	for name, rpcErr := range map[string]error{
		"ok":        nil,
		"echoing":   errors.New("400: " + builderSecret),
		"timeout":   context.DeadlineExceeded,
		"circuit":   circuitbreakerhandler.ErrCircuitOpen,
		"typed err": cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "x").Wrap(errors.New(builderSecret)),
	} {
		t.Run(name, func(t *testing.T) {
			buf.Reset()
			h, mockReq := newBuilderServiceHandler(t)
			a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
			if rpcErr == nil {
				mockReq.EXPECT().AIV1FlowBuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(&flowbuilder.ChatResponse{Message: "ok"}, nil)
			} else {
				mockReq.EXPECT().AIV1FlowBuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, rpcErr)
			}
			req := flowBuilderChatReq(builderSecret)
			req.CurrentDraft = &flowbuilder.Draft{Actions: []map[string]any{{"id": "a", "option": map[string]any{"text": builderSecret}}}}
			_, _ = h.FlowBuilderChat(context.Background(), a, req)
			if strings.Contains(buf.String(), builderSecret) {
				t.Errorf("Wrong match. a log line carries the input:\n%s", buf.String())
			}
		})
	}
}

// Both builders are one rule for who may use them, and the Assistant Builder's
// wording for its own timeout is unchanged by the shared mapper.
func Test_AIBuilderChat_timeoutWordingIsUnchanged(t *testing.T) {
	h, mockReq := newBuilderServiceHandler(t)
	a := builderAgent(amagent.PermissionCustomerAdmin, builderCustomer)
	mockReq.EXPECT().AIV1BuilderChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, context.DeadlineExceeded)

	_, err := h.AIBuilderChat(context.Background(), a, builderChatReq("hello"))
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Message != "The assistant builder took too long. Please try again." {
		t.Errorf("Wrong match. got %v", err)
	}
}
