package streaminghandler_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// encodeEventStreamMessage encodes one event stream message: the prelude with its checksum, the
// string headers, the payload and the checksum of the whole message.
func encodeEventStreamMessage(headers [][2]string, payload []byte) []byte {
	var hdr bytes.Buffer
	for _, h := range headers {
		hdr.WriteByte(byte(len(h[0])))
		hdr.WriteString(h[0])
		hdr.WriteByte(7) // string value type
		_ = binary.Write(&hdr, binary.BigEndian, uint16(len(h[1])))
		hdr.WriteString(h[1])
	}

	total := 12 + hdr.Len() + len(payload) + 4
	var msg bytes.Buffer
	_ = binary.Write(&msg, binary.BigEndian, uint32(total))
	_ = binary.Write(&msg, binary.BigEndian, uint32(hdr.Len()))
	_ = binary.Write(&msg, binary.BigEndian, crc32.ChecksumIEEE(msg.Bytes()))
	msg.Write(hdr.Bytes())
	msg.Write(payload)
	_ = binary.Write(&msg, binary.BigEndian, crc32.ChecksumIEEE(msg.Bytes()))

	return msg.Bytes()
}

// awsTranscriptEventFrame returns a transcript event frame holding one result.
func awsTranscriptEventFrame(partial bool, text string) []byte {
	payload := `{"Transcript":{"Results":[{"IsPartial":false,"Alternatives":[{"Transcript":"` + text + `"}]}]}}`
	if partial {
		payload = strings.Replace(payload, `"IsPartial":false`, `"IsPartial":true`, 1)
	}

	return encodeEventStreamMessage([][2]string{
		{":message-type", "event"},
		{":event-type", "TranscriptEvent"},
		{":content-type", "application/json"},
	}, []byte(payload))
}

// fakeAWSTranscribe is a local stand-in for the AWS Transcribe streaming service. It answers a
// stream with the configured frames, records the bytes the client sends and ends the response
// when told to.
type fakeAWSTranscribe struct {
	frames [][]byte
	finish chan struct{}

	mu       sync.Mutex
	received []byte
}

func (f *fakeAWSTranscribe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	// the call is bidirectional: the request body is read while the response is being written
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				f.mu.Lock()
				f.received = append(f.received, buf[:n]...)
				f.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	for _, frame := range f.frames {
		if _, err := w.Write(frame); err != nil {
			return
		}
		flusher.Flush()
	}

	select {
	case <-f.finish:
	case <-r.Context().Done():
	}
}

// endResponse ends the response stream, as the service does when the transcription is over.
func (f *fakeAWSTranscribe) endResponse() {
	close(f.finish)
}

// waitReceived waits until the service has received the given bytes from the client.
func (f *fakeAWSTranscribe) waitReceived(t *testing.T, want []byte) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		ok := bytes.Contains(f.received, want)
		f.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("Wrong match. the service did not receive the audio chunk")
}

// countReceived returns how many times the given bytes were received from the client.
func (f *fakeAWSTranscribe) countReceived(want []byte) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return bytes.Count(f.received, want)
}

func (f *fakeAWSTranscribe) hasReceived(want []byte) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return bytes.Contains(f.received, want)
}

const awsTestRegion = "us-west-2"

// newFakeAWSTranscribe starts the fake and returns a client connected to it.
func newFakeAWSTranscribe(t *testing.T, frames [][]byte) (*transcribestreaming.Client, *fakeAWSTranscribe) {
	t.Helper()

	fake := &fakeAWSTranscribe{frames: frames, finish: make(chan struct{})}

	srv := httptest.NewUnstartedServer(fake)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)

	client := transcribestreaming.New(transcribestreaming.Options{
		Region:       awsTestRegion,
		BaseEndpoint: aws.String(srv.URL),
		HTTPClient:   srv.Client(),
		Credentials:  credentials.NewStaticCredentialsProvider("test_access_key", "test_secret_key", ""),
		Retryer:      aws.NopRetryer{},
	})

	return client, fake
}

// waitAWSMediaReaderGone waits until no goroutine is left in awsProcessMedia.
func waitAWSMediaReaderGone(t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		if !strings.Contains(string(buf[:n]), "(*streamingHandler).awsProcessMedia") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Wrong match. awsProcessMedia is still running")
}

