package listenhandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

const (
	tQueueID     = "6c1f3a10-7e11-11ee-9c01-000000000001"
	tQueuecallID = "6c1f3a10-7e11-11ee-9c01-000000000002"
	tReferenceID = "6c1f3a10-7e11-11ee-9c01-000000000003"
	tAgentID     = "6c1f3a10-7e11-11ee-9c01-000000000004"
)

type lhMocks struct {
	sock      *sockhandler.MockSockHandler
	queue     *queuehandler.MockQueueHandler
	queuecall *queuecallhandler.MockQueuecallHandler
}

func newLH(mc *gomock.Controller) (*listenHandler, *lhMocks) {
	m := &lhMocks{
		sock:      sockhandler.NewMockSockHandler(mc),
		queue:     queuehandler.NewMockQueueHandler(mc),
		queuecall: queuecallhandler.NewMockQueuecallHandler(mc),
	}
	return &listenHandler{
		sockHandler:      m.sock,
		queueHandler:     m.queue,
		queuecallHandler: m.queuecall,
	}, m
}

// Test_processRequest_handlerErrors verifies, for every endpoint that delegates
// to a business handler, that a handler failure becomes the matching status code
// and that nothing but the single expected handler call is made.
func Test_processRequest_handlerErrors(t *testing.T) {
	qid := uuid.FromStringOrNil(tQueueID)
	qcid := uuid.FromStringOrNil(tQueuecallID)
	rid := uuid.FromStringOrNil(tReferenceID)

	typedNotFound := cerrors.NotFound(commonoutline.ServiceNameQueueManager, "X_NOT_FOUND", "not found")

	// each case wires the one expected business call to fail with err.
	endpoints := []struct {
		name    string
		request *sock.Request
		fail    func(m *lhMocks, err error)
	}{
		{
			name:    "queues count_by_customer",
			request: &sock.Request{URI: "/v1/queues/count_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"` + tQueueID + `"}`)},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().CountByCustomerID(gomock.Any(), qid).Return(0, err)
			},
		},
		{
			name:    "queues list",
			request: &sock.Request{URI: "/v1/queues?page_size=5&page_token=tok", Method: sock.RequestMethodGet, Data: []byte(`{"deleted":false}`)},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().List(gomock.Any(), uint64(5), "tok", map[queue.Field]any{queue.FieldDeleted: false}).Return(nil, err)
			},
		},
		{
			name:    "queues create",
			request: &sock.Request{URI: "/v1/queues", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":"` + tQueueID + `","name":"n","routing_method":"random","wait_timeout":10,"service_timeout":20}`)},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().Create(gomock.Any(), qid, "n", "", queue.RoutingMethodRandom, nil, uuid.Nil, 10, 20).Return(nil, err)
			},
		},
		{
			name:    "queues get",
			request: &sock.Request{URI: "/v1/queues/" + tQueueID, Method: sock.RequestMethodGet},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().Get(gomock.Any(), qid).Return(nil, err)
			},
		},
		{
			name:    "queues delete",
			request: &sock.Request{URI: "/v1/queues/" + tQueueID, Method: sock.RequestMethodDelete},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().Delete(gomock.Any(), qid).Return(nil, err)
			},
		},
		{
			name:    "queues put",
			request: &sock.Request{URI: "/v1/queues/" + tQueueID, Method: sock.RequestMethodPut, Data: []byte(`{"name":"renamed"}`)},
			fail: func(m *lhMocks, err error) {
				name := "renamed"
				m.queue.EXPECT().UpdateBasicInfo(gomock.Any(), qid, &name, nil, nil, nil, nil, nil, nil).Return(nil, err)
			},
		},
		{
			name:    "queues tag_ids put",
			request: &sock.Request{URI: "/v1/queues/" + tQueueID + "/tag_ids", Method: sock.RequestMethodPut, Data: []byte(`{"tag_ids":["` + tAgentID + `"]}`)},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().UpdateTagIDs(gomock.Any(), qid, []uuid.UUID{uuid.FromStringOrNil(tAgentID)}).Return(nil, err)
			},
		},
		{
			name:    "queues routing_method put",
			request: &sock.Request{URI: "/v1/queues/" + tQueueID + "/routing_method", Method: sock.RequestMethodPut, Data: []byte(`{"routing_method":"random"}`)},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().UpdateRoutingMethod(gomock.Any(), qid, queue.RoutingMethodRandom).Return(nil, err)
			},
		},
		{
			name:    "queues agents get",
			request: &sock.Request{URI: "/v1/queues/" + tQueueID + "/agents", Method: sock.RequestMethodGet, Data: []byte(`{"status":"available"}`)},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().GetAgents(gomock.Any(), qid, gomock.Any()).Return(nil, err)
			},
		},
		{
			name:    "queues direct-hash-regenerate",
			request: &sock.Request{URI: "/v1/queues/" + tQueueID + "/direct-hash-regenerate", Method: sock.RequestMethodPost},
			fail: func(m *lhMocks, err error) {
				m.queue.EXPECT().DirectHashRegenerate(gomock.Any(), qid).Return(nil, err)
			},
		},
		{
			name:    "queuecalls list",
			request: &sock.Request{URI: "/v1/queuecalls?page_size=7&page_token=t2", Method: sock.RequestMethodGet, Data: []byte(`{"deleted":false}`)},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().List(gomock.Any(), uint64(7), "t2", map[queuecall.Field]any{queuecall.FieldDeleted: false}).Return(nil, err)
			},
		},
		{
			name:    "queuecalls get",
			request: &sock.Request{URI: "/v1/queuecalls/" + tQueuecallID, Method: sock.RequestMethodGet},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().Get(gomock.Any(), qcid).Return(nil, err)
			},
		},
		{
			name:    "queuecalls delete",
			request: &sock.Request{URI: "/v1/queuecalls/" + tQueuecallID, Method: sock.RequestMethodDelete},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().Delete(gomock.Any(), qcid).Return(nil, err)
			},
		},
		{
			name:    "queuecalls get by reference id",
			request: &sock.Request{URI: "/v1/queuecalls/reference_id/" + tReferenceID, Method: sock.RequestMethodGet},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().GetByReferenceID(gomock.Any(), rid).Return(nil, err)
			},
		},
		{
			name:    "queuecalls kick",
			request: &sock.Request{URI: "/v1/queuecalls/" + tQueuecallID + "/kick", Method: sock.RequestMethodPost},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().Kick(gomock.Any(), qcid).Return(nil, err)
			},
		},
		{
			name:    "queuecalls kick by reference id",
			request: &sock.Request{URI: "/v1/queuecalls/reference_id/" + tReferenceID + "/kick", Method: sock.RequestMethodPost},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().KickByReferenceID(gomock.Any(), rid).Return(nil, err)
			},
		},
		{
			name:    "queuecalls status_waiting",
			request: &sock.Request{URI: "/v1/queuecalls/" + tQueuecallID + "/status_waiting", Method: sock.RequestMethodPost},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().UpdateStatusWaiting(gomock.Any(), qcid).Return(nil, err)
			},
		},
		{
			name:    "queuecalls execute",
			request: &sock.Request{URI: "/v1/queuecalls/" + tQueuecallID + "/execute", Method: sock.RequestMethodPost, Data: []byte(`{"agent_id":"` + tAgentID + `"}`)},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().Execute(gomock.Any(), qcid, uuid.FromStringOrNil(tAgentID)).Return(nil, err)
			},
		},
		{
			name:    "services type queuecall",
			request: &sock.Request{URI: "/v1/services/type/queuecall", Method: sock.RequestMethodPost, Data: []byte(`{"queue_id":"` + tQueueID + `","reference_type":"call","reference_id":"` + tReferenceID + `"}`)},
			fail: func(m *lhMocks, err error) {
				m.queuecall.EXPECT().ServiceStart(gomock.Any(), qid, uuid.Nil, queuecall.ReferenceType("call"), rid).Return(nil, err)
			},
		},
	}

	errCases := []struct {
		name         string
		err          error
		expectStatus int
	}{
		{"untyped error is an internal error", fmt.Errorf("boom"), http.StatusInternalServerError},
		{"record-not-found is mapped to not found", dbhandler.ErrNotFound, http.StatusNotFound},
		{"typed not found keeps its status", typedNotFound, http.StatusNotFound},
	}

	for _, ep := range endpoints {
		for _, e := range errCases {
			t.Run(ep.name+": "+e.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				h, m := newLH(mc)
				ep.fail(m, e.err)

				res, err := h.processRequest(ep.request)
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if res == nil || res.StatusCode != e.expectStatus {
					t.Errorf("Wrong match. expect: %d, got: %v", e.expectStatus, res)
				}
				if res != nil && len(res.Data) != 0 && res.DataType != cerrors.DataTypeVoipbinError {
					t.Errorf("Wrong match. an error response must not carry a success body. got: %v", res)
				}
			})
		}
	}
}

