package callhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestExternalMediaStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name string
		call *call.Call

		externalHost   string
		encapsulation  string
		transport      string
		connectionType string
		format         string
		direction      string
		data           string

		expectExternalHost   string
		expectEncapsulation  string
		expectTransport      string
		expectConnectionType string
		expectFormat         string
		expectDirection      string
		expectData           string
	}

	tests := []test{
		{
			"default",
			&call.Call{
				ID:         uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "8066017c-02fb-11ec-ba6c-c320820accf1",
			},
			"example.com",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",
			"",

			"example.com",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstBridgeCreate(tt.call.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(nil)
			mockReq.EXPECT().AstChannelCreateSnoop(tt.call.AsteriskID, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirectionBoth, channel.SnoopDirectionBoth).Return(nil)
			mockReq.EXPECT().AstChannelExternalMedia(tt.call.AsteriskID, gomock.Any(), tt.expectExternalHost, tt.expectEncapsulation, tt.expectTransport, tt.expectConnectionType, tt.expectFormat, tt.expectDirection, tt.expectData, gomock.Any()).Return(nil)
			if err := h.ExternalMediaStart(tt.call.ID, tt.call.UserID, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
