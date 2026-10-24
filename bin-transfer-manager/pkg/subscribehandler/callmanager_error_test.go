package subscribehandler

import (
	"context"
	"errors"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

const (
	testEventID    = "7a1c0e10-0001-11ee-8b01-000000000001"
	testTransferID = "7a1c0e10-0002-11ee-8b01-000000000002"
)

func newErrorTestHandler(mc *gomock.Controller) (*subscribeHandler, *transferhandler.MockTransferHandler) {
	mockTransfer := transferhandler.NewMockTransferHandler(mc)
	h := &subscribeHandler{
		sockHandler:     sockhandler.NewMockSockHandler(mc),
		transferHandler: mockTransfer,
	}
	return h, mockTransfer
}

func Test_processEventCM_invalidJSON(t *testing.T) {
	tests := []struct {
		name    string
		process func(h *subscribeHandler, ctx context.Context, m *sock.Event) error
		evtType string
	}{
		{
			name: "groupcall progressing",
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMGroupcallProgressing(ctx, m)
			},
			evtType: string(cmgroupcall.EventTypeGroupcallProgressing),
		},
		{
			name: "groupcall hangup",
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMGroupcallHangup(ctx, m)
			},
			evtType: string(cmgroupcall.EventTypeGroupcallHangup),
		},
		{
			name: "call hangup",
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMCallHangup(ctx, m)
			},
			evtType: string(cmcall.EventTypeCallHangup),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectation on the transfer handler: any lookup or action fails the test.
			h, _ := newErrorTestHandler(mc)

			err := tt.process(h, context.Background(), &sock.Event{
				Publisher: publisherCallManager,
				Type:      tt.evtType,
				Data:      []byte(`{invalid json`),
			})
			if err == nil {
				t.Errorf("Wrong match. expect: unmarshal error, got: ok")
			}
		})
	}
}

func Test_processEventCM_lookupMissIsIgnored(t *testing.T) {
	data := []byte(`{"id":"` + testEventID + `"}`)
	id := uuid.FromStringOrNil(testEventID)

	tests := []struct {
		name    string
		expect  func(m *transferhandler.MockTransferHandler)
		process func(h *subscribeHandler, ctx context.Context, m *sock.Event) error
	}{
		{
			name: "groupcall progressing",
			expect: func(m *transferhandler.MockTransferHandler) {
				m.EXPECT().GetByGroupcallID(gomock.Any(), id).Return(nil, errors.New("not found"))
			},
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMGroupcallProgressing(ctx, m)
			},
		},
		{
			name: "groupcall hangup",
			expect: func(m *transferhandler.MockTransferHandler) {
				m.EXPECT().GetByGroupcallID(gomock.Any(), id).Return(nil, errors.New("not found"))
			},
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMGroupcallHangup(ctx, m)
			},
		},
		{
			name: "call hangup",
			expect: func(m *transferhandler.MockTransferHandler) {
				m.EXPECT().GetByTransfererCallID(gomock.Any(), id).Return(nil, errors.New("not found"))
			},
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMCallHangup(ctx, m)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockTransfer := newErrorTestHandler(mc)
			// the Answer and Hangup actions have no expectation, so calling them fails the test.
			tt.expect(mockTransfer)

			if err := tt.process(h, context.Background(), &sock.Event{Publisher: publisherCallManager, Data: data}); err != nil {
				t.Errorf("Wrong match. expect: nil because the event is not for a transfer, got: %v", err)
			}
		})
	}
}

func Test_processEventCM_actionErrorIsLogged(t *testing.T) {
	data := []byte(`{"id":"` + testEventID + `"}`)
	id := uuid.FromStringOrNil(testEventID)
	tr := &transfer.Transfer{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(testTransferID)}}

	tests := []struct {
		name    string
		expect  func(m *transferhandler.MockTransferHandler)
		process func(h *subscribeHandler, ctx context.Context, m *sock.Event) error
	}{
		{
			name: "groupcall progressing",
			expect: func(m *transferhandler.MockTransferHandler) {
				m.EXPECT().GetByGroupcallID(gomock.Any(), id).Return(tr, nil)
				m.EXPECT().TransfereeAnswer(gomock.Any(), tr, &cmgroupcall.Groupcall{Identity: commonidentity.Identity{ID: id}}).Return(errors.New("answer failed"))
			},
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMGroupcallProgressing(ctx, m)
			},
		},
		{
			name: "groupcall hangup",
			expect: func(m *transferhandler.MockTransferHandler) {
				m.EXPECT().GetByGroupcallID(gomock.Any(), id).Return(tr, nil)
				m.EXPECT().TransfereeHangup(gomock.Any(), tr, &cmgroupcall.Groupcall{Identity: commonidentity.Identity{ID: id}}).Return(errors.New("hangup failed"))
			},
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMGroupcallHangup(ctx, m)
			},
		},
		{
			name: "call hangup",
			expect: func(m *transferhandler.MockTransferHandler) {
				m.EXPECT().GetByTransfererCallID(gomock.Any(), id).Return(tr, nil)
				m.EXPECT().TransfererHangup(gomock.Any(), tr, &cmcall.Call{Identity: commonidentity.Identity{ID: id}}).Return(errors.New("hangup failed"))
			},
			process: func(h *subscribeHandler, ctx context.Context, m *sock.Event) error {
				return h.processEventCMCallHangup(ctx, m)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockTransfer := newErrorTestHandler(mc)
			tt.expect(mockTransfer)

			// an action failure is logged and the event is still consumed (nil), so the
			// broker does not redeliver it.
			if err := tt.process(h, context.Background(), &sock.Event{Publisher: publisherCallManager, Data: data}); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})
	}
}
