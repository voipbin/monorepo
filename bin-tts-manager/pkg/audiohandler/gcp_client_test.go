package audiohandler

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gofrs/uuid"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"monorepo/bin-tts-manager/models/tts"
)

// fakeTTSServer is an in-process Cloud Text-to-Speech server bound to the
// loopback interface. Requests are handed to the test through buffered
// channels so the test goroutine reads them with a proper happens-before edge.
type fakeTTSServer struct {
	texttospeechpb.UnimplementedTextToSpeechServer

	synthAudio []byte
	synthErr   error
	listErr    error

	synthReqs chan *texttospeechpb.SynthesizeSpeechRequest
	listReqs  chan *texttospeechpb.ListVoicesRequest
}

func (s *fakeTTSServer) SynthesizeSpeech(_ context.Context, req *texttospeechpb.SynthesizeSpeechRequest) (*texttospeechpb.SynthesizeSpeechResponse, error) {
	s.synthReqs <- req
	if s.synthErr != nil {
		return nil, s.synthErr
	}
	return &texttospeechpb.SynthesizeSpeechResponse{AudioContent: s.synthAudio}, nil
}

func (s *fakeTTSServer) ListVoices(_ context.Context, req *texttospeechpb.ListVoicesRequest) (*texttospeechpb.ListVoicesResponse, error) {
	s.listReqs <- req
	if s.listErr != nil {
		return nil, s.listErr
	}
	return &texttospeechpb.ListVoicesResponse{}, nil
}

// startFakeTTS starts the fake server and returns a client connected to it
// without authentication or TLS. Both are cleaned up when the test ends.
func startFakeTTS(t *testing.T, srv *fakeTTSServer) *texttospeech.Client {
	t.Helper()

	srv.synthReqs = make(chan *texttospeechpb.SynthesizeSpeechRequest, 8)
	srv.listReqs = make(chan *texttospeechpb.ListVoicesRequest, 8)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}

	gs := grpc.NewServer()
	texttospeechpb.RegisterTextToSpeechServer(gs, srv)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = gs.Serve(lis)
	}()

	//nolint:staticcheck // same texttospeech library limitation as gcp.go
	client, err := texttospeech.NewClient(
		context.Background(),
		option.WithEndpoint(lis.Addr().String()),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		gs.Stop()
		<-done
		t.Fatalf("could not create a client. err: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		gs.Stop()
		<-done
	})

	return client
}

// closedLoopbackAddr returns a loopback address on which nothing listens.
func closedLoopbackAddr(t *testing.T) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	return addr
}

// setFakeADC points the application default credentials at a synthetic
// authorized_user file and isolates the AWS shared config lookup. The fake
// values are never sent anywhere because the tests only dial a closed
// loopback port.
func setFakeADC(t *testing.T) {
	t.Helper()

	cred := map[string]string{"type": "authorized_user"}
	cred["client_id"] = "fake-client-id"
	cred["client_secret"] = "fake-client-secret"
	cred["refresh_token"] = "fake-refresh-token"

	data, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("could not marshal the credential. err: %v", err)
	}

	path := filepath.Join(t.TempDir(), "adc.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("could not write the credential. err: %v", err)
	}

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", path)
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_PROFILE", "")
}

// setMissingADC points the application default credentials at a file that
// does not exist, so client creation fails locally and deterministically.
func setMissingADC(t *testing.T) {
	t.Helper()

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing.json"))
}

func Test_gcpAudioCreate(t *testing.T) {
	tests := []struct {
		name string

		text    string
		lang    string
		voiceID string

		expectVoiceName string
	}{
		{
			name: "explicit voice id is sent as is",

			text:    "<speak>hello</speak>",
			lang:    "en-US",
			voiceID: "en-US-Custom-Voice",

			expectVoiceName: "en-US-Custom-Voice",
		},
		{
			name: "empty voice id falls back to the language default",

			text:    "<speak>annyeong</speak>",
			lang:    "ko-KR",
			voiceID: "",

			expectVoiceName: "ko-KR-Neural2-C",
		},
		{
			name: "unknown language leaves the voice name empty for auto selection",

			text:    "<speak>hola</speak>",
			lang:    "xx-XX",
			voiceID: "",

			expectVoiceName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audio := []byte("synthetic-audio-" + tt.lang)
			srv := &fakeTTSServer{synthAudio: audio}
			h := &audioHandler{gcpClient: startFakeTTS(t, srv)}

			outPath := filepath.Join(t.TempDir(), "out.wav")
			if err := h.gcpAudioCreate(context.Background(), uuid.Must(uuid.NewV4()), tt.text, tt.lang, tt.voiceID, outPath); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			req := <-srv.synthReqs
			if req.GetInput().GetSsml() != tt.text {
				t.Errorf("Wrong ssml. expect: %s, got: %s", tt.text, req.GetInput().GetSsml())
			}
			if req.GetVoice().GetLanguageCode() != tt.lang {
				t.Errorf("Wrong language code. expect: %s, got: %s", tt.lang, req.GetVoice().GetLanguageCode())
			}
			if req.GetVoice().GetName() != tt.expectVoiceName {
				t.Errorf("Wrong voice name. expect: %s, got: %s", tt.expectVoiceName, req.GetVoice().GetName())
			}
			if req.GetAudioConfig().GetAudioEncoding() != defaultAudioEncoding {
				t.Errorf("Wrong audio encoding. expect: %v, got: %v", defaultAudioEncoding, req.GetAudioConfig().GetAudioEncoding())
			}
			if req.GetAudioConfig().GetSampleRateHertz() != defaultSampleRate {
				t.Errorf("Wrong sample rate. expect: %d, got: %d", defaultSampleRate, req.GetAudioConfig().GetSampleRateHertz())
			}

			got, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("Could not read the created file. err: %v", err)
			}
			if string(got) != string(audio) {
				t.Errorf("Wrong file content. expect: %s, got: %s", audio, got)
			}
		})
	}
}