// Test_processRequest_badBody verifies that a malformed body is rejected with
// 400 before any business handler is touched (the mocks have no expectations).
func Test_processRequest_badBody(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{"queues count_by_customer", &sock.Request{URI: "/v1/queues/count_by_customer", Method: sock.RequestMethodGet, Data: []byte(`not json`)}},
		{"queues list filters", &sock.Request{URI: "/v1/queues?page_size=5", Method: sock.RequestMethodGet, Data: []byte(`not json`)}},
		{"queues list filter with an invalid uuid", &sock.Request{URI: "/v1/queues?page_size=5", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)}},
		{"queues create", &sock.Request{URI: "/v1/queues", Method: sock.RequestMethodPost, Data: []byte(`not json`)}},
		{"queues put", &sock.Request{URI: "/v1/queues/" + tQueueID, Method: sock.RequestMethodPut, Data: []byte(`not json`)}},
		{"queues tag_ids put", &sock.Request{URI: "/v1/queues/" + tQueueID + "/tag_ids", Method: sock.RequestMethodPut, Data: []byte(`not json`)}},
		{"queues routing_method put", &sock.Request{URI: "/v1/queues/" + tQueueID + "/routing_method", Method: sock.RequestMethodPut, Data: []byte(`not json`)}},
		{"queues agents get", &sock.Request{URI: "/v1/queues/" + tQueueID + "/agents", Method: sock.RequestMethodGet, Data: []byte(`not json`)}},
		{"queuecalls list filters", &sock.Request{URI: "/v1/queuecalls?page_size=5", Method: sock.RequestMethodGet, Data: []byte(`not json`)}},
		{"queuecalls list filter with an invalid uuid", &sock.Request{URI: "/v1/queuecalls?page_size=5", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)}},
		{"queuecalls execute", &sock.Request{URI: "/v1/queuecalls/" + tQueuecallID + "/execute", Method: sock.RequestMethodPost, Data: []byte(`not json`)}},
		{"queuecalls health-check", &sock.Request{URI: "/v1/queuecalls/" + tQueuecallID + "/health-check", Method: sock.RequestMethodPost, Data: []byte(`not json`)}},
		{"services type queuecall is a plain error, mapped to 400 by the router", &sock.Request{URI: "/v1/services/type/queuecall", Method: sock.RequestMethodPost, Data: []byte(`not json`)}},
		{"unparsable url is a plain error, mapped to 400 by the router", &sock.Request{URI: "/v1/queues?page_size=\x7f", Method: sock.RequestMethodGet}},
		{"queuecalls unparsable url", &sock.Request{URI: "/v1/queuecalls?page_size=\x7f", Method: sock.RequestMethodGet}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newLH(mc)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %v", res)
			}
		})
	}
}

