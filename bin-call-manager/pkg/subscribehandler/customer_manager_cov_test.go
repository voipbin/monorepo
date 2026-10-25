package subscribehandler

import (
	"context"
	"errors"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

func Test_processEventCUCustomerDeleted_cov(t *testing.T) {
	tests := []struct {
		name string
		data []byte

		callErr, groupErr, confErr error
		expectHandlers             bool
		expectErr                  bool
	}{
		{name: "unmarshal failure", data: []byte(`{invalid`), expectErr: true},
		{name: "all handlers fail but event is ok", data: []byte(`{"id":"79348cda-f159-11ee-ab1d-8fb25dd5896d"}`), expectHandlers: true, callErr: errors.New("a"), groupErr: errors.New("b"), confErr: errors.New("c")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroup := groupcallhandler.NewMockGroupcallHandler(mc)
			mockConf := confbridgehandler.NewMockConfbridgeHandler(mc)
			h := &subscribeHandler{callHandler: mockCall, groupcallHandler: mockGroup, confbridgeHandler: mockConf}

			if tt.expectHandlers {
				mockCall.EXPECT().EventCUCustomerDeleted(gomock.Any(), gomock.Any()).Return(tt.callErr)
				mockGroup.EXPECT().EventCUCustomerDeleted(gomock.Any(), gomock.Any()).Return(tt.groupErr)
				mockConf.EXPECT().EventCUCustomerDeleted(gomock.Any(), gomock.Any()).Return(tt.confErr)
			}

			err := h.processEventCUCustomerDeleted(context.Background(), &sock.Event{Data: tt.data})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEventCUCustomerFrozen_cov(t *testing.T) {
	tests := []struct {
		name string
		data []byte

		callErr       error
		expectHandler bool
		expectErr     bool
	}{
		{name: "unmarshal failure", data: []byte(`{invalid`), expectErr: true},
		{name: "call handler fails but event is ok", data: []byte(`{"id":"79348cda-f159-11ee-ab1d-8fb25dd5896d"}`), expectHandler: true, callErr: errors.New("a")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCall := callhandler.NewMockCallHandler(mc)
			h := &subscribeHandler{callHandler: mockCall}

			if tt.expectHandler {
				mockCall.EXPECT().EventCUCustomerFrozen(gomock.Any(), gomock.Any()).Return(tt.callErr)
			}

			err := h.processEventCUCustomerFrozen(context.Background(), &sock.Event{Data: tt.data})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
