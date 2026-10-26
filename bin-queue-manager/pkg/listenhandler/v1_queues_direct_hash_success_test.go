package listenhandler

import (
	"encoding/json"
	"net/http"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

func Test_processV1QueuesIDDirectHashRegeneratePost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &listenHandler{
		sockHandler:  mockSock,
		queueHandler: mockQueue,
	}

	queueID := uuid.FromStringOrNil("aa000000-0000-0000-0000-000000000002")
	responseQueue := &queue.Queue{Name: "regenerated"}
	responseQueue.ID = queueID

	mockQueue.EXPECT().DirectHashRegenerate(gomock.Any(), queueID).Return(responseQueue, nil)

	req := &sock.Request{
		URI:    "/v1/queues/aa000000-0000-0000-0000-000000000002/direct-hash-regenerate",
		Method: sock.RequestMethodPost,
	}

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusOK, res.StatusCode)
	}
	if res.DataType != "application/json" {
		t.Errorf("Wrong data type. expect: application/json, got: %s", res.DataType)
	}

	var got queue.Queue
	if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil {
		t.Fatalf("Wrong match. the response is not a queue. err: %v", errUnmarshal)
	}
	if got.ID != queueID || got.Name != "regenerated" {
		t.Errorf("Wrong match. expect: %s/regenerated, got: %s/%s", queueID, got.ID, got.Name)
	}
}
