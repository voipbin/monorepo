package streaminghandler

import (
	"context"
	"testing"
	"time"

	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"go.uber.org/mock/gomock"
)

// fakeGCPRecvStream extends the Send/CloseSend tracking of fakeGCPStream
// (defined in gcp_say_test.go) with a channel-driven Recv, so a test can
// control exactly when a Recv call returns and with what result. Recv
// reports it was entered on enteredCh (non-blocking) before waiting on
// recvCh, so a test can deterministically wait for the goroutine to be
// inside Recv before acting (e.g. cancelling a context).
type fakeGCPRecvStream struct {
	fakeGCPStream

	recvCh    chan gcpRecvResult
	enteredCh chan struct{}
}

type gcpRecvResult struct {
	resp *texttospeechpb.StreamingSynthesizeResponse
	err  error
}

func newFakeGCPRecvStream() *fakeGCPRecvStream {
	return &fakeGCPRecvStream{
		recvCh:    make(chan gcpRecvResult, 2),
		enteredCh: make(chan struct{}, 8),
	}
}

func (f *fakeGCPRecvStream) Recv() (*texttospeechpb.StreamingSynthesizeResponse, error) {
	select {
	case f.enteredCh <- struct{}{}:
	default:
	}
	r := <-f.recvCh
	return r.resp, r.err
}

func (f *fakeGCPRecvStream) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-f.enteredCh:
	case <-time.After(3 * time.Second):
		t.Fatal("Recv was never entered")
	}
}

// Test_gcpHandler_runProcess_writesNonEmptyAudioChunk verifies a successful
// Recv with non-empty audio content is written to the Asterisk WebSocket,
// and an empty-content response ("continue") does not stop the loop.
func Test_gcpHandler_runProcess_writesNonEmptyAudioChunk(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).Times(2) // PlayStarted + PlayFinished

	client, server := newLocalWSPair(t)

	stream := newFakeGCPRecvStream()
	cf := newTestGCPConfig(&stream.fakeGCPStream)
	cf.Stream = stream
	cf.ConnAst = client

	h := &gcpHandler{notifyHandler: mockNotify}

	done := make(chan struct{})
	go func() {
		h.runProcess(cf)
		close(done)
	}()

	stream.waitEntered(t)
	// An empty-content response must not stop the loop (the "continue" branch).
	stream.recvCh <- gcpRecvResult{resp: &texttospeechpb.StreamingSynthesizeResponse{}}
	stream.waitEntered(t)
	stream.recvCh <- gcpRecvResult{resp: &texttospeechpb.StreamingSynthesizeResponse{AudioContent: []byte("audio-bytes")}}

	_, received, errRead := server.ReadMessage()
	if errRead != nil {
		t.Fatalf("Wrong match. expected: a binary frame, got error: %v", errRead)
	}
	if string(received) != "audio-bytes" {
		t.Errorf("Wrong match. expected: audio-bytes, got: %s", string(received))
	}

	// End the loop deterministically via the per-stream cancellation path.
	stream.waitEntered(t)
	cf.StreamCancel()
	stream.recvCh <- gcpRecvResult{err: context.Canceled}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runProcess did not return")
	}
}

// Test_gcpHandler_runProcess_endsOnServerDisconnect verifies that when Recv
// fails while the per-stream context is still active, runProcess nils out
// the stream/client via closeStreamLocked (so the next SayAdd reconnects)
// and calls CloseSend exactly once.
func Test_gcpHandler_runProcess_endsOnServerDisconnect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	stream := newFakeGCPRecvStream()
	cf := newTestGCPConfig(&stream.fakeGCPStream)
	cf.Stream = stream

	h := &gcpHandler{notifyHandler: mockNotify}

	done := make(chan struct{})
	go func() {
		h.runProcess(cf)
		close(done)
	}()

	stream.waitEntered(t)
	stream.recvCh <- gcpRecvResult{err: context.DeadlineExceeded} // server-side timeout, streamCtx still active

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runProcess did not return")
	}

	cf.muStream.Lock()
	gotStream := cf.Stream
	gotClient := cf.Client
	cf.muStream.Unlock()

	if gotStream != nil {
		t.Error("Wrong match. expected: Stream nil after server disconnect, got: non-nil")
	}
	if gotClient != nil {
		t.Error("Wrong match. expected: Client nil after server disconnect, got: non-nil")
	}
	if stream.closeSendCalls != 1 {
		t.Errorf("Wrong match. expected: CloseSend called once, got: %d", stream.closeSendCalls)
	}

	select {
	case <-cf.processDone:
	default:
		t.Error("Wrong match. expected: processDone closed")
	}
}

// Test_gcpHandler_runProcess_endsOnStreamCancellation verifies that when the
// per-stream context is cancelled while Recv is in progress, runProcess
// returns via the "context cancelled" branch without touching
// Stream/Client/CloseSend (closeStreamLocked is NOT called): a later
// reconnect attempt may still find a live stream if one raced in.
func Test_gcpHandler_runProcess_endsOnStreamCancellation(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	stream := newFakeGCPRecvStream()
	cf := newTestGCPConfig(&stream.fakeGCPStream)
	cf.Stream = stream

	h := &gcpHandler{notifyHandler: mockNotify}

	done := make(chan struct{})
	go func() {
		h.runProcess(cf)
		close(done)
	}()

	stream.waitEntered(t)
	cf.StreamCancel()
	// Unblock the in-progress Recv call with any error, now that the
	// per-stream context is already cancelled.
	stream.recvCh <- gcpRecvResult{err: context.Canceled}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runProcess did not return")
	}

	cf.muStream.Lock()
	gotStream := cf.Stream
	cf.muStream.Unlock()

	if gotStream == nil {
		t.Error("Wrong match. expected: Stream left untouched (non-nil), got: nil")
	}
	if stream.closeSendCalls != 0 {
		t.Errorf("Wrong match. expected: CloseSend not called, got: %d calls", stream.closeSendCalls)
	}
}
