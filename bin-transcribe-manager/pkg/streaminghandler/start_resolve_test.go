package streaminghandler

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/internal/logtest"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// countingFactory returns a client factory that counts its calls and hands out a distinct client
// for each call.
func countingFactory(calls *atomic.Int32) func(string, string) (*transcribestreaming.Client, error) {
	return func(string, string) (*transcribestreaming.Client, error) {
		calls.Add(1)
		return &transcribestreaming.Client{}, nil
	}
}

func Test_awsClientFor(t *testing.T) {
	t.Run("no supplier means no AWS", func(t *testing.T) {
		h := &streamingHandler{}
		if c, err := h.awsClientFor(context.Background()); c != nil || err != nil {
			t.Errorf("Wrong match. expect: nil and no error, got: %v %v", c, err)
		}
	})

	t.Run("an empty or a half pair means no AWS", func(t *testing.T) {
		for _, p := range [][2]string{{"", ""}, {"a", ""}, {"", "s"}} {
			var calls atomic.Int32
			h := &streamingHandler{awsKeys: pairKeys(p[0], p[1]), newAWSClient: countingFactory(&calls)}
			if c, err := h.awsClientFor(context.Background()); c != nil || err != nil || calls.Load() != 0 {
				t.Errorf("Wrong match. %v expect: nil, no error and no client built, got: %v %v %d", p, c, err, calls.Load())
			}
		}
	})

	t.Run("a read error is an error and builds nothing", func(t *testing.T) {
		var calls atomic.Int32
		readErr := errors.New("read failed")
		h := &streamingHandler{
			awsKeys:      func(context.Context) (string, string, error) { return "", "", readErr },
			newAWSClient: countingFactory(&calls),
		}
		if c, err := h.awsClientFor(context.Background()); c != nil || !errors.Is(err, readErr) || calls.Load() != 0 {
			t.Errorf("Wrong match. expect: nil and the read error, got: %v %v %d", c, err, calls.Load())
		}
	})

	t.Run("the client is reused for the same pair and rebuilt when the pair changes", func(t *testing.T) {
		var calls atomic.Int32
		pair := [2]string{"access-1", "secret-1"}
		h := &streamingHandler{
			awsKeys: func(context.Context) (string, string, error) { return pair[0], pair[1], nil },
			// the factory is the one that counts
			newAWSClient: countingFactory(&calls),
		}
		ctx := context.Background()

		first, err := h.awsClientFor(ctx)
		if first == nil || err != nil {
			t.Fatalf("Wrong match. expect: a client, got: %v %v", first, err)
		}
		again, _ := h.awsClientFor(ctx)
		if again != first || calls.Load() != 1 {
			t.Errorf("Wrong match. expect: the same client and one build, got: same=%v builds=%d", again == first, calls.Load())
		}

		// the access key changes, then the secret key: each change builds a new client
		pair[0] = "access-2"
		second, _ := h.awsClientFor(ctx)
		if second == nil || second == first || calls.Load() != 2 {
			t.Errorf("Wrong match. expect: a new client, got: same=%v builds=%d", second == first, calls.Load())
		}
		pair[1] = "secret-2"
		third, _ := h.awsClientFor(ctx)
		if third == nil || third == second || calls.Load() != 3 {
			t.Errorf("Wrong match. expect: a new client, got: same=%v builds=%d", third == second, calls.Load())
		}

		// the clients that were handed out before stay usable values: the replacement does not touch them
		if first == nil || second == nil {
			t.Errorf("Wrong match. a handed out client was changed")
		}
	})

	t.Run("a client that can not be built is no AWS and is not cached", func(t *testing.T) {
		var calls atomic.Int32
		h := &streamingHandler{
			awsKeys: pairKeys("a", "s"),
			newAWSClient: func(string, string) (*transcribestreaming.Client, error) {
				calls.Add(1)
				return nil, errors.New("build failed")
			},
		}
		for i := 0; i < 2; i++ {
			if c, err := h.awsClientFor(context.Background()); c != nil || err != nil {
				t.Errorf("Wrong match. expect: nil and no error, got: %v %v", c, err)
			}
		}
		if calls.Load() != 2 {
			t.Errorf("Wrong match. expect: a build for every call, got: %d", calls.Load())
		}
	})
}

