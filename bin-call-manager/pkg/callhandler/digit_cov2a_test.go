package callhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmvariable "monorepo/bin-flow-manager/models/variable"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/dtmf"
)

func Test_digitsReceived_cov2a(t *testing.T) {
	badOpt := map[string]any{"x": make(chan int)}

	tests := []struct {
		name       string
		actionType fmaction.Type
		opt        map[string]any
		setup      func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name:       "digits receive: variable set error drops the event",
			actionType: fmaction.TypeDigitsReceive,
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), c.ActiveflowID, map[string]string{variableCallDigits: "${" + variableCallDigits + "}5"}).Return(errCov2aBoom)
			},
		},
		{
			name:       "digits receive: option parse error",
			actionType: fmaction.TypeDigitsReceive,
			opt:        badOpt,
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), c.ActiveflowID, gomock.Any()).Return(nil)
			},
			wantErr: "could not parse the option",
		},
		{
			name:       "digits receive: next request error hangs up the call",
			actionType: fmaction.TypeDigitsReceive,
			opt:        map[string]any{"length": 1},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), c.ActiveflowID, gomock.Any()).Return(nil)
				m.req.EXPECT().FlowV1VariableGet(gomock.Any(), c.ActiveflowID).
					Return(&fmvariable.Variable{Variables: map[string]string{variableCallDigits: "5"}}, nil)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
		},
		{
			name:       "talk: variable set error is only logged",
			actionType: fmaction.TypeTalk,
			opt:        map[string]any{},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), c.ActiveflowID, map[string]string{variableCallDigits: "5"}).Return(errCov2aBoom)
			},
		},
		{
			name:       "talk: option parse error",
			actionType: fmaction.TypeTalk,
			opt:        badOpt,
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), c.ActiveflowID, map[string]string{variableCallDigits: "5"}).Return(nil)
			},
			wantErr: "could not parse the option",
		},
		{
			name:       "talk: digits handle next and next request error hangs up the call",
			actionType: fmaction.TypeTalk,
			opt:        map[string]any{"digits_handle": "next"},
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), c.ActiveflowID, map[string]string{variableCallDigits: "5"}).Return(nil)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, true).Return(errCov2aBoom)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
		},
		{
			name:       "other action: variable set error is only logged",
			actionType: fmaction.TypePlay,
			setup: func(m *cov2aMocks, c *call.Call) {
				m.req.EXPECT().FlowV1VariableSetVariable(gomock.Any(), c.ActiveflowID, map[string]string{variableCallDigits: "5"}).Return(errCov2aBoom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)
			ctx := context.Background()

			now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			c := cov2aNewCall(tt.opt)
			c.Action.Type = tt.actionType
			cn := &channel.Channel{ID: c.ChannelID}

			m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
			m.util.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("44444444-0000-0000-0000-000000000040"))
			m.util.EXPECT().TimeNow().Return(&now)
			m.notify.EXPECT().PublishEvent(ctx, dtmf.EventTypeDTMFReceived, gomock.Any())
			tt.setup(m, c)

			err := h.digitsReceived(ctx, cn, "5", 100)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}

func Test_DigitsGet_cov2a(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)

		expectRes string
		wantErr   string
	}{
		{
			name: "call lookup error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "variable lookup error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.req.EXPECT().FlowV1VariableGet(gomock.Any(), c.ActiveflowID).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "digits are returned",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.req.EXPECT().FlowV1VariableGet(gomock.Any(), c.ActiveflowID).
					Return(&fmvariable.Variable{Variables: map[string]string{variableCallDigits: "123#"}}, nil)
			},
			expectRes: "123#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			got, err := h.DigitsGet(context.Background(), c.ID)
			cov2aCheckErr(t, err, tt.wantErr)
			if got != tt.expectRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectRes, got)
			}
		})
	}
}