func Test_gcpAudioCreate_synthesizeError(t *testing.T) {
	// a non-retryable code, so the client returns at once
	srv := &fakeTTSServer{synthErr: status.Error(codes.InvalidArgument, "bad request")}
	h := &audioHandler{gcpClient: startFakeTTS(t, srv)}

	outPath := filepath.Join(t.TempDir(), "out.wav")
	err := h.gcpAudioCreate(context.Background(), uuid.Must(uuid.NewV4()), "<speak>x</speak>", "en-US", "", outPath)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Wrong error. expect: %v, got: %v", codes.InvalidArgument, err)
	}

	if _, errStat := os.Stat(outPath); !os.IsNotExist(errStat) {
		t.Errorf("The file must not be created on a synthesize error. stat err: %v", errStat)
	}
}

func Test_gcpAudioCreate_writeError(t *testing.T) {
	srv := &fakeTTSServer{synthAudio: []byte("synthetic-audio")}
	h := &audioHandler{gcpClient: startFakeTTS(t, srv)}

	// the parent directory does not exist, so the write fails regardless of the user's privileges
	outPath := filepath.Join(t.TempDir(), "missing-dir", "out.wav")
	err := h.gcpAudioCreate(context.Background(), uuid.Must(uuid.NewV4()), "<speak>x</speak>", "en-US", "", outPath)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("Wrong error. expect: a not-exist write error, got: %v", err)
	}
}

func Test_AudioCreate_provider(t *testing.T) {
	t.Run("gcp is delegated to the gcp synthesizer", func(t *testing.T) {
		srv := &fakeTTSServer{synthAudio: []byte("synthetic-audio")}
		h := &audioHandler{gcpClient: startFakeTTS(t, srv)}

		outPath := filepath.Join(t.TempDir(), "out.wav")
		if err := h.AudioCreate(context.Background(), uuid.Must(uuid.NewV4()), "<speak>x</speak>", "en-US", tts.ProviderGCP, "", outPath); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case <-srv.synthReqs:
		default:
			t.Errorf("The gcp synthesizer was not called.")
		}
	})

	t.Run("unsupported provider is rejected", func(t *testing.T) {
		h := &audioHandler{}

		err := h.AudioCreate(context.Background(), uuid.Must(uuid.NewV4()), "<speak>x</speak>", "en-US", tts.Provider("unknown"), "", "unused")
		if err == nil || !strings.Contains(err.Error(), "unsupported provider: unknown") {
			t.Fatalf("Wrong error. expect: unsupported provider, got: %v", err)
		}
	})
}

func Test_gcpWarmUpConnection(t *testing.T) {
	tests := []struct {
		name string

		listErr error
	}{
		{
			name: "warm-up succeeds",
		},
		{
			name: "warm-up failure is not fatal",

			listErr: status.Error(codes.Unimplemented, "not implemented"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &fakeTTSServer{listErr: tt.listErr}
			client := startFakeTTS(t, srv)

			gcpWarmUpConnection(context.Background(), client)

			select {
			case <-srv.listReqs:
			default:
				t.Errorf("ListVoices was not called by the warm-up.")
			}
		})
	}
}

func Test_gcpGetClient_missingCredentials(t *testing.T) {
	setMissingADC(t)

	// an empty endpoint takes the default endpoint branch, then fails on the missing credential file
	client, err := gcpGetClient(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "could not create a new client") {
		t.Fatalf("Wrong error. expect: could not create a new client, got: %v", err)
	}
	if client != nil {
		t.Errorf("Wrong client. expect: nil, got: %v", client)
	}
}

func Test_gcpGetClient_warmUpFailureIsNotFatal(t *testing.T) {
	setFakeADC(t)

	// nothing listens on the endpoint, so the warm-up runs into the context deadline
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client, err := gcpGetClient(ctx, closedLoopbackAddr(t))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if client == nil {
		t.Fatalf("Wrong client. expect: not nil, got: nil")
	}
	_ = client.Close()
}

func Test_NewAudioHandler(t *testing.T) {
	t.Run("returns nil when the gcp client cannot be created", func(t *testing.T) {
		setMissingADC(t)

		if res := NewAudioHandler(context.Background(), "fake-access", "fake-secret", "unused"); res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})

	t.Run("returns nil when the aws client cannot be created", func(t *testing.T) {
		setFakeADC(t)
		t.Setenv("AWS_PROFILE", "does-not-exist")

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		if res := NewAudioHandler(ctx, "fake-access", "fake-secret", closedLoopbackAddr(t)); res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})

	t.Run("returns a handler holding both clients", func(t *testing.T) {
		setFakeADC(t)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		res := NewAudioHandler(ctx, "fake-access", "fake-secret", closedLoopbackAddr(t))
		h, ok := res.(*audioHandler)
		if !ok || h == nil {
			t.Fatalf("Wrong match. expect: *audioHandler, got: %T", res)
		}
		if h.gcpClient == nil || h.awsClient == nil {
			t.Errorf("Wrong clients. gcp nil: %v, aws nil: %v", h.gcpClient == nil, h.awsClient == nil)
		}
		_ = h.gcpClient.Close()
	})
}