// Concurrent calls with one pair build once, and calls with changing pairs never race (run with
// -race). The pair is read outside the lock.
func Test_awsClientFor_concurrent(t *testing.T) {
	var calls atomic.Int32
	h := &streamingHandler{awsKeys: pairKeys("a", "s"), newAWSClient: countingFactory(&calls)}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c, err := h.awsClientFor(context.Background()); c == nil || err != nil {
				t.Errorf("Wrong match. expect: a client, got: %v %v", c, err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Errorf("Wrong match. expect: one build for one pair, got: %d", calls.Load())
	}

	var n atomic.Int32
	h2 := &streamingHandler{
		awsKeys: func(context.Context) (string, string, error) {
			return "a", string(rune('a' + n.Add(1)%3)), nil
		},
		newAWSClient: countingFactory(&calls),
	}
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h2.awsClientFor(context.Background())
		}()
	}
	wg.Wait()
}

func newResolveHandler(mc *gomock.Controller) (*streamingHandler, *requesthandler.MockRequestHandler) {
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &streamingHandler{
		utilHandler:   utilhandler.NewMockUtilHandler(mc),
		reqHandler:    mockReq,
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		mapStreaming:  make(map[uuid.UUID]*streaming.Streaming),
	}

	return h, mockReq
}

func startForTest(h *streamingHandler) (*streaming.Streaming, error) {
	return h.Start(context.Background(), uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, uuid.Nil, "en-US", transcript.DirectionIn, transcribe.ProviderEmpty)
}

// With no provider Start answers at once. The mocks have no expectation, so a record, an event or
// an external media would fail the test: nothing is created (the baseline of the disabled handler).
func Test_Start_noProvider(t *testing.T) {
	tests := []struct {
		name string

		awsKeys AWSKeyFunc

		expectReason string
	}{
		{"no supplier", nil, ErrSTTNotConfiguredReason},
		{"no credential", pairKeys("", ""), ErrSTTNotConfiguredReason},
		{"half a credential", pairKeys("a", ""), ErrSTTNotConfiguredReason},
		{"read error", func(context.Context) (string, string, error) { return "", "", errors.New("read failed") }, ErrSTTKeyUnavailableReason},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newResolveHandler(mc)
			h.awsKeys = tt.awsKeys
			h.newAWSClient = func(string, string) (*transcribestreaming.Client, error) { return &transcribestreaming.Client{}, nil }

			res, err := startForTest(h)
			if res != nil || !isReason(err, tt.expectReason) {
				t.Errorf("Wrong match. expect: nil and %s, got: %v %v", tt.expectReason, res, err)
			}
			var ve *cerrors.VoipbinError
			if !errors.As(err, &ve) || ve.Status != cerrors.StatusUnavailable {
				t.Errorf("Wrong match. expect: Unavailable, got: %v", err)
			}
			if len(h.mapStreaming) != 0 {
				t.Errorf("Wrong match. expect: no streaming record, got: %d", len(h.mapStreaming))
			}
		})
	}
}

// A handler is not pinned to "no provider" by what it saw earlier: after the supplier starts to
// hand out a pair (the key was stored in the database), the same handler goes past the gate.
func Test_Start_notPinnedToNoProvider(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newResolveHandler(mc)
	var stored atomic.Bool
	h.awsKeys = func(context.Context) (string, string, error) {
		if stored.Load() {
			return "access", "secret", nil
		}
		return "", "", nil
	}
	h.newAWSClient = func(string, string) (*transcribestreaming.Client, error) { return &transcribestreaming.Client{}, nil }

	if _, err := startForTest(h); !isReason(err, ErrSTTNotConfiguredReason) {
		t.Fatalf("Wrong match. expect: %s, got: %v", ErrSTTNotConfiguredReason, err)
	}

	stored.Store(true)
	h.utilHandler.(*utilhandler.MockUtilHandler).EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("5c0f1b10-0000-11ee-8c10-000000000001"))
	h.notifyHandler.(*notifyhandler.MockNotifyHandler).EXPECT().PublishEvent(gomock.Any(), streaming.EventTypeStreamingStarted, gomock.Any())
	// the external media start is reached: the gate let the request through
	startErr := errors.New("call manager unavailable")
	mockReq.EXPECT().CallV1ExternalMediaStart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, startErr)

	if _, err := startForTest(h); !errors.Is(err, startErr) {
		t.Errorf("Wrong match. expect: the external media error, got: %v", err)
	}
}

