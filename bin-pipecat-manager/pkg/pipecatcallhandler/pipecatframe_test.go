package pipecatcallhandler

import (
	"context"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"net"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

type DummyConn struct {
	Written [][]byte
}

func NewDummyConn() *DummyConn {
	return &DummyConn{
		Written: make([][]byte, 0),
	}
}

func (d *DummyConn) Write(b []byte) (n int, err error) {
	cpy := make([]byte, len(b))
	copy(cpy, b)
	d.Written = append(d.Written, cpy)
	return len(b), nil
}
func (d *DummyConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (d *DummyConn) Close() error                       { return nil }
func (d *DummyConn) LocalAddr() net.Addr                { return nil }
func (d *DummyConn) RemoteAddr() net.Addr               { return nil }
func (d *DummyConn) SetDeadline(t time.Time) error      { return nil }
func (d *DummyConn) SetReadDeadline(t time.Time) error  { return nil }
func (d *DummyConn) SetWriteDeadline(t time.Time) error { return nil }

func Test_sendProtobufFrame(t *testing.T) {

	tests := []struct {
		name string

		ws    *websocket.Conn
		frame *pipecatframe.Frame

		expectFrame []byte
	}{
		{
			name: "normal audio",

			ws: &websocket.Conn{},
			frame: &pipecatframe.Frame{
				Frame: &pipecatframe.Frame_Audio{
					Audio: &pipecatframe.AudioRawFrame{
						Id:          1,
						Name:        "test-audio",
						Audio:       []byte{0x01, 0x02, 0x03, 0x04}, // PCM16 example
						SampleRate:  16000,
						NumChannels: 1,
					},
				},
			},
			expectFrame: []byte{
				0x12, 0x19, 0x08, 0x01, // field headers
				0x12, 0x0A, 't', 'e', 's', 't', '-', 'a', 'u', 'd', 'i', 'o', // Name="test-audio"
				0x1A, 0x04, 0x01, 0x02, 0x03, 0x04, // Audio data
				0x20, 0x80, 0x7D, // SampleRate=16000
				0x28, 0x01, // NumChannels=1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockWebsock := NewMockWebsocketHandler(mc)

			h := pipecatframeHandler{
				websocketHandler: mockWebsock,
			}

			mockWebsock.EXPECT().WriteMessage(tt.ws, websocket.BinaryMessage, tt.expectFrame).Return(nil)

			if err := h.sendFrame(tt.ws, tt.frame); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_SendAudio(t *testing.T) {
	tests := []struct {
		name string

		se        *pipecatcall.Session
		ws        *websocket.Conn
		packetID  uint64
		audioData []byte

		expectRes *pipecatframe.Frame
	}{
		{
			name: "simple audio frame",

			se: &pipecatcall.Session{
				Ctx:                 context.Background(),
				RunnerWebsocketChan: make(chan *pipecatframe.Frame, 1),
			},
			ws:        &websocket.Conn{},
			packetID:  1,
			audioData: []byte{0x01, 0x02, 0x03, 0x04},
			expectRes: &pipecatframe.Frame{
				Frame: &pipecatframe.Frame_Audio{
					Audio: &pipecatframe.AudioRawFrame{
						Id:          1,
						Audio:       []byte{0x01, 0x02, 0x03, 0x04},
						SampleRate:  defaultMediaSampleRate,
						NumChannels: defaultMediaNumChannel,
					},
				},
			},
		},
		{
			name: "empty audio frame",

			se: &pipecatcall.Session{
				Ctx:                 context.Background(),
				RunnerWebsocketChan: make(chan *pipecatframe.Frame, 1),
			},
			ws:        &websocket.Conn{},
			packetID:  2,
			audioData: []byte{},
			expectRes: &pipecatframe.Frame{
				Frame: &pipecatframe.Frame_Audio{
					Audio: &pipecatframe.AudioRawFrame{
						Id:          2,
						Audio:       []byte{},
						SampleRate:  defaultMediaSampleRate,
						NumChannels: defaultMediaNumChannel,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &pipecatframeHandler{}

			if err := h.SendAudio(tt.se, tt.packetID, tt.audioData); err != nil {
				t.Fatalf("SendAudio returned error: %v", err)
			}

			select {
			case got := <-tt.se.RunnerWebsocketChan:
				if got == nil {
					t.Fatal("got frame is nil")
				}

				if !reflect.DeepEqual(got, tt.expectRes) {
					t.Errorf("frame mismatch\nGot:  %+v\nWant: %+v", got, tt.expectRes)
				}
			default:
				t.Fatal("no frame pushed to RunnerWebsocketChan")
			}
		})
	}
}

func Test_SendRTVIText(t *testing.T) {
	tests := []struct {
		name           string
		se             *pipecatcall.Session
		ws             *websocket.Conn
		id             string
		text           string
		runImmediately bool
		audioResponse  bool
		expectRes      *pipecatframe.Frame
	}{
		{
			name: "simple RTVI text frame",
			se: &pipecatcall.Session{
				Ctx:                 context.Background(),
				RunnerWebsocketChan: make(chan *pipecatframe.Frame, 1),
			},
			ws:             &websocket.Conn{},
			id:             "123",
			text:           "Hello World",
			runImmediately: true,
			audioResponse:  true,
			expectRes: &pipecatframe.Frame{
				Frame: &pipecatframe.Frame_Message{
					Message: &pipecatframe.MessageFrame{
						Data: string([]byte(`{"id":"123","label":"rtvi-ai","type":"send-text","data":{"content":"Hello World","options":{"run_immediately":true,"audio_response":true}}}`)),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &pipecatframeHandler{}

			if err := h.SendRTVIText(tt.se, tt.id, tt.text, tt.runImmediately, tt.audioResponse); err != nil {
				t.Fatalf("SendRTVIText returned error: %v", err)
			}

			select {
			case got := <-tt.se.RunnerWebsocketChan:
				if got == nil {
					t.Fatal("got frame is nil")
				}

				if !reflect.DeepEqual(got, tt.expectRes) {
					t.Errorf("frame mismatch\nGot:  %+v\nWant: %+v", got, tt.expectRes)
				}

			default:
				t.Fatal("no frame pushed to RunnerWebsocketChan")
			}
		})
	}
}
