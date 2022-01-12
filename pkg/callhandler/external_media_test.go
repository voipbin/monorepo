package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func TestExternalMediaStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	tests := []struct {
		name string
		call *call.Call

		externalHost   string
		encapsulation  string
		transport      string
		connectionType string
		format         string
		direction      string

		expectExternalHost   string
		expectEncapsulation  string
		expectTransport      string
		expectConnectionType string
		expectFormat         string
		expectDirection      string
	}{
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

			"example.com",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.call.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(nil)
			mockReq.EXPECT().AstChannelCreateSnoop(gomock.Any(), tt.call.AsteriskID, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirectionBoth, channel.SnoopDirectionBoth).Return(nil)
			mockReq.EXPECT().AstChannelExternalMedia(gomock.Any(), tt.call.AsteriskID, gomock.Any(), tt.expectExternalHost, tt.expectEncapsulation, tt.expectTransport, tt.expectConnectionType, tt.expectFormat, tt.expectDirection, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)
			_, err := h.ExternalMediaStart(context.Background(), tt.call.ID, false, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestExternalMediaStop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	tests := []struct {
		name     string
		call     *call.Call
		extMedia *externalmedia.ExternalMedia
	}{
		{
			"default",
			&call.Call{
				ID:         uuid.FromStringOrNil("8e84ccd4-1afc-11ec-a0c7-673cb57f1064"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "8eee1c0c-1afc-11ec-947d-1b608000e61c",
			},
			&externalmedia.ExternalMedia{
				CallID:         uuid.FromStringOrNil("8e84ccd4-1afc-11ec-a0c7-673cb57f1064"),
				AsteriskID:     "42:01:0a:a4:00:05",
				ChannelID:      "abab8726-1afc-11ec-873c-df1a7c3c2f22",
				LocalIP:        "127.0.0.1",
				LocalPort:      10001,
				ExternalHost:   "example.com",
				Encapsulation:  "rtp",
				Transport:      "udp",
				ConnectionType: "client",
				Format:         "ulaw",
				Direction:      "both",
			},
		},
		{
			"have no external media",
			&call.Call{
				ID:         uuid.FromStringOrNil("af2fcbae-1afd-11ec-9e9c-b3602bb157eb"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "af7005ac-1afd-11ec-b889-9fea8ae57cf3",
			},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ExternalMediaGet(gomock.Any(), tt.call.ID).Return(tt.extMedia, nil)
			if tt.extMedia != nil {
				mockReq.EXPECT().AstChannelHangup(gomock.Any(), tt.extMedia.AsteriskID, tt.extMedia.ChannelID, ari.ChannelCauseNormalClearing).Return(nil)
				mockDB.EXPECT().ExternalMediaDelete(gomock.Any(), tt.call.ID).Return(nil)
			}

			if err := h.ExternalMediaStop(context.Background(), tt.call.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
