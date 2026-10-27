package callhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmvariable "monorepo/bin-flow-manager/models/variable"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/playback"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

// cov2aMocks bundles the mocks used by the cov2a tests of this package.
type cov2aMocks struct {
	util       *utilhandler.MockUtilHandler
	req        *requesthandler.MockRequestHandler
	db         *dbhandler.MockDBHandler
	notify     *notifyhandler.MockNotifyHandler
	channel    *channelhandler.MockChannelHandler
	bridge     *bridgehandler.MockBridgeHandler
	confbridge *confbridgehandler.MockConfbridgeHandler
	groupcall  *groupcallhandler.MockGroupcallHandler
	recording  *recordinghandler.MockRecordingHandler
	extMedia   *externalmediahandler.MockExternalMediaHandler
}

func cov2aNewHandler(t *testing.T) (*callHandler, *cov2aMocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &cov2aMocks{
		util:       utilhandler.NewMockUtilHandler(mc),
		req:        requesthandler.NewMockRequestHandler(mc),
		db:         dbhandler.NewMockDBHandler(mc),
		notify:     notifyhandler.NewMockNotifyHandler(mc),
		channel:    channelhandler.NewMockChannelHandler(mc),
		bridge:     bridgehandler.NewMockBridgeHandler(mc),
		confbridge: confbridgehandler.NewMockConfbridgeHandler(mc),
		groupcall:  groupcallhandler.NewMockGroupcallHandler(mc),
		recording:  recordinghandler.NewMockRecordingHandler(mc),
		extMedia:   externalmediahandler.NewMockExternalMediaHandler(mc),
	}
	h := &callHandler{
		utilHandler:          m.util,
		reqHandler:           m.req,
		db:                   m.db,
		notifyHandler:        m.notify,
		channelHandler:       m.channel,
		bridgeHandler:        m.bridge,
		confbridgeHandler:    m.confbridge,
		groupcallHandler:     m.groupcall,
		recordingHandler:     m.recording,
		externalMediaHandler: m.extMedia,
	}
	return h, m
}

const (
	cov2aCallID       = "44444444-0000-0000-0000-000000000001"
	cov2aActionID     = "44444444-0000-0000-0000-000000000002"
	cov2aActiveflowID = "44444444-0000-0000-0000-000000000003"
	cov2aCustomerID   = "44444444-0000-0000-0000-000000000004"
	cov2aOtherID      = "44444444-0000-0000-0000-000000000005"
)

var errCov2aBoom = errors.New("cov2a boom")

// cov2aNewCall returns a progressing call with the given action option.
func cov2aNewCall(opt map[string]any) *call.Call {
	c := &call.Call{
		ChannelID:    "cov2a-ch-1",
		ActiveflowID: uuid.FromStringOrNil(cov2aActiveflowID),
		Status:       call.StatusProgressing,
	}
	c.ID = uuid.FromStringOrNil(cov2aCallID)
	c.CustomerID = uuid.FromStringOrNil(cov2aCustomerID)
	c.Action = fmaction.Action{
		ID:     uuid.FromStringOrNil(cov2aActionID),
		Option: opt,
	}
	return c
}

func cov2aCheckErr(t *testing.T, err error, wantErr string) {
	t.Helper()
	if wantErr == "" {
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Errorf("Wrong match. expect error containing: %q, got: %v", wantErr, err)
	}
}

