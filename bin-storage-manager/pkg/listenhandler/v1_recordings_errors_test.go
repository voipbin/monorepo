package listenhandler

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

func Test_v1RecordingsIDDelete_failureIsReturned(t *testing.T) {
	errDelete := errors.New("simulated delete failure")
	id := uuid.FromStringOrNil("f63a4c32-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockStorage, _ := newTestListenHandler(mc)
	mockStorage.EXPECT().RecordingDelete(gomock.Any(), id).Return(errDelete)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/recordings/" + id.String(),
		Method: sock.RequestMethodDelete,
	})
	if !errors.Is(err, errDelete) || res != nil {
		t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errDelete, res, err)
	}
}