// With a GCP client a credential that can not be read does not stop Start: AWS is skipped, the
// request goes on, and one warning names the failure without any key text.
func Test_Start_readErrorWithGCPContinues(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockReq := newResolveHandler(mc)
	h.gcpClient = &speech.Client{}
	h.awsKeys = func(context.Context) (string, string, error) { return "", "", errors.New("read failed") }

	h.utilHandler.(*utilhandler.MockUtilHandler).EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("5c0f1b10-0000-11ee-8c10-000000000002"))
	h.notifyHandler.(*notifyhandler.MockNotifyHandler).EXPECT().PublishEvent(gomock.Any(), streaming.EventTypeStreamingStarted, gomock.Any())
	startErr := errors.New("call manager unavailable")
	mockReq.EXPECT().CallV1ExternalMediaStart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, startErr)

	logs := logtest.New(t)
	if _, err := startForTest(h); !errors.Is(err, startErr) {
		t.Errorf("Wrong match. expect: the external media error, got: %v", err)
	}
	if !logs.HasLevel(logrus.WarnLevel) || !strings.Contains(logs.Dump(), "AWS is not used for this streaming") {
		t.Errorf("Wrong match. expect: a warning, got: %s", logs.Dump())
	}
}

// Start passes its context to the supplier, so a caller that gives up is not kept waiting.
func Test_Start_supplierFollowsTheContext(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newResolveHandler(mc)
	h.awsKeys = func(ctx context.Context) (string, string, error) {
		<-ctx.Done()
		return "", "", ctx.Err()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := h.Start(ctx, uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, uuid.Nil, "en-US", transcript.DirectionIn, transcribe.ProviderEmpty)
	if !isReason(err, ErrSTTKeyUnavailableReason) || time.Since(started) > 2*time.Second {
		t.Errorf("Wrong match. expect: %s promptly, got: %v after %v", ErrSTTKeyUnavailableReason, err, time.Since(started))
	}
}

// Stop and Run never read the credential, and Stop of an unknown session is the NotFound that
// transcribehandler treats as already stopped (it used to be STT_NOT_CONFIGURED on a handler
// without a provider).
func Test_Stop_andRun_doNotReadTheCredential(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newResolveHandler(mc)
	h.awsKeys = func(context.Context) (string, string, error) {
		t.Errorf("Wrong match. the credential must not be read by Stop or Run")
		return "", "", nil
	}

	if err := h.Run(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	_, err := h.Stop(context.Background(), uuid.FromStringOrNil("5c0f1b10-0000-11ee-8c10-000000000003"))
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Status != cerrors.StatusNotFound || ve.Reason != "STREAMING_NOT_FOUND" {
		t.Errorf("Wrong match. expect: STREAMING_NOT_FOUND, got: %v", err)
	}
}

// The constructor reads the credential once to log what is available. A read failure warns and
// never stops the start, the handler still reads the credential at every Start, and no key text
// reaches the log.
func Test_NewStreamingHandler_readFailureOnlyWarns(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing.json"))

	var reads atomic.Int32
	logs := logtest.New(t)
	h := NewStreamingHandler(
		requesthandler.NewMockRequestHandler(mc),
		notifyhandler.NewMockNotifyHandler(mc),
		transcripthandler.NewMockTranscriptHandler(mc),
		func(context.Context) (string, string, error) {
			reads.Add(1)
			return "", "", errors.New("read failed")
		},
	)
	if h == nil || reads.Load() != 1 {
		t.Fatalf("Wrong match. expect: a handler and one read, got: %v %d", h, reads.Load())
	}
	if !logs.HasLevel(logrus.WarnLevel) {
		t.Errorf("Wrong match. expect: a warning, got: %s", logs.Dump())
	}

	if _, err := startForTest(h.(*streamingHandler)); !isReason(err, ErrSTTKeyUnavailableReason) || reads.Load() != 2 {
		t.Errorf("Wrong match. expect: a read at every Start and %s, got: %v after %d reads", ErrSTTKeyUnavailableReason, err, reads.Load())
	}
}

// The key text never reaches the log or the error of a Start that fails to resolve the provider.
func Test_Start_noKeyTextInLogs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	const secret = "synthetic-credential-text"
	h, _ := newResolveHandler(mc)
	h.awsKeys = func(context.Context) (string, string, error) { return secret, secret, errors.New("read failed") }

	logs := logtest.New(t)
	_, err := startForTest(h)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Errorf("Wrong match. the error must not hold the key: %v", err)
	}
	logs.AssertAbsent(t, secret)
}