func Test_processRequest_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{"unknown path", &sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet}},
		{"known path with an unsupported method", &sock.Request{URI: "/v1/queues/" + tQueueID, Method: sock.RequestMethodPost}},
		{"queuecall id with a non-uuid", &sock.Request{URI: "/v1/queuecalls/not-a-uuid", Method: sock.RequestMethodGet}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newLH(mc)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusNotFound {
				t.Errorf("Wrong match. expect: 404, got: %v", res)
			}
		})
	}
}

// Test_shortURIs calls the handlers directly with URIs that have too few path
// items (the router never produces them), and with a non-uuid id.
// The mocks have no expectations: a short or malformed URI must never reach
// the business layer.
func Test_shortURIs(t *testing.T) {
	type handlerFunc func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)

	validBody := []byte(`{"agent_id":"` + tAgentID + `","tag_ids":[],"routing_method":"random","retry_count":1}`)

	// itemsThree: fewer than four items. itemsFour: exactly four items.
	const itemsThree = "/v1/queues"
	const itemsFour = "/v1/queues/" + tQueueID

	tests := []struct {
		name   string
		call   handlerFunc
		uri    string
		body   []byte
		expect int
	}{
		{"queues get", (*listenHandler).processV1QueuesIDGet, itemsThree, nil, 400},
		{"queues get non-uuid", (*listenHandler).processV1QueuesIDGet, "/v1/queues/not-a-uuid", nil, 400},
		{"queues delete", (*listenHandler).processV1QueuesIDDelete, itemsThree, nil, 400},
		{"queues delete non-uuid", (*listenHandler).processV1QueuesIDDelete, "/v1/queues/not-a-uuid", nil, 400},
		{"queues put", (*listenHandler).processV1QueuesIDPut, itemsThree, validBody, 400},
		{"queues tag_ids put", (*listenHandler).processV1QueuesIDTagIDsPut, itemsFour, validBody, 400},
		{"queues routing_method put", (*listenHandler).processV1QueuesIDRoutingMethodPut, itemsFour, validBody, 400},
		{"queues agents get", (*listenHandler).processV1QueuesIDAgentsGet, itemsFour, nil, 400},
		{"queues direct-hash-regenerate", (*listenHandler).processV1QueuesIDDirectHashRegeneratePost, itemsThree, nil, 400},
		{"queuecalls get", (*listenHandler).processV1QueuecallsIDGet, "/v1/queuecalls", nil, 400},
		{"queuecalls get non-uuid", (*listenHandler).processV1QueuecallsIDGet, "/v1/queuecalls/not-a-uuid", nil, 400},
		{"queuecalls delete", (*listenHandler).processV1QueuecallsIDDelete, "/v1/queuecalls", nil, 400},
		{"queuecalls delete non-uuid", (*listenHandler).processV1QueuecallsIDDelete, "/v1/queuecalls/not-a-uuid", nil, 400},
		{"queuecalls get by reference id", (*listenHandler).processV1QueuecallsReferenceIDIDGet, "/v1/queuecalls", nil, 400},
		{"queuecalls kick", (*listenHandler).processV1QueuecallsIDKickPost, "/v1/queuecalls/" + tQueuecallID, nil, 400},
		{"queuecalls kick by reference id", (*listenHandler).processV1QueuecallsReferenceIDIDKickPost, "/v1/queuecalls/reference_id", nil, 400},
		{"queuecalls status_waiting", (*listenHandler).processV1QueuecallsIDStatusWaitingPost, "/v1/queuecalls/" + tQueuecallID, nil, 400},
		{"queuecalls timeout_wait", (*listenHandler).processV1QueuecallsIDTimeoutWaitPost, "/v1/queuecalls/" + tQueuecallID, nil, 400},
		{"queuecalls timeout_service", (*listenHandler).processV1QueuecallsIDTimeoutServicePost, "/v1/queuecalls/" + tQueuecallID, nil, 400},
		{"queuecalls execute", (*listenHandler).processV1QueuecallsIDExecutePost, "/v1/queuecalls/" + tQueuecallID, validBody, 400},
		{"queuecalls health-check", (*listenHandler).processV1QueuecallsIDHealthCheckPost, "/v1/queuecalls/" + tQueuecallID, validBody, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newLH(mc)

			res, err := tt.call(h, context.Background(), &sock.Request{URI: tt.uri, Data: tt.body})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expect, res)
			}
		})
	}
}

