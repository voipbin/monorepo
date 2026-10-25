package streaminghandler

import (
	"context"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// fakeStreamingSpeech is an in-memory speech service for the streaming recognize call. For every
// stream it publishes the first request (the recognition config), answers with the configured
// responses and then ends the stream.
type fakeStreamingSpeech struct {
	speechpb.UnimplementedSpeechServer

	responses []*speechpb.StreamingRecognizeResponse
	configs   chan *speechpb.StreamingRecognizeRequest
}

func (f *fakeStreamingSpeech) StreamingRecognize(stream speechpb.Speech_StreamingRecognizeServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	f.configs <- first

	for _, resp := range f.responses {
		if errSend := stream.Send(resp); errSend != nil {
			return errSend
		}
	}

	return nil
}

// newFakeStreamingSpeech returns a client connected to the fake. When failStream is set, creating a
// stream is rejected with a code that the client does not retry. streamCalls counts the attempts.
func newFakeStreamingSpeech(t *testing.T, responses []*speechpb.StreamingRecognizeResponse, failStream bool) (*speech.Client, *fakeStreamingSpeech, *atomic.Int32) {
	t.Helper()

	fake := &fakeStreamingSpeech{
		responses: responses,
		configs:   make(chan *speechpb.StreamingRecognizeRequest, 8),
	}
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	speechpb.RegisterSpeechServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()

	streamCalls := &atomic.Int32{}
	counter := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		streamCalls.Add(1)
		if failStream {
			return nil, grpcstatus.Error(codes.PermissionDenied, "rejected")
		}
		return streamer(ctx, desc, cc, method, opts...)
	}

	client, err := speech.NewClient(
		context.Background(),
		option.WithGRPCDialOption(grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() })),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithGRPCDialOption(grpc.WithChainStreamInterceptor(counter)),
		option.WithEndpoint("passthrough:///bufnet"),
		option.WithoutAuthentication(),
	)
	if err != nil {
		srv.Stop()
		t.Fatalf("could not create the speech client. err: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		srv.Stop()
	})

	return client, fake, streamCalls
}

// recordingStreamClient records the audio sent to the speech service.
type recordingStreamClient struct {
	mockGCPStreamClient

	mu      sync.Mutex
	audio   [][]byte
	sendErr error
}

func (r *recordingStreamClient) Send(req *speechpb.StreamingRecognizeRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.audio = append(r.audio, req.GetAudioContent())
	return r.sendErr
}

func (r *recordingStreamClient) sent() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([][]byte{}, r.audio...)
}

