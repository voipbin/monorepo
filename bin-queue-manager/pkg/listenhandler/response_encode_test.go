package listenhandler

import (
	"net/http"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/models/sock"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

// Test_processRequest_responseCannotBeEncoded checks that a result the encoder
// rejects is answered with an internal error instead of a partial body. The
// timestamp of the returned resource is outside the range JSON can express.
func Test_processRequest_responseCannotBeEncoded(t *testing.T) {
	id := "6b1c2d3e-6001-11f1-8d00-000000000001"
	refID := "6b1c2d3e-6002-11f1-8d00-000000000002"

	outOfRange := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	badQueue := &queue.Queue{TMCreate: &outOfRange}
	badQueuecall := &queuecall.Queuecall{TMCreate: &outOfRange}

	tests := []struct {
		name string

		request *sock.Request

		setup func(mockQueue *queuehandler.MockQueueHandler, mockQueuecall *queuecallhandler.MockQueuecallHandler)
	}{
		{
			name:    "queue create",
			request: &sock.Request{URI: "/v1/queues", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{"name":"queue-a"}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().Create(gomock.Any(), gomock.Any(), "queue-a", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(badQueue, nil)
			},
		},
		{
			name:    "queue list",
			request: &sock.Request{URI: "/v1/queues?page_size=10", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().List(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return([]*queue.Queue{badQueue}, nil)
			},
		},
		{
			name:    "queue get",
			request: &sock.Request{URI: "/v1/queues/" + id, Method: sock.RequestMethodGet},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(id)).Return(badQueue, nil)
			},
		},
		{
			name:    "queue delete",
			request: &sock.Request{URI: "/v1/queues/" + id, Method: sock.RequestMethodDelete},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(id)).Return(badQueue, nil)
			},
		},
		{
			name:    "queue update",
			request: &sock.Request{URI: "/v1/queues/" + id, Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().UpdateBasicInfo(gomock.Any(), uuid.FromStringOrNil(id), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(badQueue, nil)
			},
		},
		{
			name:    "queue tag ids update",
			request: &sock.Request{URI: "/v1/queues/" + id + "/tag_ids", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{"tag_ids":[]}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().UpdateTagIDs(gomock.Any(), uuid.FromStringOrNil(id), gomock.Any()).Return(badQueue, nil)
			},
		},
		{
			name:    "queue routing method update",
			request: &sock.Request{URI: "/v1/queues/" + id + "/routing_method", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{"routing_method":"random"}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().UpdateRoutingMethod(gomock.Any(), uuid.FromStringOrNil(id), queue.RoutingMethodRandom).Return(badQueue, nil)
			},
		},
		{
			name:    "queue agents",
			request: &sock.Request{URI: "/v1/queues/" + id + "/agents", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().GetAgents(gomock.Any(), uuid.FromStringOrNil(id), gomock.Any()).Return([]amagent.Agent{{TMCreate: &outOfRange}}, nil)
			},
		},
		{
			name:    "queue direct hash regenerate",
			request: &sock.Request{URI: "/v1/queues/" + id + "/direct-hash-regenerate", Method: sock.RequestMethodPost},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				q.EXPECT().DirectHashRegenerate(gomock.Any(), uuid.FromStringOrNil(id)).Return(badQueue, nil)
			},
		},
		{
			name:    "queuecall list",
			request: &sock.Request{URI: "/v1/queuecalls?page_size=10", Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(`{}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().List(gomock.Any(), uint64(10), gomock.Any(), gomock.Any()).Return([]*queuecall.Queuecall{badQueuecall}, nil)
			},
		},
		{
			name:    "queuecall get by reference",
			request: &sock.Request{URI: "/v1/queuecalls/reference_id/" + refID, Method: sock.RequestMethodGet},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().GetByReferenceID(gomock.Any(), uuid.FromStringOrNil(refID)).Return(badQueuecall, nil)
			},
		},
		{
			name:    "queuecall get",
			request: &sock.Request{URI: "/v1/queuecalls/" + id, Method: sock.RequestMethodGet},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(id)).Return(badQueuecall, nil)
			},
		},
		{
			name:    "queuecall delete",
			request: &sock.Request{URI: "/v1/queuecalls/" + id, Method: sock.RequestMethodDelete},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(id)).Return(badQueuecall, nil)
			},
		},
		{
			name:    "queuecall execute",
			request: &sock.Request{URI: "/v1/queuecalls/" + id + "/execute", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{"agent_id":"` + refID + `"}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().Execute(gomock.Any(), uuid.FromStringOrNil(id), uuid.FromStringOrNil(refID)).Return(badQueuecall, nil)
			},
		},
		{
			name:    "queuecall kick",
			request: &sock.Request{URI: "/v1/queuecalls/" + id + "/kick", Method: sock.RequestMethodPost},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().Kick(gomock.Any(), uuid.FromStringOrNil(id)).Return(badQueuecall, nil)
			},
		},
		{
			name:    "queuecall kick by reference",
			request: &sock.Request{URI: "/v1/queuecalls/reference_id/" + refID + "/kick", Method: sock.RequestMethodPost},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().KickByReferenceID(gomock.Any(), uuid.FromStringOrNil(refID)).Return(badQueuecall, nil)
			},
		},
		{
			name:    "queuecall status waiting",
			request: &sock.Request{URI: "/v1/queuecalls/" + id + "/status_waiting", Method: sock.RequestMethodPost},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().UpdateStatusWaiting(gomock.Any(), uuid.FromStringOrNil(id)).Return(badQueuecall, nil)
			},
		},
		{
			name:    "service start",
			request: &sock.Request{URI: "/v1/services/type/queuecall", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{}`)},
			setup: func(q *queuehandler.MockQueueHandler, qc *queuecallhandler.MockQueuecallHandler) {
				qc.EXPECT().ServiceStart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&commonservice.Service{
					PushActions: []fmaction.Action{{Option: map[string]any{"unencodable": make(chan int)}}},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockQueue := queuehandler.NewMockQueueHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)
			h := &listenHandler{
				queueHandler:     mockQueue,
				queuecallHandler: mockQueuecall,
			}
			tt.setup(mockQueue, mockQueuecall)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusInternalServerError {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong match. no body must be sent on failure, got: %s", res.Data)
			}
		})
	}
}

func Test_errorResponse_typedErrorCannotBeConverted(t *testing.T) {
	// a typed error without a value cannot be converted into a response
	var ve *cerrors.VoipbinError

	res := errorResponse(ve)
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
}
