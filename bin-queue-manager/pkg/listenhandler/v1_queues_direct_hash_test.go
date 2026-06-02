package listenhandler

import (
	"net/http"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

func Test_processV1QueuesIDDirectHashRegeneratePost_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		sockHandler:  mockSock,
		queueHandler: mockQueue,
	}

	queueID := uuid.FromStringOrNil("aa000000-0000-0000-0000-000000000001")

	mockQueue.EXPECT().DirectHashRegenerate(gomock.Any(), queueID).Return(nil, dbhandler.ErrNotFound)

	req := &sock.Request{
		URI:    "/v1/queues/aa000000-0000-0000-0000-000000000001/direct-hash-regenerate",
		Method: sock.RequestMethodPost,
	}

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusNotFound, res.StatusCode)
	}
}
