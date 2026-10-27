package streaminghandler

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// setMissingGCPCredentials points the application default credentials at a
// file that does not exist, so client creation fails locally.
func setMissingGCPCredentials(t *testing.T) {
	t.Helper()

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing.json"))
}

// setSyntheticGCPCredentials points the application default credentials at a
// synthetic authorized-user file. The values are never sent anywhere because
// the tests that use it also cancel the context before any stream is opened.
func setSyntheticGCPCredentials(t *testing.T) {
	t.Helper()

	cred := map[string]string{"type": "authorized_user"}
	cred["client_id"] = "fake-client-id"
	cred["client_secret"] = "fake-client-secret"
	cred["refresh_token"] = "fake-refresh-token"

	data, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("could not marshal the credential: %v", err)
	}

	path := filepath.Join(t.TempDir(), "adc.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("could not write the credential: %v", err)
	}

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", path)
}

func newGCPInitStreaming() *streaming.Streaming {
	st := &streaming.Streaming{
		// an explicit voice avoids the activeflow variable lookup
		VoiceID:  "en-US-Chirp3-HD-Charon",
		Language: "en-US",
	}
	st.ID = uuid.FromStringOrNil("6c2d1e00-0000-4000-8000-0000000000b1")

	return st
}

func Test_gcpHandler_Init_failures(t *testing.T) {
	tests := []struct {
		name string

		setCredentials func(t *testing.T)
		cancelContext  bool

		expectErrs []string
	}{
		{
			name: "the client cannot be created",

			setCredentials: setMissingGCPCredentials,

			expectErrs: []string{"failed to initialize GCP StreamingSynthesize", "could not create GCP TTS client"},
		},
		{
			name: "the stream cannot be opened",

			setCredentials: setSyntheticGCPCredentials,
			cancelContext:  true,

			expectErrs: []string{"failed to initialize GCP StreamingSynthesize", "could not start StreamingSynthesize"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			tt.setCredentials(t)

			// no event is expected: the mock fails the test if one is published
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &gcpHandler{notifyHandler: mockNotify}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancelContext {
				cancel()
			}

			res, err := h.Init(ctx, newGCPInitStreaming())
			if res != nil || err == nil {
				t.Fatalf("Wrong match. expect: error, got: %v, %v", res, err)
			}
			for _, expect := range tt.expectErrs {
				if !strings.Contains(err.Error(), expect) {
					t.Errorf("Wrong error. expect to contain: %q, got: %v", expect, err)
				}
			}
		})
	}
}

func newGCPReconnectConfig(stream *fakeGCPStream) *GCPConfig {
	cf := newTestGCPConfig(stream)
	cf.VoiceID = "en-US-Chirp3-HD-Charon"
	cf.LangCode = "en-US"
	// the previous runProcess has already exited
	close(cf.processDone)

	return cf
}

func Test_gcpHandler_waitAndReconnectLocked_connectFailure(t *testing.T) {
	setMissingGCPCredentials(t)

	cf := newGCPReconnectConfig(nil)
	t.Cleanup(cf.Cancel)
	h := &gcpHandler{}

	cf.muStream.Lock()
	err := h.waitAndReconnectLocked(cf)
	cf.muStream.Unlock()

	if err == nil || !strings.Contains(err.Error(), "failed to reconnect GCP stream") {
		t.Fatalf("Wrong match. expect: reconnect error, got: %v", err)
	}
	if cf.Stream != nil || cf.Client != nil {
		t.Errorf("no stream or client may remain after the failure. stream: %v, client: %v", cf.Stream, cf.Client)
	}
	// the lock was handed back to the caller and is free again
	if !cf.muStream.TryLock() {
		t.Fatal("the stream lock must be released after the failure")
	}
	cf.muStream.Unlock()
}

func Test_gcpHandler_waitAndReconnectLocked_alreadyReconnected(t *testing.T) {
	stream := &fakeGCPStream{}
	cf := newGCPReconnectConfig(stream)
	t.Cleanup(cf.Cancel)
	h := &gcpHandler{}

	doneBefore := cf.processDone

	cf.muStream.Lock()
	err := h.waitAndReconnectLocked(cf)
	cf.muStream.Unlock()

	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if cf.Stream != stream {
		t.Error("the existing stream must be kept")
	}
	if cf.processDone != doneBefore {
		t.Error("no new session may be started when the stream is already there")
	}
	if stream.closeSendCalls != 0 {
		t.Errorf("the existing stream must not be closed. got: %d", stream.closeSendCalls)
	}
}

