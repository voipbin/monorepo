package pipecatcallhandler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	ammessage "monorepo/bin-ai-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// newHandleTestContext builds a gin context whose request carries the given url and body.
func newHandleTestContext(url string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	return c, w
}

// registerHandleTestSession puts a minimal session into the handler's session map.
func registerHandleTestSession(t *testing.T, h *pipecatcallHandler, id uuid.UUID) *pipecatcall.Session {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{ID: id},
		Ctx:      ctx,
		Cancel:   cancel,
	}
	h.mapPipecatcallSession[id] = se
	return se
}

func Test_RunnerWebsocketHandle_errorPaths(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-0001-4000-8000-000000000001")

	tests := []struct {
		name string

		url       string
		expectErr string
	}{
		{name: "invalid direction", url: "/?direction=sideways", expectErr: "invalid direction parameter: sideways"},
		{name: "missing direction", url: "/", expectErr: "invalid direction parameter: "},
		{name: "input without session", url: "/?direction=input", expectErr: "could not get pipecatcall session"},
		{name: "output without session", url: "/?direction=output", expectErr: "could not get pipecatcall session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &pipecatcallHandler{
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
			}
			c, _ := newHandleTestContext(tt.url, "")

			err := h.RunnerWebsocketHandle(pcID, c)
			if err == nil {
				t.Fatalf("expected an error")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("wrong error. expect contains: %q, got: %q", tt.expectErr, err.Error())
			}
		})
	}
}

func Test_RunnerToolHandle_errorPaths(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-0002-4000-8000-000000000001")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-0002-4000-8000-000000000002")
	validBody := `{"id":"tool-1","type":"function","function":{"name":"get_variables","arguments":"{}"}}`

	tests := []struct {
		name string

		withSession bool
		getResult   *pipecatcall.Pipecatcall
		getErr      error
		body        string
		rpcErr      error

		expectErr  string
		expectCode int
	}{
		{name: "no session", expectErr: "could not get pipecatcall session", expectCode: http.StatusOK},
		{name: "get fails", withSession: true, getErr: fmt.Errorf("db down"), body: validBody, expectErr: "could not get pipecatcall: ", expectCode: http.StatusOK},
		{
			name: "reference type is not ai call", withSession: true, body: validBody,
			getResult: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{ID: pcID}, ReferenceType: pipecatcall.ReferenceTypeCall, ReferenceID: referenceID,
			},
			expectErr: "pipecatcall reference type is not ai-call", expectCode: http.StatusOK,
		},
		{
			name: "invalid json body", withSession: true, body: "{not json",
			getResult: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{ID: pcID}, ReferenceType: pipecatcall.ReferenceTypeAICall, ReferenceID: referenceID,
			},
			expectErr: "could not bind tool request JSON", expectCode: http.StatusBadRequest,
		},
		{
			name: "rpc fails", withSession: true, body: validBody, rpcErr: fmt.Errorf("ai-manager unavailable"),
			getResult: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{ID: pcID}, ReferenceType: pipecatcall.ReferenceTypeAICall, ReferenceID: referenceID,
			},
			expectErr: "could not execute tool via ai-manager: ai-manager unavailable", expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &pipecatcallHandler{
				db:                    mockDB,
				requestHandler:        mockReq,
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
			}

			if tt.withSession {
				registerHandleTestSession(t, h, pcID)
				mockDB.EXPECT().PipecatcallGet(gomock.Any(), pcID).Return(tt.getResult, tt.getErr)
			}
			if tt.rpcErr != nil {
				mockReq.EXPECT().AIV1AIcallToolExecute(gomock.Any(), referenceID, "tool-1", ammessage.ToolType("function"), gomock.Any(), pcID).Return(nil, tt.rpcErr)
			}

			c, w := newHandleTestContext("/", tt.body)
			err := h.RunnerToolHandle(pcID, c)
			if err == nil {
				t.Fatalf("expected an error")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("wrong error. expect contains: %q, got: %q", tt.expectErr, err.Error())
			}
			if w.Code != tt.expectCode {
				t.Errorf("wrong status code. expect: %d, got: %d", tt.expectCode, w.Code)
			}
		})
	}
}

