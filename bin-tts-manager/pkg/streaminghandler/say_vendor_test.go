package streaminghandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

var (
	testStreamingID = uuid.FromStringOrNil("7a1f0c52-0000-4000-8000-000000000001")
	testMessageID   = uuid.FromStringOrNil("7a1f0c52-0000-4000-8000-000000000002")
	otherMessageID  = uuid.FromStringOrNil("7a1f0c52-0000-4000-8000-000000000003")
)

func newSayTestHandler(t *testing.T, st *streaming.Streaming) (*streamingHandler, *Mockstreamer) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockStreamer := NewMockstreamer(mc)
	h := &streamingHandler{
		mapStreaming:      map[uuid.UUID]*streaming.Streaming{},
		elevenlabsHandler: mockStreamer,
	}
	if st != nil {
		h.mapStreaming[st.ID] = st
	}

	return h, mockStreamer
}

func newSayTestStreaming(vendor streaming.VendorName, config any) *streaming.Streaming {
	return &streaming.Streaming{
		Identity:     commonidentity.Identity{ID: testStreamingID},
		MessageID:    testMessageID,
		VendorName:   vendor,
		VendorConfig: config,
	}
}

func Test_SayStop(t *testing.T) {
	cfg := &ElevenlabsConfig{}

	tests := []struct {
		name string
		st   *streaming.Streaming

		// handlerErr is returned by the vendor SayStop. nil means "expect no call" unless expectCall is set.
		expectCall bool
		handlerErr error
		expectErr  bool
	}{
		{
			name:       "vendor session exists, the vendor is told to stop",
			st:         newSayTestStreaming(streaming.VendorNameElevenlabs, cfg),
			expectCall: true,
		},
		{
			name:       "vendor error is returned",
			st:         newSayTestStreaming(streaming.VendorNameElevenlabs, cfg),
			expectCall: true,
			handlerErr: errors.New("stop failed"),
			expectErr:  true,
		},
		{
			name: "no vendor selected yet",
			st:   newSayTestStreaming(streaming.VendorNameNone, nil),
		},
		{
			name: "vendor selected but not initialized",
			st:   newSayTestStreaming(streaming.VendorNameElevenlabs, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockStreamer := newSayTestHandler(t, tt.st)
			if tt.expectCall {
				mockStreamer.EXPECT().SayStop(cfg).Return(tt.handlerErr)
			}

			err := h.SayStop(context.Background(), testStreamingID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong error state. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.st.MessageID != uuid.Nil {
				t.Errorf("the current message must be cleared. got: %s", tt.st.MessageID)
			}
		})
	}

	t.Run("unknown streaming", func(t *testing.T) {
		h, _ := newSayTestHandler(t, nil)
		if err := h.SayStop(context.Background(), testStreamingID); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("no streamer is configured for the fallback provider", func(t *testing.T) {
		st := newSayTestStreaming(streaming.VendorNameElevenlabs, cfg)
		h, _ := newSayTestHandler(t, st)
		h.elevenlabsHandler = nil

		if err := h.SayStop(context.Background(), testStreamingID); err != nil {
			t.Fatalf("expected no error when there is nothing to stop, got: %v", err)
		}
	})
}

func Test_SayAdd_errors(t *testing.T) {
	cfg := &ElevenlabsConfig{}

	t.Run("unknown streaming", func(t *testing.T) {
		h, _ := newSayTestHandler(t, nil)
		if err := h.SayAdd(context.Background(), testStreamingID, testMessageID, "text"); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("message id mismatch does not reach the vendor", func(t *testing.T) {
		h, _ := newSayTestHandler(t, newSayTestStreaming(streaming.VendorNameElevenlabs, cfg))
		err := h.SayAdd(context.Background(), testStreamingID, otherMessageID, "text")
		if err == nil || !strings.Contains(err.Error(), "message ID mismatch") {
			t.Fatalf("expected a message ID mismatch error, got: %v", err)
		}
	})

	t.Run("vendor not initialized", func(t *testing.T) {
		h, _ := newSayTestHandler(t, newSayTestStreaming(streaming.VendorNameElevenlabs, nil))
		err := h.SayAdd(context.Background(), testStreamingID, testMessageID, "text")
		if err == nil || !strings.Contains(err.Error(), "vendor config is nil") {
			t.Fatalf("expected a nil vendor config error, got: %v", err)
		}
	})

	t.Run("no streamer is configured", func(t *testing.T) {
		h, _ := newSayTestHandler(t, newSayTestStreaming(streaming.VendorNameElevenlabs, cfg))
		h.elevenlabsHandler = nil
		err := h.SayAdd(context.Background(), testStreamingID, testMessageID, "text")
		if err == nil || !strings.Contains(err.Error(), "unsupported vendor") {
			t.Fatalf("expected an unsupported vendor error, got: %v", err)
		}
	})

	t.Run("vendor error is returned as is", func(t *testing.T) {
		h, mockStreamer := newSayTestHandler(t, newSayTestStreaming(streaming.VendorNameElevenlabs, cfg))
		wantErr := errors.New("send failed")
		mockStreamer.EXPECT().SayAdd(cfg, "text").Return(wantErr)

		if err := h.SayAdd(context.Background(), testStreamingID, testMessageID, "text"); !errors.Is(err, wantErr) {
			t.Fatalf("expected the vendor error, got: %v", err)
		}
	})
}

func Test_SayFlush(t *testing.T) {
	cfg := &ElevenlabsConfig{}

	t.Run("flush is delegated while the vendor lock is held", func(t *testing.T) {
		st := newSayTestStreaming(streaming.VendorNameElevenlabs, cfg)
		h, mockStreamer := newSayTestHandler(t, st)
		mockStreamer.EXPECT().SayFlush(cfg).DoAndReturn(func(any) error {
			if st.VendorLock.TryLock() {
				st.VendorLock.Unlock()
				t.Error("the vendor lock must be held during the flush")
			}
			return nil
		})

		if err := h.SayFlush(context.Background(), testStreamingID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !st.VendorLock.TryLock() {
			t.Fatal("the vendor lock must be released after the flush")
		}
		st.VendorLock.Unlock()
	})

	t.Run("vendor error is returned and the lock is released", func(t *testing.T) {
		st := newSayTestStreaming(streaming.VendorNameElevenlabs, cfg)
		h, mockStreamer := newSayTestHandler(t, st)
		wantErr := errors.New("flush failed")
		mockStreamer.EXPECT().SayFlush(cfg).Return(wantErr)

		if err := h.SayFlush(context.Background(), testStreamingID); !errors.Is(err, wantErr) {
			t.Fatalf("expected the vendor error, got: %v", err)
		}
		if !st.VendorLock.TryLock() {
			t.Fatal("the vendor lock must be released on error")
		}
		st.VendorLock.Unlock()
	})

	t.Run("unknown streaming", func(t *testing.T) {
		h, _ := newSayTestHandler(t, nil)
		if err := h.SayFlush(context.Background(), testStreamingID); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("no streamer is configured", func(t *testing.T) {
		st := newSayTestStreaming(streaming.VendorNameElevenlabs, cfg)
		h, _ := newSayTestHandler(t, st)
		h.elevenlabsHandler = nil

		err := h.SayFlush(context.Background(), testStreamingID)
		if err == nil || !strings.Contains(err.Error(), "unsupported vendor") {
			t.Fatalf("expected an unsupported vendor error, got: %v", err)
		}
		if !st.VendorLock.TryLock() {
			t.Fatal("the vendor lock must be released on error")
		}
		st.VendorLock.Unlock()
	})
}

func Test_SayFinish(t *testing.T) {
	cfg := &ElevenlabsConfig{}

	t.Run("finish is delegated and the streaming is returned", func(t *testing.T) {
		st := newSayTestStreaming(streaming.VendorNameElevenlabs, cfg)
		h, mockStreamer := newSayTestHandler(t, st)
		mockStreamer.EXPECT().SayFinish(cfg).Return(nil)

		res, err := h.SayFinish(context.Background(), testStreamingID, testMessageID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != st {
			t.Errorf("expected the stored streaming to be returned, got: %+v", res)
		}
	})

	t.Run("vendor error is wrapped and no streaming is returned", func(t *testing.T) {
		h, mockStreamer := newSayTestHandler(t, newSayTestStreaming(streaming.VendorNameElevenlabs, cfg))
		wantErr := errors.New("finish failed")
		mockStreamer.EXPECT().SayFinish(cfg).Return(wantErr)

		res, err := h.SayFinish(context.Background(), testStreamingID, testMessageID)
		if !errors.Is(err, wantErr) || res != nil {
			t.Fatalf("expected a wrapped vendor error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("unknown streaming", func(t *testing.T) {
		h, _ := newSayTestHandler(t, nil)
		if _, err := h.SayFinish(context.Background(), testStreamingID, testMessageID); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("message id mismatch does not reach the vendor", func(t *testing.T) {
		h, _ := newSayTestHandler(t, newSayTestStreaming(streaming.VendorNameElevenlabs, cfg))
		_, err := h.SayFinish(context.Background(), testStreamingID, otherMessageID)
		if err == nil || !strings.Contains(err.Error(), "message ID mismatch") {
			t.Fatalf("expected a message ID mismatch error, got: %v", err)
		}
	})

	t.Run("vendor not initialized", func(t *testing.T) {
		h, _ := newSayTestHandler(t, newSayTestStreaming(streaming.VendorNameElevenlabs, nil))
		_, err := h.SayFinish(context.Background(), testStreamingID, testMessageID)
		if err == nil || !strings.Contains(err.Error(), "vendor config is nil") {
			t.Fatalf("expected a nil vendor config error, got: %v", err)
		}
	})

	t.Run("no streamer is configured", func(t *testing.T) {
		h, _ := newSayTestHandler(t, newSayTestStreaming(streaming.VendorNameElevenlabs, cfg))
		h.elevenlabsHandler = nil
		_, err := h.SayFinish(context.Background(), testStreamingID, testMessageID)
		if err == nil || !strings.Contains(err.Error(), "unsupported vendor") {
			t.Fatalf("expected an unsupported vendor error, got: %v", err)
		}
	})
}

func Test_SayInit_errors(t *testing.T) {
	t.Run("unknown streaming", func(t *testing.T) {
		h, _ := newSayTestHandler(t, nil)
		res, err := h.SayInit(context.Background(), testStreamingID, testMessageID)
		if err == nil || res != nil {
			t.Fatalf("expected an error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("streamer initialization failure is returned", func(t *testing.T) {
		st := newSayTestStreaming(streaming.VendorNameNone, nil)
		st.Provider = string(streaming.VendorNameGCP)
		h, _ := newSayTestHandler(t, st)

		mc := gomock.NewController(t)
		defer mc.Finish()
		mockGCP := NewMockstreamer(mc)
		h.gcpHandler = mockGCP
		wantErr := errors.New("init failed")
		mockGCP.EXPECT().Init(gomock.Any(), st).Return(nil, wantErr)

		res, err := h.SayInit(context.Background(), testStreamingID, testMessageID)
		if !errors.Is(err, wantErr) || res != nil {
			t.Fatalf("expected the init error and nil result, got: %v, %v", res, err)
		}
		if st.MessageID != testMessageID {
			t.Errorf("the message id is recorded before the streamer starts. got: %s", st.MessageID)
		}
	})
}

func Test_getStreamerByProvider(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	eleven := NewMockstreamer(mc)
	gcp := NewMockstreamer(mc)
	aws := NewMockstreamer(mc)
	h := &streamingHandler{elevenlabsHandler: eleven, gcpHandler: gcp, awsHandler: aws}

	tests := []struct {
		provider   string
		expect     streamer
		expectName streaming.VendorName
	}{
		{"elevenlabs", eleven, streaming.VendorNameElevenlabs},
		{"gcp", gcp, streaming.VendorNameGCP},
		{"aws", aws, streaming.VendorNameAWS},
		{"", eleven, streaming.VendorNameElevenlabs},
		{"unknown-provider", eleven, streaming.VendorNameElevenlabs},
	}

	for _, tt := range tests {
		t.Run("provider "+tt.provider, func(t *testing.T) {
			got, name := h.getStreamerByProvider(tt.provider)
			if got != tt.expect {
				t.Errorf("Wrong streamer for %q", tt.provider)
			}
			if name != tt.expectName {
				t.Errorf("Wrong vendor name. expected: %s, got: %s", tt.expectName, name)
			}
		})
	}
}

func Test_runStreamer(t *testing.T) {
	t.Run("provider without a streamer is rejected", func(t *testing.T) {
		st := &streaming.Streaming{Provider: "unknown-provider"}
		h := &streamingHandler{}

		err := h.runStreamer(context.Background(), st)
		if err == nil || !strings.Contains(err.Error(), "unsupported or unconfigured provider") {
			t.Fatalf("expected an unsupported provider error, got: %v", err)
		}
		if st.VendorName != streaming.VendorNameNone || st.VendorConfig != nil {
			t.Errorf("no vendor info may be recorded. got: %q %v", st.VendorName, st.VendorConfig)
		}
	})

	t.Run("init failure is wrapped and nothing is recorded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		st := &streaming.Streaming{Provider: "aws"}
		mockAWS := NewMockstreamer(mc)
		h := &streamingHandler{awsHandler: mockAWS}
		wantErr := errors.New("init failed")
		mockAWS.EXPECT().Init(gomock.Any(), st).Return(nil, wantErr)

		err := h.runStreamer(context.Background(), st)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected the init error, got: %v", err)
		}
		if st.VendorName != streaming.VendorNameNone || st.VendorConfig != nil {
			t.Errorf("no vendor info may be recorded. got: %q %v", st.VendorName, st.VendorConfig)
		}
	})

	for _, runErr := range []error{nil, errors.New("run failed")} {
		caseName := "vendor info is recorded while running and cleared after Run returns"
		if runErr != nil {
			caseName += " even when Run fails"
		}

		t.Run(caseName, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			st := &streaming.Streaming{Provider: "gcp"}
			cfg := &GCPConfig{}
			mockGCP := NewMockstreamer(mc)
			h := &streamingHandler{gcpHandler: mockGCP}

			release := make(chan struct{})
			runReturned := make(chan struct{})
			mockGCP.EXPECT().Init(gomock.Any(), st).Return(cfg, nil)
			mockGCP.EXPECT().Run(cfg).DoAndReturn(func(any) error {
				<-release
				defer close(runReturned)
				return runErr
			})

			if err := h.runStreamer(context.Background(), st); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			st.VendorLock.Lock()
			gotName, gotConfig := st.VendorName, st.VendorConfig
			st.VendorLock.Unlock()
			if gotName != streaming.VendorNameGCP || gotConfig != any(cfg) {
				t.Fatalf("vendor info must be recorded before Run completes. got: %q %v", gotName, gotConfig)
			}

			close(release)
			<-runReturned

			deadline := time.Now().Add(5 * time.Second)
			for {
				st.VendorLock.Lock()
				cleared := st.VendorName == streaming.VendorNameNone && st.VendorConfig == nil
				st.VendorLock.Unlock()
				if cleared {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("vendor info was not cleared after Run returned")
				}
				runtime.Gosched()
			}
		})
	}
}

func Test_streamingHandler_Run(t *testing.T) {
	h := &streamingHandler{}
	if err := h.Run(); err != nil {
		t.Fatalf("Run is a no-op and must not fail, got: %v", err)
	}
}

func Test_UpdateConnAst(t *testing.T) {
	client, _ := newLocalWSPair(t)
	st := &streaming.Streaming{Identity: commonidentity.Identity{ID: testStreamingID}}
	h := &streamingHandler{mapStreaming: map[uuid.UUID]*streaming.Streaming{testStreamingID: st}}

	res, err := h.UpdateConnAst(testStreamingID, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != st || st.ConnAst != client {
		t.Errorf("the connection must be stored on the tracked streaming")
	}

	if _, errMissing := h.UpdateConnAst(otherMessageID, client); errMissing == nil {
		t.Error("expected an error for an unknown streaming")
	}
}

func Test_SetVendorInfo(t *testing.T) {
	st := &streaming.Streaming{}
	h := &streamingHandler{}
	cfg := &AWSConfig{}

	h.SetVendorInfo(st, streaming.VendorNameAWS, cfg)
	if st.VendorName != streaming.VendorNameAWS || st.VendorConfig != any(cfg) {
		t.Errorf("vendor info was not stored. got: %q %v", st.VendorName, st.VendorConfig)
	}

	h.SetVendorInfo(st, streaming.VendorNameNone, nil)
	if st.VendorName != streaming.VendorNameNone || st.VendorConfig != nil {
		t.Errorf("vendor info was not cleared. got: %q %v", st.VendorName, st.VendorConfig)
	}
}

func Test_Stop_errors(t *testing.T) {
	t.Run("unknown streaming", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &streamingHandler{
			requestHandler: requesthandler.NewMockRequestHandler(mc),
			mapStreaming:   map[uuid.UUID]*streaming.Streaming{},
		}
		res, err := h.Stop(context.Background(), testStreamingID)
		if err == nil || res != nil {
			t.Fatalf("expected an error and nil result, got: %v, %v", res, err)
		}
	})

	t.Run("external media stop failure keeps the connection and returns the streaming", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		client, server := newLocalWSPair(t)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		st := &streaming.Streaming{
			Identity:   commonidentity.Identity{ID: testStreamingID},
			VendorName: streaming.VendorNameGCP,
			ConnAst:    client,
		}
		h := &streamingHandler{
			requestHandler: mockReq,
			mapStreaming:   map[uuid.UUID]*streaming.Streaming{testStreamingID: st},
		}
		mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), testStreamingID).Return(nil, errors.New("stop failed"))

		res, err := h.Stop(context.Background(), testStreamingID)
		if err == nil || res != st {
			t.Fatalf("expected an error together with the streaming, got: %v, %v", res, err)
		}

		// the connection was not closed, so a message can still be sent through it
		if errWrite := client.WriteMessage(websocket.TextMessage, []byte("still open")); errWrite != nil {
			t.Errorf("the Asterisk connection must stay open on failure: %v", errWrite)
		}
		_ = server.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, data, errRead := server.ReadMessage(); errRead != nil || string(data) != "still open" {
			t.Errorf("the message did not arrive. err: %v, data: %q", errRead, string(data))
		}
	})

	t.Run("successful stop closes the Asterisk connection", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		client, _ := newLocalWSPair(t)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		st := &streaming.Streaming{Identity: commonidentity.Identity{ID: testStreamingID}, ConnAst: client}
		h := &streamingHandler{
			requestHandler: mockReq,
			mapStreaming:   map[uuid.UUID]*streaming.Streaming{testStreamingID: st},
		}
		mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), testStreamingID).
			Return(&cmexternalmedia.ExternalMedia{ID: testStreamingID}, nil)

		if _, err := h.Stop(context.Background(), testStreamingID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if errWrite := client.WriteMessage(websocket.TextMessage, []byte("late")); errWrite == nil {
			t.Error("the Asterisk connection must be closed after a successful stop")
		}
	})
}

// newMediaStartServer starts a local WebSocket server that sends the given
// first message to every client and returns its ws:// URL.
func newMediaStartServer(t *testing.T, firstMessageType int, first string) string {
	t.Helper()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.WriteMessage(firstMessageType, []byte(first))
		go func() {
			// keep the connection open until the client goes away
			for {
				if _, _, errRead := conn.ReadMessage(); errRead != nil {
					_ = conn.Close()
					return
				}
			}
		}()
	}))
	t.Cleanup(srv.Close)

	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func Test_startExternalMedia(t *testing.T) {
	newHandler := func(t *testing.T, st *streaming.Streaming) (*streamingHandler, *requesthandler.MockRequestHandler) {
		mc := gomock.NewController(t)
		t.Cleanup(mc.Finish)

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &streamingHandler{
			requestHandler: mockReq,
			notifyHandler:  notifyhandler.NewMockNotifyHandler(mc),
			mapStreaming:   map[uuid.UUID]*streaming.Streaming{st.ID: st},
		}
		return h, mockReq
	}

	t.Run("connects to the media endpoint and tracks the connection", func(t *testing.T) {
		st := &streaming.Streaming{Identity: commonidentity.Identity{ID: testStreamingID}, Provider: "aws"}
		h, mockReq := newHandler(t, st)
		uri := newMediaStartServer(t, websocket.TextMessage, "MEDIA_START")
		mockReq.EXPECT().CallV1ExternalMediaStart(
			gomock.Any(), testStreamingID, gomock.Any(), gomock.Any(), "INCOMING",
			defaultEncapsulation, defaultTransport, "", defaultConnectionType, formatSlin,
			gomock.Any(), gomock.Any(),
		).Return(&cmexternalmedia.ExternalMedia{ID: testStreamingID, MediaURI: uri}, nil)

		if err := h.startExternalMedia(context.Background(), st); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if st.ConnAst == nil || st.ConnAstDone == nil {
			t.Fatal("the connection and the done channel must be set")
		}

		// closing the Asterisk connection must signal the done channel
		_ = st.ConnAst.Close()
		select {
		case <-st.ConnAstDone:
		case <-time.After(5 * time.Second):
			t.Fatal("ConnAstDone was not closed after the connection was closed")
		}
	})

	t.Run("external media start failure is returned", func(t *testing.T) {
		st := &streaming.Streaming{Identity: commonidentity.Identity{ID: testStreamingID}}
		h, mockReq := newHandler(t, st)
		wantErr := errors.New("start failed")
		mockReq.EXPECT().CallV1ExternalMediaStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(nil, wantErr)

		if err := h.startExternalMedia(context.Background(), st); !errors.Is(err, wantErr) {
			t.Fatalf("expected the start error, got: %v", err)
		}
		if st.ConnAst != nil {
			t.Error("no connection may be stored")
		}
	})

	t.Run("a non-text first message is rejected and the media is stopped", func(t *testing.T) {
		st := &streaming.Streaming{Identity: commonidentity.Identity{ID: testStreamingID}}
		h, mockReq := newHandler(t, st)
		uri := newMediaStartServer(t, websocket.BinaryMessage, "not text")
		mockReq.EXPECT().CallV1ExternalMediaStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(&cmexternalmedia.ExternalMedia{ID: testStreamingID, MediaURI: uri}, nil)
		mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), testStreamingID).
			Return(nil, errors.New("stop failed too"))

		if err := h.startExternalMedia(context.Background(), st); err == nil {
			t.Fatal("expected an error for a binary MEDIA_START message")
		}
		if st.ConnAst != nil {
			t.Error("no connection may be stored")
		}
	})

	t.Run("a streaming that vanished during the handshake closes the new connection", func(t *testing.T) {
		st := &streaming.Streaming{Identity: commonidentity.Identity{ID: testStreamingID}}
		h, mockReq := newHandler(t, st)
		delete(h.mapStreaming, testStreamingID)
		uri := newMediaStartServer(t, websocket.TextMessage, "MEDIA_START")
		mockReq.EXPECT().CallV1ExternalMediaStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(&cmexternalmedia.ExternalMedia{ID: testStreamingID, MediaURI: uri}, nil)

		if err := h.startExternalMedia(context.Background(), st); err == nil {
			t.Fatal("expected an error when the streaming is not tracked")
		}
	})
}