func Test_processV1QueuecallsTimeouts(t *testing.T) {
	qcid := uuid.FromStringOrNil(tQueuecallID)

	tests := []struct {
		name   string
		uri    string
		expect func(m *lhMocks)
	}{
		{
			name: "timeout_wait is forwarded with the queuecall id",
			uri:  "/v1/queuecalls/" + tQueuecallID + "/timeout_wait",
			expect: func(m *lhMocks) {
				m.queuecall.EXPECT().TimeoutWait(gomock.Any(), qcid)
			},
		},
		{
			name: "timeout_service is forwarded with the queuecall id",
			uri:  "/v1/queuecalls/" + tQueuecallID + "/timeout_service",
			expect: func(m *lhMocks) {
				m.queuecall.EXPECT().TimeoutService(gomock.Any(), qcid)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newLH(mc)
			tt.expect(m)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodPost})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusOK || res.DataType != "application/json" || len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: 200 without a body, got: %v", res)
			}
		})
	}
}

func Test_processV1QueuecallsIDExecutePost_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newLH(mc)

	qcid := uuid.FromStringOrNil(tQueuecallID)
	agentID := uuid.FromStringOrNil(tAgentID)
	executed := &queuecall.Queuecall{
		Identity:       commonidentity.Identity{ID: qcid},
		ServiceAgentID: agentID,
	}

	m.queuecall.EXPECT().Execute(gomock.Any(), qcid, agentID).Return(executed, nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/queuecalls/" + tQueuecallID + "/execute",
		Method: sock.RequestMethodPost,
		Data:   []byte(`{"agent_id":"` + tAgentID + `"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusOK || res.DataType != "application/json" {
		t.Fatalf("Wrong match. expect: 200 json, got: %v", res)
	}
	if !strings.Contains(string(res.Data), `"id":"`+tQueuecallID+`"`) || !strings.Contains(string(res.Data), `"service_agent_id":"`+tAgentID+`"`) {
		t.Errorf("Wrong match. the executed queuecall must be returned. got: %s", res.Data)
	}
}

func Test_Run(t *testing.T) {
	t.Run("queue declaration failure is returned and nothing is consumed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newLH(mc)
		errQueue := errors.New("queue create failed")

		m.sock.EXPECT().QueueCreate("test-queue", "normal").Return(errQueue)

		if err := h.Run("test-queue", "delay"); !errors.Is(err, errQueue) {
			t.Errorf("Wrong match. expect: %v, got: %v", errQueue, err)
		}
	})

	t.Run("successful setup starts consuming rpc requests on the declared queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newLH(mc)

		consumed := make(chan string, 1)
		m.sock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
		m.sock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "queue-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, queueName string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				consumed <- queueName
				return errors.New("consumer stopped")
			})

		if err := h.Run("test-queue", "delay"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case got := <-consumed:
			if got != "test-queue" {
				t.Errorf("Wrong match. expect: test-queue, got: %s", got)
			}
		case <-time.After(time.Second):
			t.Fatalf("Wrong match. ConsumeRPC was not started")
		}
	})
}

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	_, m := newLH(mc)

	h, ok := NewListenHandler(m.sock, m.queue, m.queuecall).(*listenHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *listenHandler")
	}
	if h.sockHandler != m.sock || h.queueHandler != m.queue || h.queuecallHandler != m.queuecall {
		t.Errorf("Wrong match. the injected handlers must be kept as given")
	}
	if h.utilHanlder == nil {
		t.Errorf("Wrong match. the util handler must be initialised")
	}
}
