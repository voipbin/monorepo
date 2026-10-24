package httphandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// newTestRouter builds a router with the same route patterns as Run().
// The routes are registered here again, so a route change in Run() is not
// detected by these tests; they cover the handler behavior only.
func newTestRouter(h *httpHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/:id/ws", h.wsHandle)
	router.POST("/:id/tools", h.toolHandle)
	router.POST("/:id/member-switched", h.memberSwitchedHandle)

	return router
}

func Test_NewHTTPHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	res := NewHTTPHandler(nil, mockPipecatcall)
	h, ok := res.(*httpHandler)
	if !ok {
		t.Fatalf("Wrong type. expect: *httpHandler, got: %T", res)
	}
	if h.pipecatcallHandler != mockPipecatcall {
		t.Errorf("Wrong pipecatcallHandler. expect: the given handler, got: %v", h.pipecatcallHandler)
	}
}

func Test_handlers(t *testing.T) {
	validID := uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c")

	tests := []struct {
		name   string
		method string
		suffix string

		// expectCall registers the expected call on the mock for the given id and error.
		expectCall func(m *pipecatcallhandler.MockPipecatcallHandler, id uuid.UUID, ret error)
	}{
		{
			name:   "ws",
			method: http.MethodGet,
			suffix: "/ws",
			expectCall: func(m *pipecatcallhandler.MockPipecatcallHandler, id uuid.UUID, ret error) {
				m.EXPECT().RunnerWebsocketHandle(id, gomock.AssignableToTypeOf(&gin.Context{})).Return(ret)
			},
		},
		{
			name:   "tools",
			method: http.MethodPost,
			suffix: "/tools",
			expectCall: func(m *pipecatcallhandler.MockPipecatcallHandler, id uuid.UUID, ret error) {
				m.EXPECT().RunnerToolHandle(id, gomock.AssignableToTypeOf(&gin.Context{})).Return(ret)
			},
		},
		{
			name:   "member-switched",
			method: http.MethodPost,
			suffix: "/member-switched",
			expectCall: func(m *pipecatcallhandler.MockPipecatcallHandler, id uuid.UUID, ret error) {
				m.EXPECT().RunnerMemberSwitchedHandle(id, gomock.AssignableToTypeOf(&gin.Context{})).Return(ret)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" invalid id", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectation: the handler must not be called with an invalid id.
			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			router := newTestRouter(&httpHandler{pipecatcallHandler: mockPipecatcall})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/not-a-uuid"+tt.suffix, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong status. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
			if w.Body.Len() != 0 {
				t.Errorf("Wrong body. expect: empty, got: %q", w.Body.String())
			}
		})

		t.Run(tt.name+" nil uuid", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			router := newTestRouter(&httpHandler{pipecatcallHandler: mockPipecatcall})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/00000000-0000-0000-0000-000000000000"+tt.suffix, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong status. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
		})

		t.Run(tt.name+" handler error", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			router := newTestRouter(&httpHandler{pipecatcallHandler: mockPipecatcall})

			tt.expectCall(mockPipecatcall, validID, errors.New("session not found"))

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/"+validID.String()+tt.suffix, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong status. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
			expectBody := `{"error":"session not found"}`
			if w.Body.String() != expectBody {
				t.Errorf("Wrong body. expect: %s, got: %s", expectBody, w.Body.String())
			}
		})

		t.Run(tt.name+" success", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			router := newTestRouter(&httpHandler{pipecatcallHandler: mockPipecatcall})

			tt.expectCall(mockPipecatcall, validID, nil)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/"+validID.String()+tt.suffix, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Wrong status. expect: %d, got: %d", http.StatusOK, w.Code)
			}
			if w.Body.Len() != 0 {
				t.Errorf("Wrong body. expect: empty (the runner handler writes the response), got: %q", w.Body.String())
			}
		})
	}
}