// connectAWSTestMedia connects a websocket client to a loopback media server that runs the
// scenario on the server side and then waits for the client to close.
func connectAWSTestMedia(t *testing.T, scenario func(conn *websocket.Conn)) *websocket.Conn {
	t.Helper()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		scenario(conn)

		// wait for the client to close, with a bound so a missing close fails instead of hanging
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			if _, _, errRead := conn.ReadMessage(); errRead != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("could not connect the media server. err: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

func newAWSTestStreaming(direction transcript.Direction) *streaming.Streaming {
	return &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("7d2a3c10-0000-11ee-8b11-000000000011"),
			CustomerID: uuid.FromStringOrNil("7d2a3c10-0000-11ee-8b11-000000000012"),
		},
		TranscribeID: uuid.FromStringOrNil("7d2a3c10-0000-11ee-8b11-000000000013"),
		Language:     "en-US",
		Direction:    direction,
	}
}

// runAWSRun runs awsRun in the background and returns the channel carrying its result.
func runAWSRun(
	client *transcribestreaming.Client,
	notifyHandler notifyhandler.NotifyHandler,
	transcriptHandler transcripthandler.TranscriptHandler,
	st *streaming.Streaming,
) <-chan error {
	done := make(chan error, 1)
	go func() { done <- streaminghandler.AWSRunForTest(client, notifyHandler, transcriptHandler, st) }()

	return done
}

func waitAWSRunResult(t *testing.T, done <-chan error) error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. awsRun did not return")
		return nil
	}
}

func Test_awsRun_streamsAudioAndDeliversTranscripts(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	client, fake := newFakeAWSTranscribe(t, [][]byte{
		awsTranscriptEventFrame(true, "hel"),
		awsTranscriptEventFrame(false, "hello world"),
	})

	audio := []byte("audio-chunk-one-0123456789")
	released := make(chan struct{})
	st := newAWSTestStreaming(transcript.DirectionIn)
	st.ConnAst = connectAWSTestMedia(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("status"))
		_ = conn.WriteMessage(websocket.BinaryMessage, []byte{})
		_ = conn.WriteMessage(websocket.BinaryMessage, audio)
		<-released
	})

	gomock.InOrder(
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionIn, "hello world", gomock.Any()).Return(&transcript.Transcript{}, nil),
	)

	done := runAWSRun(client, mockNotify, mockTranscript, st)

	// the audio reaches the service; only then is the response ended
	fake.waitReceived(t, audio)
	fake.endResponse()

	if err := waitAWSRunResult(t, done); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	close(released)
	_ = st.ConnAst.Close()
	waitAWSMediaReaderGone(t)

	// only the non-empty binary audio is forwarded: not the text status message and not the empty chunk
	if fake.hasReceived([]byte("status")) {
		t.Errorf("Wrong match. expect: the text message is not forwarded as audio")
	}
	if got := fake.countReceived([]byte("AudioEvent")); got != 1 {
		t.Errorf("Wrong match. expect: 1 audio event, got: %d", got)
	}
}

func Test_awsRun_returnsWhenTheMediaEnds(t *testing.T) {
	client, fake := newFakeAWSTranscribe(t, nil)

	audio := []byte("audio-chunk-only-0123456789")
	st := newAWSTestStreaming(transcript.DirectionBoth)
	st.ConnAst = connectAWSTestMedia(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.BinaryMessage, audio)
		// the media side ends first: the connection to Asterisk is closed
		_ = conn.Close()
	})

	done := runAWSRun(client, nil, nil, st)

	// the response stream is never ended by the service here; awsRun must return on its own
	if err := waitAWSRunResult(t, done); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	waitAWSMediaReaderGone(t)

	fake.waitReceived(t, audio)
}

func Test_awsProcessMedia_afterTheResponseEnded(t *testing.T) {
	tests := []struct {
		name string

		opcode  int
		payload []byte
	}{
		{"audio that arrives after the response ended is not delivered", websocket.BinaryMessage, []byte("audio-chunk-late-9876543210")},
		{"a text message that arrives after the end stops the reader", websocket.TextMessage, []byte("status")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, fake := newFakeAWSTranscribe(t, nil)

			first := []byte("audio-chunk-first-0123456789")
			released := make(chan struct{})
			st := newAWSTestStreaming(transcript.DirectionOut)
			st.ConnAst = connectAWSTestMedia(t, func(conn *websocket.Conn) {
				_ = conn.WriteMessage(websocket.BinaryMessage, first)
				<-released
				_ = conn.WriteMessage(tt.opcode, tt.payload)
			})

			done := runAWSRun(client, nil, nil, st)

			// the reader is inside its loop once the first chunk reached the service
			fake.waitReceived(t, first)
			fake.endResponse()

			if err := waitAWSRunResult(t, done); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			close(released)
			waitAWSMediaReaderGone(t)

			if tt.opcode == websocket.BinaryMessage && fake.hasReceived(tt.payload) {
				t.Errorf("Wrong match. expect: no audio after the response ended")
			}
		})
	}
}
