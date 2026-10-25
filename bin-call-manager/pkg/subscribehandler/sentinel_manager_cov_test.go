package subscribehandler

import (
	"context"
	"errors"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/callhandler"
	smcontainer "monorepo/bin-sentinel-manager/models/container"
)

func Test_processEventSMContainerDied_cov(t *testing.T) {
	tests := []struct {
		name string
		data []byte

		expectArg     *smcontainer.Event
		callErr       error
		expectHandler bool
		expectErr     bool
	}{
		{name: "unmarshal failure", data: []byte(`{invalid`), expectErr: true},
		{name: "call handler fails but event is ok", data: []byte(`{"container_name":"cn-1","service":"asterisk-call","asterisk_id":"ast-1"}`), expectHandler: true, expectArg: &smcontainer.Event{ContainerName: "cn-1", Service: "asterisk-call", AsteriskID: "ast-1"}, callErr: errors.New("a")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCall := callhandler.NewMockCallHandler(mc)
			h := &subscribeHandler{callHandler: mockCall}

			if tt.expectHandler {
				mockCall.EXPECT().EventSMContainerDied(gomock.Any(), tt.expectArg).Return(tt.callErr)
			}

			err := h.processEventSMContainerDied(context.Background(), &sock.Event{Data: tt.data})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
