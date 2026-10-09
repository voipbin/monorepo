package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

func flowBuilderBody(content string) []byte {
	b, _ := json.Marshal(map[string]any{
		"messages":               []map[string]string{{"role": "user", "content": content}},
		"supported_action_types": []string{"talk", "hangup"},
	})
	return b
}

func Test_PostFlowBuilderChat_success(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := builderServerAgent()

	mockSvc.EXPECT().FlowBuilderChat(gomock.Any(), a, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *auth.AuthIdentity, req *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error) {
			if len(req.Messages) != 1 || req.Messages[0].Content != "hello" || len(req.SupportedActionTypes) != 2 {
				t.Errorf("Wrong match. the body did not reach the service intact: %+v", req)
			}
			return &flowbuilder.ChatResponse{
				Message:        "hi",
				SensitiveNodes: []uuid.UUID{uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071")},
			}, nil
		})

	w := serveBuilder(t, mockSvc, a, "POST", "/flow_builder/chat", flowBuilderBody("hello"))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"message":"hi"`) || !strings.Contains(w.Body.String(), `"sensitive_nodes":["6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"]`) {
		t.Errorf("Wrong match. got %d %s", w.Code, w.Body.String())
	}
}

// The draft is the contract with the editor: every action key, including the
// open-ended option, the next_id and the labels, must reach the
// service exactly as the client sent it.
func Test_PostFlowBuilderChat_passesTheDraftOnWithoutLosingAnything(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := builderServerAgent()

	body := []byte(`{
	  "messages":[{"role":"user","content":"first"},{"role":"assistant","content":"second"},{"role":"user","content":"third"}],
	  "supported_action_types":["talk","branch","hangup"],
	  "current_draft":{
	    "actions":[
	      {"id":"6c73ff34-7f4c-11ec-b4d5-5b94d40e4071","type":"talk","option":{"text":"Hi","language":"en-US","async":true},"next_id":"841c5fa2-f0c2-11ee-834f-53b2b00ec88d"},
	      {"id":"841c5fa2-f0c2-11ee-834f-53b2b00ec88d","type":"branch","option":{"variable":"v","target_ids":{"1":"6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"},"default_target_id":"6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"}}
	    ],
	    "labels":{"6c73ff34-7f4c-11ec-b4d5-5b94d40e4071":"greet"}
	  }
	}`)

	mockSvc.EXPECT().FlowBuilderChat(gomock.Any(), a, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *auth.AuthIdentity, got *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error) {
			d := got.CurrentDraft
			if d == nil || len(d.Actions) != 2 {
				t.Fatalf("Wrong match. the draft was not passed on: %+v", got)
			}
			wantOption := map[string]any{"text": "Hi", "language": "en-US", "async": true}
			if !reflect.DeepEqual(d.Actions[0]["option"], wantOption) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", wantOption, d.Actions[0]["option"])
			}
			if d.Actions[0]["next_id"] != "841c5fa2-f0c2-11ee-834f-53b2b00ec88d" || d.Actions[0]["type"] != "talk" {
				t.Errorf("Wrong match. next_id or type lost: %+v", d.Actions[0])
			}
			branch := d.Actions[1]["option"].(map[string]any)
			if !reflect.DeepEqual(branch["target_ids"], map[string]any{"1": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"}) {
				t.Errorf("Wrong match. the branch targets were lost: %v", branch)
			}
			if d.Labels["6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"] != "greet" {
				t.Errorf("Wrong match. labels: %+v", d.Labels)
			}
			if len(got.Messages) != 3 || got.Messages[1].Role != "assistant" {
				t.Errorf("Wrong match. messages: %+v", got.Messages)
			}
			return &flowbuilder.ChatResponse{Message: "ok"}, nil
		})

	w := serveBuilder(t, mockSvc, a, "POST", "/flow_builder/chat", body)
	if w.Code != http.StatusOK {
		t.Fatalf("Wrong match. got %d %s", w.Code, w.Body.String())
	}
}

func Test_PostFlowBuilderChat_nonAgentIsRefusedWith403AndNeverReachesTheService(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc) // strict: no service call
	for name, id := range map[string]*auth.AuthIdentity{
		"accesskey": builderServerAccesskey(),
		"direct":    auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")}),
		"delegate":  auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")}),
	} {
		t.Run(name, func(t *testing.T) {
			w := serveBuilder(t, mockSvc, id, "POST", "/flow_builder/chat", flowBuilderBody("hello"))
			if w.Code != http.StatusForbidden {
				t.Errorf("Wrong match. got %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func Test_PostFlowBuilderChat_noIdentityIs401(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	w := serveBuilder(t, mockSvc, nil, "POST", "/flow_builder/chat", flowBuilderBody("hello"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong match. got %d", w.Code)
	}
}

func Test_PostFlowBuilderChat_badJSONIs400AndEchoesNothing(t *testing.T) {
	var buf strings.Builder
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	w := serveBuilder(t, mockSvc, builderServerAgent(), "POST", "/flow_builder/chat", []byte("{"+builderServerSecret))
	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong match. got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), builderServerSecret) || strings.Contains(buf.String(), builderServerSecret) {
		t.Errorf("Wrong match. the body was echoed:\nresponse: %s\nlog: %s", w.Body.String(), buf.String())
	}
	if strings.Contains(buf.String(), "err:") || strings.Contains(buf.String(), "invalid character") {
		t.Errorf("Wrong match. the log carries the underlying json error:\n%s", buf.String())
	}
}

// The cap is 512 KiB, not the Assistant Builder's 160 KiB, because a draft
// travels with the conversation.
func Test_PostFlowBuilderChat_bodyCap(t *testing.T) {
	if flowBuilderMaxBodyBytes != 512<<10 {
		t.Errorf("Wrong match. cap: got %d", flowBuilderMaxBodyBytes)
	}

	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc) // strict: never reaches the service

	t.Run("json over the cap", func(t *testing.T) {
		big := []byte(`{"messages":[{"role":"user","content":"` + strings.Repeat("a", flowBuilderMaxBodyBytes+10) + `"}],"supported_action_types":["talk"]}`)
		w := serveBuilder(t, mockSvc, builderServerAgent(), "POST", "/flow_builder/chat", big)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("Wrong match. got %d", w.Code)
		}
		if got := reasonFromBody(t, w); got != builder.ReasonInputTooLarge {
			t.Errorf("Wrong match. reason: got %q, want %s", got, builder.ReasonInputTooLarge)
		}
	})

	t.Run("a body between the two caps is read", func(t *testing.T) {
		// Over the Assistant Builder's cap, under this one.
		mc := gomock.NewController(t)
		mockSvc := servicehandler.NewMockServiceHandler(mc)
		a := builderServerAgent()
		mockSvc.EXPECT().FlowBuilderChat(gomock.Any(), a, gomock.Any()).Return(&flowbuilder.ChatResponse{Message: "ok"}, nil)

		pad := strings.Repeat("a", builderMaxBodyBytes+1000)
		body := []byte(`{"messages":[{"role":"user","content":"hi"}],"supported_action_types":["talk"],"pad":"` + pad + `"}`)
		if len(body) >= flowBuilderMaxBodyBytes {
			t.Fatal("the test body is not under the cap")
		}
		w := serveBuilder(t, mockSvc, a, "POST", "/flow_builder/chat", body)
		if w.Code != http.StatusOK {
			t.Errorf("Wrong match. got %d %s", w.Code, w.Body.String())
		}
	})
}

func Test_PostFlowBuilderChat_errorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantReason string
	}{
		{"daily limit", cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonDailyLimit, "x"), http.StatusTooManyRequests, builder.ReasonDailyLimit},
		{"busy", cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonBusy, "x"), http.StatusTooManyRequests, builder.ReasonBusy},
		{"timeout", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "x"), http.StatusServiceUnavailable, builder.ReasonTimeout},
		{"response invalid", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "x"), http.StatusServiceUnavailable, builder.ReasonResponseInvalid},
		{"unavailable", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "x"), http.StatusServiceUnavailable, builder.ReasonUnavailable},
		{"invalid argument", cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "x"), http.StatusBadRequest, builder.ReasonInvalidArgument},
		{"untyped", errors.New("boom"), http.StatusInternalServerError, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := builderServerAgent()
			mockSvc.EXPECT().FlowBuilderChat(gomock.Any(), a, gomock.Any()).Return(nil, tt.err)

			w := serveBuilder(t, mockSvc, a, "POST", "/flow_builder/chat", flowBuilderBody("hello"))
			if w.Code != tt.wantStatus {
				t.Errorf("Wrong match. status: got %d, want %d", w.Code, tt.wantStatus)
			}
			if got := reasonFromBody(t, w); got != tt.wantReason {
				t.Errorf("Wrong match. reason: got %q, want %q (%s)", got, tt.wantReason, w.Body.String())
			}
		})
	}
}

func Test_PostFlowBuilderChat_neverLogsTheInput(t *testing.T) {
	var buf bytes.Buffer
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	for name, svcErr := range map[string]error{
		"success":          nil,
		"echoing provider": errors.New("400: " + builderServerSecret),
		"typed with cause": cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "x").Wrap(errors.New(builderServerSecret)),
	} {
		t.Run(name, func(t *testing.T) {
			buf.Reset()
			mc := gomock.NewController(t)
			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := builderServerAgent()
			if svcErr == nil {
				mockSvc.EXPECT().FlowBuilderChat(gomock.Any(), a, gomock.Any()).Return(&flowbuilder.ChatResponse{Message: "ok"}, nil)
			} else {
				mockSvc.EXPECT().FlowBuilderChat(gomock.Any(), a, gomock.Any()).Return(nil, svcErr)
			}
			w := serveBuilder(t, mockSvc, a, "POST", "/flow_builder/chat", flowBuilderBody(builderServerSecret))
			if strings.Contains(buf.String(), builderServerSecret) {
				t.Errorf("Wrong match. a log line carries the input:\n%s", buf.String())
			}
			if strings.Contains(w.Body.String(), builderServerSecret) {
				t.Errorf("Wrong match. the response carries the input: %s", w.Body.String())
			}
		})
	}
}
