package callhandler

import (
	"context"
	"testing"

	tmtts "monorepo/bin-tts-manager/models/tts"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/playback"
)

func Test_Talk_cov2a_errors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name: "call lookup error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			wantErr: "could not get call info",
		},
		{
			name: "answer error",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.Status = call.StatusRinging
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.channel.EXPECT().Answer(gomock.Any(), c.ChannelID).Return(errCov2aBoom)
			},
			wantErr: "could not answer the call",
		},
		{
			name: "tts creation error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.req.EXPECT().TTSV1SpeecheCreate(gomock.Any(), c.ID, "hello", "en-US", tmtts.Provider(""), "", 10000).Return(nil, errCov2aBoom)
			},
			wantErr: "could not create tts wav",
		},
		{
			name: "play error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.req.EXPECT().TTSV1SpeecheCreate(gomock.Any(), c.ID, "hello", "en-US", tmtts.Provider(""), "", 10000).
					Return(&tmtts.TTS{MediaFilepath: "http://tts.example.invalid/a.wav"}, nil)
				m.channel.EXPECT().Play(gomock.Any(), c.ChannelID, playback.IDPrefixCall+c.Action.ID.String(), []string{"sound:http://tts.example.invalid/a.wav"}, "", 0, 0).Return(errCov2aBoom)
			},
			wantErr: "could not play the media for tts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			err := h.Talk(context.Background(), c.ID, true, "hello", "en-US", "", "")
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}

func Test_Play_cov2a_errors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name: "call lookup error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			wantErr: "could not get call info",
		},
		{
			name: "play error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.channel.EXPECT().Play(gomock.Any(), c.ChannelID, playback.IDPrefixCall+"00000000-0000-0000-0000-000000000000", []string{"sound:http://media.example.invalid/a.wav"}, "", 0, 0).Return(errCov2aBoom)
			},
			wantErr: "could not play the media",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			err := h.Play(context.Background(), c.ID, false, []string{"http://media.example.invalid/a.wav"})
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}

func Test_MediaStop_cov2a(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name: "call lookup error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			wantErr: "could not get call info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			tt.setup(m, c)

			err := h.MediaStop(context.Background(), c.ID)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}
