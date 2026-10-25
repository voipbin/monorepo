package callhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/channelhandler"
)

func covAQueryResponse() map[string]interface{} {
	return map[string]interface{}{
		"tags": map[string]interface{}{
			"ext": map[string]interface{}{
				"medias": []interface{}{
					map[string]interface{}{
						"streams": []interface{}{
							map[string]interface{}{"local port": float64(30000)},
						},
					},
				},
			},
			"int": map[string]interface{}{
				"medias": []interface{}{
					map[string]interface{}{
						"streams": []interface{}{
							map[string]interface{}{"local port": float64(30001)},
						},
					},
				},
			},
		},
	}
}

func Test_rtpDebugStartRecording_covA(t *testing.T) {
	c := &call.Call{}
	c.ID = uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	newCn := func(addr string) *channel.Channel {
		return &channel.Channel{
			ID:        "ch-1",
			SIPCallID: "sip-call-1",
			SIPData: map[string]string{
				channel.SIPDataKeyRTPEngineAddress: addr,
				"from_tag":                         "int",
			},
		}
	}

	tests := []struct {
		name  string
		cn    *channel.Channel
		setup func(m *requesthandler.MockRequestHandler)
	}{
		{
			name:  "no address",
			cn:    newCn(""),
			setup: func(m *requesthandler.MockRequestHandler) {},
		},
		{
			name: "query error",
			cn:   newCn("rtp-1"),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().RTPEngineV1CommandsSend(gomock.Any(), "rtp-1", gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
		},
		{
			name: "extract error",
			cn:   newCn("rtp-1"),
			setup: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().RTPEngineV1CommandsSend(gomock.Any(), "rtp-1", gomock.Any()).Return(map[string]interface{}{}, nil)
			},
		},
		{
			name: "exec error",
			cn:   newCn("rtp-1"),
			setup: func(m *requesthandler.MockRequestHandler) {
				gomock.InOrder(
					m.EXPECT().RTPEngineV1CommandsSend(gomock.Any(), "rtp-1", gomock.Any()).Return(covAQueryResponse(), nil),
					m.EXPECT().RTPEngineV1CommandsSend(gomock.Any(), "rtp-1", gomock.Any()).Return(nil, fmt.Errorf("err")),
				)
			},
		},
		{
			name: "success",
			cn:   newCn("rtp-1"),
			setup: func(m *requesthandler.MockRequestHandler) {
				gomock.InOrder(
					m.EXPECT().RTPEngineV1CommandsSend(gomock.Any(), "rtp-1", map[string]interface{}{
						"type": "ng", "command": "query", "call-id": "sip-call-1",
					}).Return(covAQueryResponse(), nil),
					m.EXPECT().RTPEngineV1CommandsSend(gomock.Any(), "rtp-1", gomock.Any()).DoAndReturn(
						func(_ context.Context, _ string, msg map[string]interface{}) (map[string]interface{}, error) {
							expect := map[string]interface{}{
								"type": "exec", "id": "sip-call-1", "command": "tcpdump",
								"parameters": []string{"udp port 30000"},
							}
							if !reflect.DeepEqual(msg, expect) {
								t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, msg)
							}
							return nil, nil
						}),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &callHandler{reqHandler: mockReq}
			tt.setup(mockReq)

			h.rtpDebugStartRecording(context.Background(), c, tt.cn)
		})
	}
}

func Test_rtpDebugStopRecording_covA(t *testing.T) {
	c := &call.Call{ChannelID: "ch-1"}
	c.ID = uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")

	tests := []struct {
		name  string
		setup func(ch *channelhandler.MockChannelHandler, rq *requesthandler.MockRequestHandler)
	}{
		{
			name: "channel get error",
			setup: func(ch *channelhandler.MockChannelHandler, rq *requesthandler.MockRequestHandler) {
				ch.EXPECT().Get(gomock.Any(), "ch-1").Return(nil, fmt.Errorf("err"))
			},
		},
		{
			name: "no address",
			setup: func(ch *channelhandler.MockChannelHandler, rq *requesthandler.MockRequestHandler) {
				ch.EXPECT().Get(gomock.Any(), "ch-1").Return(&channel.Channel{SIPData: map[string]string{}}, nil)
			},
		},
		{
			name: "send error",
			setup: func(ch *channelhandler.MockChannelHandler, rq *requesthandler.MockRequestHandler) {
				ch.EXPECT().Get(gomock.Any(), "ch-1").Return(&channel.Channel{
					SIPCallID: "sip-1", SIPData: map[string]string{channel.SIPDataKeyRTPEngineAddress: "rtp-1"},
				}, nil)
				rq.EXPECT().RTPEngineV1CommandsSend(gomock.Any(), "rtp-1", map[string]interface{}{"type": "kill", "id": "sip-1"}).Return(nil, fmt.Errorf("err"))
			},
		},
		{
			name: "success",
			setup: func(ch *channelhandler.MockChannelHandler, rq *requesthandler.MockRequestHandler) {
				ch.EXPECT().Get(gomock.Any(), "ch-1").Return(&channel.Channel{
					SIPCallID: "sip-1", SIPData: map[string]string{channel.SIPDataKeyRTPEngineAddress: "rtp-1"},
				}, nil)
				rq.EXPECT().RTPEngineV1CommandsSend(gomock.Any(), "rtp-1", map[string]interface{}{"type": "kill", "id": "sip-1"}).Return(nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := &callHandler{reqHandler: mockReq, channelHandler: mockChannel}
			tt.setup(mockChannel, mockReq)

			h.rtpDebugStopRecording(context.Background(), c)
		})
	}
}
