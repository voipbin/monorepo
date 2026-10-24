package listenhandler

import (
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

func Test_Run(t *testing.T) {
	queue := "test-listen-queue"

	t.Run("queue create fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		// ConsumeRPC has no expectation: it must not be started when the queue is missing.
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errors.New("queue failed"))

		h := NewListenHandler(mockSock, queue, "", nil)
		if err := h.Run(); err == nil {
			t.Errorf("Wrong match. expect: error, got: ok")
		}
	})

	t.Run("consume starts and its error is logged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		hook := logrustest.NewGlobal()
		defer hook.Reset()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, constCosumerName, false, false, false, 10, gomock.Any()).Return(errors.New("consume failed"))

		h := NewListenHandler(mockSock, queue, "", nil)
		if err := h.Run(); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			for _, e := range hook.AllEntries() {
				if e.Level == logrus.ErrorLevel && e.Message == "Could not consume the request message correctly. err: consume failed" {
					return
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Errorf("Wrong match. expect: the consume error logged, got: none")
	})
}

func Test_processV1TransfersPost_serviceStartFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockTransfer := transferhandler.NewMockTransferHandler(mc)
	h := &listenHandler{
		sockHandler:     sockhandler.NewMockSockHandler(mc),
		transferHandler: mockTransfer,
	}

	mockTransfer.EXPECT().ServiceStart(gomock.Any(), transfer.TypeAttended, uuid.FromStringOrNil("5b1d7a20-0001-11ee-9c01-000000000001"), gomock.Any()).Return(nil, errors.New("start failed"))

	res, err := h.processRequest(&sock.Request{
		URI:      "/v1/transfers",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"type":"attended","transferer_call_id":"5b1d7a20-0001-11ee-9c01-000000000001","transferee_addresses":[{"type":"tel","target":"+15550100"}]}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("Wrong status code. expect: 500, got: %d", res.StatusCode)
	}
	if len(res.Data) != 0 {
		t.Errorf("Wrong match. expect: empty body, got: %s", res.Data)
	}
}
