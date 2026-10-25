package subscribehandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/callhandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
)

func Test_processEventFMActiveflowUpdated_cov(t *testing.T) {
	tests := []struct {
		name string
		data []byte

		expectArg     *fmactiveflow.Activeflow
		callErr       error
		expectHandler bool
		expectErr     bool
	}{
		{name: "unmarshal failure", data: []byte(`{invalid`), expectErr: true},
		{name: "call handler fails but event is ok", data: []byte(`{"id":"79348cda-f159-11ee-ab1d-8fb25dd5896d"}`), expectHandler: true, expectArg: &fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("79348cda-f159-11ee-ab1d-8fb25dd5896d")}}, callErr: errors.New("a")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCall := callhandler.NewMockCallHandler(mc)
			h := &subscribeHandler{callHandler: mockCall}

			if tt.expectHandler {
				mockCall.EXPECT().EventFMActiveflowUpdated(gomock.Any(), tt.expectArg).Return(tt.callErr)
			}

			err := h.processEventFMActiveflowUpdated(context.Background(), &sock.Event{Data: tt.data})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
