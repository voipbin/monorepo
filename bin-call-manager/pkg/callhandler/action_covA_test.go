package callhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/playback"
	"monorepo/bin-call-manager/pkg/channelhandler"
)

func covACall(opt map[string]any) *call.Call {
	c := &call.Call{ChannelID: "ch-1"}
	c.ID = uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001")
	c.Action = fmaction.Action{
		ID:     uuid.FromStringOrNil("33333333-0000-0000-0000-000000000002"),
		Option: opt,
	}
	return c
}

func Test_actionExecuteBeep_covA(t *testing.T) {
	tests := []struct {
		name      string
		opt       map[string]any
		playErr   error
		expectErr bool
	}{
		{name: "nil option", opt: nil},
		{name: "with option", opt: map[string]any{}},
		{name: "invalid option", opt: map[string]any{"x": make(chan int)}, expectErr: true},
		{name: "play error", opt: nil, playErr: fmt.Errorf("err"), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := &callHandler{channelHandler: mockChannel}
			c := covACall(tt.opt)

			if tt.name != "invalid option" {
				mockChannel.EXPECT().Play(gomock.Any(), "ch-1", playback.IDPrefixCall+c.Action.ID.String(), []string{"sound:beep"}, "", 0, 0).Return(tt.playErr)
			}

			err := h.actionExecuteBeep(context.Background(), c)
			if (err != nil) != tt.expectErr {
				t.Errorf("err: %v, expectErr: %v", err, tt.expectErr)
			}
		})
	}
}

func Test_actionExecuteSleep_covA(t *testing.T) {
	tests := []struct {
		name       string
		opt        map[string]any
		timeoutErr error
		duration   int
		expectErr  bool
		expectCall bool
	}{
		{name: "normal", opt: map[string]any{"duration": 3000}, duration: 3000, expectCall: true},
		{name: "nil option", opt: nil, duration: 0, expectCall: true},
		{name: "invalid option", opt: map[string]any{"duration": "abc"}, expectErr: true},
		{name: "timeout error", opt: map[string]any{"duration": 10}, duration: 10, timeoutErr: fmt.Errorf("err"), expectErr: true, expectCall: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &callHandler{reqHandler: mockReq}
			c := covACall(tt.opt)

			if tt.expectCall {
				mockReq.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, tt.duration, &c.Action).Return(tt.timeoutErr)
			}

			err := h.actionExecuteSleep(context.Background(), c)
			if (err != nil) != tt.expectErr {
				t.Errorf("err: %v, expectErr: %v", err, tt.expectErr)
			}
		})
	}
}

func Test_actionExecuteEcho_covA(t *testing.T) {
	tests := []struct {
		name        string
		opt         map[string]any
		continueErr error
		timeoutErr  error
		duration    int
		expectErr   bool
		expectCont  bool
		expectTO    bool
	}{
		{name: "given duration", opt: map[string]any{"duration": 5000}, duration: 5000, expectCont: true, expectTO: true},
		{name: "default duration", opt: map[string]any{}, duration: 180000, expectCont: true, expectTO: true},
		{name: "invalid option", opt: map[string]any{"duration": "abc"}, expectErr: true},
		{name: "continue error", opt: map[string]any{}, continueErr: fmt.Errorf("err"), expectErr: true, expectCont: true},
		{name: "timeout error", opt: map[string]any{}, duration: 180000, timeoutErr: fmt.Errorf("err"), expectErr: true, expectCont: true, expectTO: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := &callHandler{reqHandler: mockReq, channelHandler: mockChannel}
			c := covACall(tt.opt)

			if tt.expectCont {
				mockChannel.EXPECT().Continue(gomock.Any(), "ch-1", "svc-echo", "s", 1, "").Return(tt.continueErr)
			}
			if tt.expectTO {
				mockReq.EXPECT().CallV1CallActionTimeout(gomock.Any(), c.ID, tt.duration, &c.Action).Return(tt.timeoutErr)
			}

			err := h.actionExecuteEcho(context.Background(), c)
			if (err != nil) != tt.expectErr {
				t.Errorf("err: %v, expectErr: %v", err, tt.expectErr)
			}
		})
	}
}