func Test_gcpHandler_SayAdd_reconnectFailure(t *testing.T) {
	setMissingGCPCredentials(t)

	cf := newGCPReconnectConfig(nil)
	t.Cleanup(cf.Cancel)
	h := &gcpHandler{}

	err := h.SayAdd(cf, "text")
	if err == nil || !strings.Contains(err.Error(), "failed to reconnect GCP stream") {
		t.Fatalf("Wrong match. expect: reconnect error, got: %v", err)
	}
	if cf.Message.TotalCount != 0 || cf.Message.TotalMessage != "" {
		t.Errorf("a failed SayAdd must not count the message. got: %+v", cf.Message)
	}
	if !cf.muStream.TryLock() {
		t.Fatal("the stream lock must be released after the failure")
	}
	cf.muStream.Unlock()
}

func Test_closeStreamLocked_closesClient(t *testing.T) {
	//nolint:staticcheck // same texttospeech library limitation as gcp.go
	client, err := texttospeech.NewClient(
		context.Background(),
		option.WithEndpoint("127.0.0.1:1"),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("could not create an offline client: %v", err)
	}

	stream := &fakeGCPStream{}
	cf := newTestGCPConfig(stream)
	t.Cleanup(cf.Cancel)
	cf.Client = client

	cf.muStream.Lock()
	closeStreamLocked(cf)
	cf.muStream.Unlock()

	if cf.Stream != nil || cf.Client != nil {
		t.Errorf("the stream and the client must be released. stream: %v, client: %v", cf.Stream, cf.Client)
	}
	if stream.closeSendCalls != 1 {
		t.Errorf("Wrong CloseSend count. expect: 1, got: %d", stream.closeSendCalls)
	}

	// a second call has nothing left to close
	cf.muStream.Lock()
	closeStreamLocked(cf)
	cf.muStream.Unlock()
	if stream.closeSendCalls != 1 {
		t.Errorf("a released stream must not be closed again. got: %d", stream.closeSendCalls)
	}
}

// runKeepaliveAndWait runs runKeepalive and waits for it to end on its own.
func runKeepaliveAndWait(t *testing.T, h *gcpHandler, cf *GCPConfig) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		h.runKeepalive(cf)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * gcpKeepaliveInterval):
		cf.StreamCancel()
		t.Fatal("runKeepalive did not end")
	}
}

func Test_gcpHandler_runKeepalive_endsWhenStreamIsGone(t *testing.T) {
	t.Parallel()

	cf := newTestGCPConfig(nil)
	t.Cleanup(cf.Cancel)

	runKeepaliveAndWait(t, &gcpHandler{}, cf)

	if cf.Stream != nil {
		t.Error("Wrong match. expect: no stream")
	}
}

func Test_gcpHandler_runKeepalive_sendFailureClosesStream(t *testing.T) {
	t.Parallel()

	stream := &fakeGCPStream{sendErr: errors.New("stream broken")}
	cf := newTestGCPConfig(stream)
	t.Cleanup(cf.Cancel)
	// the last send is long ago, so the keepalive must be attempted
	cf.lastSendTime = time.Now().Add(-time.Hour)

	runKeepaliveAndWait(t, &gcpHandler{}, cf)

	if cf.Stream != nil {
		t.Error("a failed keepalive must release the stream so that the next SayAdd reconnects")
	}
	if stream.closeSendCalls != 1 {
		t.Errorf("Wrong CloseSend count. expect: 1, got: %d", stream.closeSendCalls)
	}
}

func Test_gcpHandler_runProcess_stopsWhenAsteriskWriteFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	stream := newFakeGCPRecvStream()
	cf := newTestGCPConfig(&stream.fakeGCPStream)
	cf.Stream = stream

	// the Asterisk side is gone, so writing the received audio must fail
	astClient, _ := newLocalWSPair(t)
	_ = astClient.Close()
	cf.ConnAst = astClient

	stream.recvCh <- gcpRecvResult{resp: &texttospeechpb.StreamingSynthesizeResponse{AudioContent: []byte{1, 2, 3}}}

	h := &gcpHandler{notifyHandler: mockNotify}

	done := make(chan struct{})
	go func() {
		h.runProcess(cf)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runProcess did not stop after the Asterisk write failed")
	}
}