// connectTestMedia connects a websocket client to a loopback media server running the scenario.
func connectTestMedia(t *testing.T, scenario func(conn net.Conn)) *websocket.Conn {
	t.Helper()

	url := newTestMediaServer(t, func(conn net.Conn) {
		serverWriteFrame(conn, websocket.TextMessage, []byte("MEDIA_START channel:test"))
		scenario(conn)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := websocketConnect(ctx, url)
	if err != nil {
		t.Fatalf("could not connect the media server. err: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

// waitMediaReaderGone waits until no goroutine is left in gcpProcessMedia.
func waitMediaReaderGone(t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		if !strings.Contains(string(buf[:n]), "(*streamingHandler).gcpProcessMedia") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Wrong match. gcpProcessMedia is still running")
}

func newFakeSTTStreaming(language string, direction transcript.Direction) *streaming.Streaming {
	return &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("7d2a3c10-0000-11ee-8b11-000000000001"),
			CustomerID: uuid.FromStringOrNil("7d2a3c10-0000-11ee-8b11-000000000002"),
		},
		TranscribeID: uuid.FromStringOrNil("7d2a3c10-0000-11ee-8b11-000000000003"),
		Language:     language,
		Direction:    direction,
	}
}

func Test_gcpInit_sendsRecognitionConfig(t *testing.T) {
	client, fake, streamCalls := newFakeStreamingSpeech(t, nil, false)
	h := &streamingHandler{gcpClient: client}
	st := newFakeSTTStreaming("ko-KR", transcript.DirectionIn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, err := h.gcpInit(ctx, st)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect a stream client, got: nil")
	}

	var first *speechpb.StreamingRecognizeRequest
	select {
	case first = <-fake.configs:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. the service did not receive the first request")
	}

	cfg := first.GetStreamingConfig()
	if cfg == nil {
		t.Fatalf("Wrong match. expect the first request to carry the streaming config, got: %v", first)
	}
	if cfg.GetConfig().GetLanguageCode() != "ko-KR" ||
		cfg.GetConfig().GetEncoding() != defaultGCPEncoding ||
		cfg.GetConfig().GetSampleRateHertz() != int32(defaultGCPSampleRate) ||
		cfg.GetConfig().GetAudioChannelCount() != int32(defaultGCPAudioChannelCount) ||
		!cfg.GetConfig().GetEnableAutomaticPunctuation() ||
		!cfg.GetConfig().GetUseEnhanced() ||
		!cfg.GetInterimResults() {
		t.Errorf("Wrong match. unexpected streaming config: %v", cfg)
	}
	if streamCalls.Load() != 1 {
		t.Errorf("Wrong match. expect: 1 stream, got: %d", streamCalls.Load())
	}
}

func Test_gcpInit_streamCreateFails(t *testing.T) {
	client, _, streamCalls := newFakeStreamingSpeech(t, nil, true)
	h := &streamingHandler{gcpClient: client}

	res, err := h.gcpInit(context.Background(), newFakeSTTStreaming("en-US", transcript.DirectionBoth))
	if err == nil {
		t.Fatalf("Wrong match. expect: err, got: ok (%v)", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if streamCalls.Load() != 1 {
		t.Errorf("Wrong match. expect: 1 attempt, got: %d", streamCalls.Load())
	}
}

func Test_gcpProcessMedia_forwardsOnlyAudio(t *testing.T) {
	conn := connectTestMedia(t, func(conn net.Conn) {
		serverWriteFrame(conn, websocket.TextMessage, []byte("status"))
		serverWriteFrame(conn, websocket.BinaryMessage, []byte{})
		serverWriteFrame(conn, websocket.BinaryMessage, []byte{0x01, 0x02, 0x03})
		serverWriteFrame(conn, websocket.BinaryMessage, []byte{0x04, 0x05})
		serverWriteFrame(conn, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
		serverWaitClosed(conn)
	})

	h := &streamingHandler{}
	st := newFakeSTTStreaming("en-US", transcript.DirectionIn)
	st.ConnAst = conn
	streamClient := &recordingStreamClient{}

	ctx, cancel := context.WithCancel(context.Background())
	h.gcpProcessMedia(ctx, cancel, st, streamClient)

	sent := streamClient.sent()
	if len(sent) != 2 || string(sent[0]) != "\x01\x02\x03" || string(sent[1]) != "\x04\x05" {
		t.Errorf("Wrong match. expect only the two non-empty binary frames, got: %v", sent)
	}
	if ctx.Err() == nil {
		t.Errorf("Wrong match. expect the context to be cancelled when the media ends")
	}
}

func Test_gcpProcessMedia_sendFailureStops(t *testing.T) {
	tests := []struct {
		name string

		sendErr error
	}{
		{"service closed the stream", io.EOF},
		{"send failed", grpcstatus.Error(codes.Internal, "send failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := connectTestMedia(t, func(conn net.Conn) {
				serverWriteFrame(conn, websocket.BinaryMessage, []byte{0x01})
				serverWriteFrame(conn, websocket.BinaryMessage, []byte{0x02})
				serverWaitClosed(conn)
			})

			h := &streamingHandler{}
			st := newFakeSTTStreaming("en-US", transcript.DirectionIn)
			st.ConnAst = conn
			streamClient := &recordingStreamClient{sendErr: tt.sendErr}

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				defer close(done)
				h.gcpProcessMedia(ctx, cancel, st, streamClient)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("Wrong match. gcpProcessMedia did not stop after the send failure")
			}

			if len(streamClient.sent()) != 1 {
				t.Errorf("Wrong match. expect: stop after the first failed send, got: %v", streamClient.sent())
			}
		})
	}
}

func Test_gcpProcessMedia_cancelledContextReadsNothing(t *testing.T) {
	conn := connectTestMedia(t, func(conn net.Conn) {
		serverWriteFrame(conn, websocket.BinaryMessage, []byte{0x01})
		serverWaitClosed(conn)
	})

	h := &streamingHandler{}
	st := newFakeSTTStreaming("en-US", transcript.DirectionIn)
	st.ConnAst = conn
	streamClient := &recordingStreamClient{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	h.gcpProcessMedia(ctx, cancel, st, streamClient)

	if len(streamClient.sent()) != 0 {
		t.Errorf("Wrong match. expect: no audio after the context is cancelled, got: %v", streamClient.sent())
	}
}

func Test_gcpRun_deliversFinalResult(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	client, fake, _ := newFakeStreamingSpeech(t, []*speechpb.StreamingRecognizeResponse{gcpFinalResponse("hello world")}, false)
	h := &streamingHandler{gcpClient: client, notifyHandler: mockNotify, transcriptHandler: mockTranscript}

	st := newFakeSTTStreaming("en-US", transcript.DirectionOut)
	st.ConnAst = connectTestMedia(t, func(conn net.Conn) { serverWaitClosed(conn) })

	mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionOut, "hello world", gomock.Any()).Return(&transcript.Transcript{}, nil)

	if err := h.gcpRun(st); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case first := <-fake.configs:
		if first.GetStreamingConfig().GetConfig().GetLanguageCode() != "en-US" {
			t.Errorf("Wrong match. unexpected config: %v", first)
		}
	default:
		t.Errorf("Wrong match. the service did not receive the recognition config")
	}

	// the media reader only ends when the connection to Asterisk is closed
	_ = st.ConnAst.Close()
	waitMediaReaderGone(t)
}

func Test_gcpRun_initFailure(t *testing.T) {
	client, _, streamCalls := newFakeStreamingSpeech(t, nil, true)
	h := &streamingHandler{gcpClient: client}

	st := newFakeSTTStreaming("en-US", transcript.DirectionBoth)

	if err := h.gcpRun(st); err == nil {
		t.Fatalf("Wrong match. expect: err, got: ok")
	}
	if streamCalls.Load() != 1 {
		t.Errorf("Wrong match. expect: 1 attempt, got: %d", streamCalls.Load())
	}
}

func Test_runSTT_withGCPClient(t *testing.T) {
	tests := []struct {
		name string

		provider transcribe.Provider
	}{
		{"requested gcp provider runs once", transcribe.Provider("gcp")},
		{"default order runs gcp once", transcribe.ProviderEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			client, _, streamCalls := newFakeStreamingSpeech(t, []*speechpb.StreamingRecognizeResponse{gcpFinalResponse("good day")}, false)
			h := &streamingHandler{gcpClient: client, notifyHandler: mockNotify, transcriptHandler: mockTranscript}

			st := newFakeSTTStreaming("en-US", transcript.DirectionIn)
			st.ConnAst = connectTestMedia(t, func(conn net.Conn) { serverWaitClosed(conn) })

			mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionIn, "good day", gomock.Any()).Return(&transcript.Transcript{}, nil)

			h.runSTT(st, tt.provider)

			if streamCalls.Load() != 1 {
				t.Errorf("Wrong match. expect: 1 stream, got: %d", streamCalls.Load())
			}

			_ = st.ConnAst.Close()
			waitMediaReaderGone(t)
		})
	}
}

func Test_runSTT_requestedProviderFails(t *testing.T) {
	tests := []struct {
		name string

		provider transcribe.Provider
	}{
		{"requested gcp provider is not retried by the default order", transcribe.Provider("gcp")},
		{"default order tries gcp once", transcribe.ProviderEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _, streamCalls := newFakeStreamingSpeech(t, nil, true)
			h := &streamingHandler{gcpClient: client}

			st := newFakeSTTStreaming("en-US", transcript.DirectionIn)

			h.runSTT(st, tt.provider)

			if streamCalls.Load() != 1 {
				t.Errorf("Wrong match. expect: 1 attempt, got: %d", streamCalls.Load())
			}
		})
	}
}