func Test_RunnerToolHandle_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &pipecatcallHandler{
		db:                    mockDB,
		requestHandler:        mockReq,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-0003-4000-8000-000000000001")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-0003-4000-8000-000000000002")
	registerHandleTestSession(t, h, pcID)

	mockDB.EXPECT().PipecatcallGet(gomock.Any(), pcID).Return(&pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: pcID},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   referenceID,
	}, nil)

	wantFunction := &ammessage.FunctionCall{Name: "get_variables", Arguments: `{"k":"v"}`}
	mockReq.EXPECT().AIV1AIcallToolExecute(gomock.Any(), referenceID, "tool-1", ammessage.ToolType("function"), gomock.Any(), pcID).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ string, _ ammessage.ToolType, f *ammessage.FunctionCall, _ uuid.UUID) (map[string]any, error) {
			if !reflect.DeepEqual(f, wantFunction) {
				t.Errorf("wrong function. expect: %v, got: %v", wantFunction, f)
			}
			return map[string]any{"result": "ok"}, nil
		})

	body := `{"id":"tool-1","type":"function","function":{"name":"get_variables","arguments":"{\"k\":\"v\"}"}}`
	c, w := newHandleTestContext("/", body)
	if err := h.RunnerToolHandle(pcID, c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("wrong status code. expect: %d, got: %d", http.StatusOK, w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != `{"result":"ok"}` {
		t.Errorf("wrong body. got: %s", got)
	}
}

func Test_RunnerMemberSwitchedHandle_errorPaths(t *testing.T) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-0004-4000-8000-000000000001")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-0004-4000-8000-000000000002")
	validBody := `{"transition_function_name":"go_next","from_member":{"id":"c3d4e5f6-0004-4000-8000-000000000003"},"to_member":{"id":"c3d4e5f6-0004-4000-8000-000000000004"}}`

	tests := []struct {
		name string

		withSession bool
		getResult   *pipecatcall.Pipecatcall
		getErr      error
		body        string

		expectErr  string
		expectCode int
	}{
		{name: "no session", expectErr: "could not get pipecatcall session", expectCode: http.StatusOK},
		{name: "get fails", withSession: true, getErr: fmt.Errorf("db down"), body: validBody, expectErr: "could not get pipecatcall: ", expectCode: http.StatusOK},
		{
			name: "reference type is not ai call", withSession: true, body: validBody,
			getResult: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{ID: pcID}, ReferenceType: pipecatcall.ReferenceTypeCall, ReferenceID: referenceID,
			},
			expectErr: "pipecatcall reference type is not ai-call", expectCode: http.StatusOK,
		},
		{
			name: "invalid json body", withSession: true, body: "{not json",
			getResult: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{ID: pcID}, ReferenceType: pipecatcall.ReferenceTypeAICall, ReferenceID: referenceID,
			},
			expectErr: "could not bind member-switched request JSON", expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			// no PublishEvent expectation: any publish fails the test.
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &pipecatcallHandler{
				db:                    mockDB,
				notifyHandler:         mockNotify,
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
			}

			if tt.withSession {
				registerHandleTestSession(t, h, pcID)
				mockDB.EXPECT().PipecatcallGet(gomock.Any(), pcID).Return(tt.getResult, tt.getErr)
			}

			c, w := newHandleTestContext("/", tt.body)
			err := h.RunnerMemberSwitchedHandle(pcID, c)
			if err == nil {
				t.Fatalf("expected an error")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("wrong error. expect contains: %q, got: %q", tt.expectErr, err.Error())
			}
			if w.Code != tt.expectCode {
				t.Errorf("wrong status code. expect: %d, got: %d", tt.expectCode, w.Code)
			}
		})
	}
}

func Test_RunnerMemberSwitchedHandle_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &pipecatcallHandler{
		db:                    mockDB,
		notifyHandler:         mockNotify,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-0005-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("d4e5f6a7-0005-4000-8000-000000000002")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-0005-4000-8000-000000000003")
	activeflowID := uuid.FromStringOrNil("e5f6a7b8-0005-4000-8000-000000000004")
	fromID := uuid.FromStringOrNil("c3d4e5f6-0005-4000-8000-000000000005")
	toID := uuid.FromStringOrNil("c3d4e5f6-0005-4000-8000-000000000006")
	registerHandleTestSession(t, h, pcID)

	mockDB.EXPECT().PipecatcallGet(gomock.Any(), pcID).Return(&pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: pcID, CustomerID: customerID},
		ActiveflowID:  activeflowID,
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   referenceID,
	}, nil)

	want := &message.MemberSwitchedEvent{
		CustomerID:               customerID,
		PipecatcallID:            pcID,
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   referenceID,
		ActiveflowID:             activeflowID,
		TransitionFunctionName:   "go_next",
		FromMember:               message.MemberInfo{ID: fromID, Name: "front"},
		ToMember:                 message.MemberInfo{ID: toID, Name: "back"},
	}

	mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypeTeamMemberSwitched, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, data any) {
			got, ok := data.(*message.MemberSwitchedEvent)
			if !ok {
				t.Errorf("event must be published as a pointer, got %T", data)
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("wrong event. expect: %v, got: %v", want, got)
			}
		}).Times(1)

	body := fmt.Sprintf(
		`{"transition_function_name":"go_next","from_member":{"id":"%s","name":"front"},"to_member":{"id":"%s","name":"back"}}`,
		fromID, toID,
	)
	c, w := newHandleTestContext("/", body)
	if err := h.RunnerMemberSwitchedHandle(pcID, c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("wrong status code. expect: %d, got: %d", http.StatusOK, w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != `{"status":"ok"}` {
		t.Errorf("wrong body. got: %s", got)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("wrong content type. got: %s", ct)
	}
}