func Test_actionExecute_cov2a_parseOptionError(t *testing.T) {
	badOpt := map[string]any{"x": make(chan int)}

	tests := []struct {
		name string
		exec func(h *callHandler, ctx context.Context, c *call.Call) error
	}{
		{"answer", func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteAnswer(ctx, c) }},
		{"confbridge join", func(h *callHandler, ctx context.Context, c *call.Call) error {
			return h.actionExecuteConfbridgeJoin(ctx, c)
		}},
		{"play", func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecutePlay(ctx, c) }},
		{"stream echo", func(h *callHandler, ctx context.Context, c *call.Call) error {
			return h.actionExecuteStreamEcho(ctx, c)
		}},
		{"hangup", func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteHangup(ctx, c) }},
		{"talk", func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteTalk(ctx, c) }},
		{"recording start", func(h *callHandler, ctx context.Context, c *call.Call) error {
			return h.actionExecuteRecordingStart(ctx, c)
		}},
		{"recording stop", func(h *callHandler, ctx context.Context, c *call.Call) error {
			return h.actionExecuteRecordingStop(ctx, c)
		}},
		{"digits receive", func(h *callHandler, ctx context.Context, c *call.Call) error {
			return h.actionExecuteDigitsReceive(ctx, c)
		}},
		{"digits send", func(h *callHandler, ctx context.Context, c *call.Call) error {
			return h.actionExecuteDigitsSend(ctx, c)
		}},
		{"external media start", func(h *callHandler, ctx context.Context, c *call.Call) error {
			return h.actionExecuteExternalMediaStart(ctx, c)
		}},
		{"amd", func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteAMD(ctx, c) }},
		{"sleep", func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteSleep(ctx, c) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := cov2aNewHandler(t)

			err := tt.exec(h, context.Background(), cov2aNewCall(badOpt))
			cov2aCheckErr(t, err, "could not parse the option")
		})
	}
}

func Test_actionExecute_cov2a_dispatch(t *testing.T) {
	tests := []struct {
		name       string
		actionID   uuid.UUID
		actionType fmaction.Type
		opt        map[string]any
		setup      func(m *cov2aMocks, c *call.Call)

		expectType fmaction.Type
	}{
		{
			name:       "action id finish is converted to hangup",
			actionID:   fmaction.IDFinish,
			actionType: fmaction.TypeAnswer,
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			expectType: fmaction.TypeHangup,
		},
		{
			name:       "action id empty is converted to hangup",
			actionID:   fmaction.IDEmpty,
			actionType: fmaction.TypeAnswer,
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			expectType: fmaction.TypeHangup,
		},
		{
			name:       "beep",
			actionID:   uuid.FromStringOrNil(cov2aActionID),
			actionType: fmaction.TypeBeep,
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().Play(gomock.Any(), c.ChannelID, playback.IDPrefixCall+c.Action.ID.String(), []string{"sound:beep"}, "", 0, 0).Return(nil)
			},
			expectType: fmaction.TypeBeep,
		},
		{
			name:       "echo with default duration",
			actionID:   uuid.FromStringOrNil(cov2aActionID),
			actionType: fmaction.TypeEcho,
			opt:        map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().Continue(gomock.Any(), c.ChannelID, "svc-echo", "s", 1, "").Return(nil)
				m.req.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, 180000, &c.Action).Return(nil)
			},
			expectType: fmaction.TypeEcho,
		},
		{
			name:       "sleep",
			actionID:   uuid.FromStringOrNil(cov2aActionID),
			actionType: fmaction.TypeSleep,
			opt:        map[string]any{"duration": 1500},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, 1500, &c.Action).Return(nil)
			},
			expectType: fmaction.TypeSleep,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(tt.opt)
			c.Action.ID = tt.actionID
			c.Action.Type = tt.actionType
			tt.setup(m, c)

			if err := h.actionExecute(context.Background(), c); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if c.Action.Type != tt.expectType {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectType, c.Action.Type)
			}
		})
	}
}

func Test_actionExecute_cov2a_errorPaths(t *testing.T) {
	cbID := uuid.FromStringOrNil(cov2aOtherID)
	emID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000007")
	snoopID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000008")

	tests := []struct {
		name  string
		opt   map[string]any
		setup func(m *cov2aMocks, c *call.Call)
		exec  func(h *callHandler, ctx context.Context, c *call.Call) error

		wantErr string
	}{
		{
			name: "answer: channel answer error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().Answer(gomock.Any(), c.ChannelID).Return(errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteAnswer(ctx, c) },
			wantErr: "could not answer",
		},
		{
			name: "answer: next action request error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().Answer(gomock.Any(), c.ChannelID).Return(nil)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteAnswer(ctx, c) },
			wantErr: "could not send the next action request",
		},
		{
			name: "confbridge join: join error",
			opt:  map[string]any{"confbridge_id": cbID.String()},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.confbridge.EXPECT().Join(gomock.Any(), cbID, c.ID).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteConfbridgeJoin(ctx, c)
			},
			wantErr: "boom",
		},
		{
			name: "play: call lookup error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecutePlay(ctx, c) },
			wantErr: "could not play the media correctly",
		},
		{
			name: "stream echo: continue error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().Continue(gomock.Any(), c.ChannelID, "svc-stream_echo", "s", 1, "").Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteStreamEcho(ctx, c)
			},
			wantErr: "could not continue the call for action",
		},
		{
			name: "stream echo: timeout error",
			opt:  map[string]any{"duration": 2000},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().Continue(gomock.Any(), c.ChannelID, "svc-stream_echo", "s", 1, "").Return(nil)
				m.req.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, 2000, &c.Action).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteStreamEcho(ctx, c)
			},
			wantErr: "could not set action timeout",
		},
		{
			name: "echo: continue error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().Continue(gomock.Any(), c.ChannelID, "svc-echo", "s", 1, "").Return(errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteEcho(ctx, c) },
			wantErr: "could not continue the call for action",
		},
		{
			name: "echo: timeout error",
			opt:  map[string]any{"duration": 3000},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().Continue(gomock.Any(), c.ChannelID, "svc-echo", "s", 1, "").Return(nil)
				m.req.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, 3000, &c.Action).Return(errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteEcho(ctx, c) },
			wantErr: "could not set action timeout",
		},
		{
			name: "talk: talk error",
			opt:  map[string]any{"text": "hello"},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteTalk(ctx, c) },
			wantErr: "could not talk correctly",
		},
		{
			name: "recording start: start error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteRecordingStart(ctx, c)
			},
			wantErr: "boom",
		},
		{
			name: "recording start: next action request error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil).Times(2)
				m.recording.EXPECT().Start(gomock.Any(), c.ActiveflowID, recording.ReferenceTypeCall, c.ID, recording.Format(""), 0, "", 0, uuid.Nil).
					Return(&recording.Recording{Identity: c.Identity}, nil)
				m.db.EXPECT().CallSetRecordingID(gomock.Any(), c.ID, c.ID).Return(nil)
				m.db.EXPECT().CallAddRecordingIDs(gomock.Any(), c.ID, c.ID).Return(nil)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteRecordingStart(ctx, c)
			},
			wantErr: "could not send the next action request",
		},
		{
			name: "recording stop: stop error is tolerated and next is requested",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(nil)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteRecordingStop(ctx, c)
			},
			wantErr: "",
		},
		{
			name: "recording stop: next action request error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteRecordingStop(ctx, c)
			},
			wantErr: "boom",
		},
		{
			name: "digits receive: variable lookup error",
			opt:  map[string]any{"length": 2},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableGet(gomock.Any(), c.ActiveflowID).Return(nil, errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteDigitsReceive(ctx, c)
			},
			wantErr: "boom",
		},
		{
			name: "digits receive: condition met but next action request fails",
			opt:  map[string]any{"length": 2},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableGet(gomock.Any(), c.ActiveflowID).
					Return(&fmvariable.Variable{Variables: map[string]string{variableCallDigits: "12"}}, nil)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteDigitsReceive(ctx, c)
			},
			wantErr: "could not send the next action request",
		},
		{
			name: "digits receive: timeout request error",
			opt:  map[string]any{"length": 5, "duration": 4000},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableGet(gomock.Any(), c.ActiveflowID).
					Return(&fmvariable.Variable{Variables: map[string]string{variableCallDigits: "1"}}, nil)
				m.req.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, 4000, &c.Action).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteDigitsReceive(ctx, c)
			},
			wantErr: "could not set action timeout",
		},
		{
			name: "digits send: dtmf send error",
			opt:  map[string]any{"digits": "12", "duration": 100, "interval": 50},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().DTMFSend(gomock.Any(), c.ChannelID, "12", 100, 0, 50, 0).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteDigitsSend(ctx, c)
			},
			wantErr: "could not send the dtmfs",
		},
		{
			name: "digits send: timeout request error with computed timeout",
			opt:  map[string]any{"digits": "123", "duration": 100, "interval": 50},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.channel.EXPECT().DTMFSend(gomock.Any(), c.ChannelID, "123", 100, 0, 50, 0).Return(nil)
				// 100 * 3 + 50 * 2
				m.req.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, 400, &c.Action).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteDigitsSend(ctx, c)
			},
			wantErr: "could not set action timeout",
		},
		{
			name: "external media start: start error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteExternalMediaStart(ctx, c)
			},
			wantErr: "boom",
		},
		{
			name: "external media start: success with external media ids",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				withMedia := cov2aNewCall(nil)
				withMedia.ExternalMediaIDs = []uuid.UUID{emID}
				gomock.InOrder(
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil),
					m.extMedia.EXPECT().Start(gomock.Any(), uuid.Nil, externalmedia.ReferenceTypeCall, c.ID, "", externalmedia.Encapsulation(""), externalmedia.Transport(""), "", "", "", externalmedia.Direction(""), externalmedia.Direction("")).
						Return(&externalmedia.ExternalMedia{ID: emID}, nil),
					m.db.EXPECT().CallAddExternalMediaID(gomock.Any(), c.ID, emID).Return(nil),
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(withMedia, nil),
					m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(nil),
				)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteExternalMediaStart(ctx, c)
			},
			wantErr: "",
		},
		{
			name: "external media stop: no external media",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(nil)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteExternalMediaStop(ctx, c)
			},
			wantErr: "",
		},
		{
			name: "external media stop: stop error is skipped and next is requested",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.ExternalMediaIDs = []uuid.UUID{emID}
				m.extMedia.EXPECT().Stop(gomock.Any(), emID).Return(nil, errCov2aBoom)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
			exec: func(h *callHandler, ctx context.Context, c *call.Call) error {
				return h.actionExecuteExternalMediaStop(ctx, c)
			},
			wantErr: "boom",
		},
		{
			name: "amd: snoop channel error",
			opt:  map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.util.EXPECT().UUIDCreate().Return(snoopID)
				m.channel.EXPECT().StartSnoop(gomock.Any(), c.ChannelID, snoopID.String(), gomock.Any(), channel.SnoopDirectionBoth, channel.SnoopDirectionBoth).Return(nil, errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteAMD(ctx, c) },
			wantErr: "could not create a snoop channel for the AMD",
		},
		{
			name: "amd: cache set error hangs up the snoop channel; async requests next",
			opt:  map[string]any{"async": true},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.util.EXPECT().UUIDCreate().Return(snoopID)
				m.channel.EXPECT().StartSnoop(gomock.Any(), c.ChannelID, snoopID.String(), gomock.Any(), channel.SnoopDirectionBoth, channel.SnoopDirectionBoth).
					Return(&channel.Channel{ID: "cov2a-snoop"}, nil)
				m.db.EXPECT().CallApplicationAMDSet(gomock.Any(), snoopID.String(), gomock.Any()).Return(errCov2aBoom)
				m.channel.EXPECT().HangingUp(gomock.Any(), "cov2a-snoop", ari.ChannelCauseNormalClearing).Return(nil, nil)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteAMD(ctx, c) },
			wantErr: "boom",
		},
		{
			name: "sleep: timeout request error",
			opt:  map[string]any{"duration": 1000},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, 1000, &c.Action).Return(errCov2aBoom)
			},
			exec:    func(h *callHandler, ctx context.Context, c *call.Call) error { return h.actionExecuteSleep(ctx, c) },
			wantErr: "could not set action timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(tt.opt)
			tt.setup(m, c)

			err := tt.exec(h, context.Background(), c)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}

func Test_ActionNext_cov2a(t *testing.T) {
	nextActionID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000010")

	tests := []struct {
		name   string
		modify func(c *call.Call)
		setup  func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name:   "call already hung up",
			modify: func(c *call.Call) { c.Status = call.StatusHangup },
			setup:  func(m *cov2aMocks, c *call.Call) {},
		},
		{
			name:   "action next is on hold",
			modify: func(c *call.Call) { c.ActionNextHold = true },
			setup:  func(m *cov2aMocks, c *call.Call) {},
		},
		{
			name:   "set action next hold error hangs up the call",
			modify: func(c *call.Call) {},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallSetActionNextHold(gomock.Any(), c.ID, true).Return(errCov2aBoom)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
		},
		{
			name:   "set action error moves to the next action request",
			modify: func(c *call.Call) {},
			setup: func(m *cov2aMocks, c *call.Call) {
				now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
				m.db.EXPECT().CallSetActionNextHold(gomock.Any(), c.ID, true).Return(nil)
				m.req.EXPECT().FlowV1ActiveflowGetNextAction(gomock.Any(), c.ActiveflowID, c.Action.ID).
					Return(&fmaction.Action{ID: nextActionID, Type: fmaction.TypeBeep}, nil)
				m.util.EXPECT().TimeNow().Return(&now)
				m.db.EXPECT().CallSetActionAndActionNextHold(gomock.Any(), c.ID, &fmaction.Action{ID: nextActionID, Type: fmaction.TypeBeep, TMExecute: &now}, false).Return(errCov2aBoom)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name:   "action execute error hangs up the call",
			modify: func(c *call.Call) {},
			setup: func(m *cov2aMocks, c *call.Call) {
				now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
				unknown := fmaction.Type("cov2a_unknown")
				updated := cov2aNewCall(nil)
				updated.Action = fmaction.Action{ID: nextActionID, Type: unknown, TMExecute: &now}

				m.db.EXPECT().CallSetActionNextHold(gomock.Any(), c.ID, true).Return(nil)
				m.req.EXPECT().FlowV1ActiveflowGetNextAction(gomock.Any(), c.ActiveflowID, c.Action.ID).
					Return(&fmaction.Action{ID: nextActionID, Type: unknown}, nil)
				m.util.EXPECT().TimeNow().Return(&now)
				m.db.EXPECT().CallSetActionAndActionNextHold(gomock.Any(), c.ID, &fmaction.Action{ID: nextActionID, Type: unknown, TMExecute: &now}, false).Return(nil)
				gomock.InOrder(
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(updated, nil),
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom),
				)
				m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), updated.CustomerID, call.EventTypeCallUpdated, updated)
				// the unknown action type fails and the failure notification request fails too.
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.modify(c)
			tt.setup(m, c)

			err := h.ActionNext(context.Background(), c)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}

func Test_ActionNextForce_cov2a(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)
	}{
		{
			name: "unhold error hangs up the call",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallSetActionNextHold(gomock.Any(), c.ID, false).Return(errCov2aBoom)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			if err := h.ActionNextForce(context.Background(), c); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionTimeout_cov2a(t *testing.T) {
	tests := []struct {
		name  string
		input func(c *call.Call) *fmaction.Action
		setup func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name:  "call lookup error",
			input: func(c *call.Call) *fmaction.Action { return &c.Action },
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "action id mismatch",
			input: func(c *call.Call) *fmaction.Action {
				return &fmaction.Action{ID: uuid.FromStringOrNil(cov2aOtherID)}
			},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
			},
			wantErr: "invalid timed out action condition",
		},
		{
			name:  "channel lookup error",
			input: func(c *call.Call) *fmaction.Action { return &c.Action },
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.channel.EXPECT().Get(gomock.Any(), c.ChannelID).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name:  "channel outside of the stasis is redirected",
			input: func(c *call.Call) *fmaction.Action { return &c.Action },
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.channel.EXPECT().Get(gomock.Any(), c.ChannelID).Return(&channel.Channel{ID: c.ChannelID}, nil)
				m.channel.EXPECT().Redirect(gomock.Any(), c.ChannelID, "svc-stasis", "s", "1").Return(errCov2aBoom)
			},
			wantErr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			err := h.ActionTimeout(context.Background(), c.ID, tt.input(c))
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}
