package audiohandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/gofrs/uuid"
)

// countingBuilder counts the calls of the client builder and returns a client of a local server.
type countingBuilder struct {
	calls    atomic.Int32
	lastPair atomic.Value
	endpoint string
}

func (b *countingBuilder) build(_ context.Context, accessKey, secretKey string) (*polly.Client, error) {
	b.calls.Add(1)
	b.lastPair.Store(accessKey + "/" + secretKey)

	return newLocalPollyClient(b.endpoint), nil
}

// pairSupplier returns a supplier whose pair can be changed between calls.
type pairSupplier struct {
	mu     sync.Mutex
	access string
	secret string
	err    error
}

func (p *pairSupplier) set(access, secret string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.access, p.secret, p.err = access, secret, err
}

func (p *pairSupplier) get(context.Context) (string, string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.access, p.secret, p.err
}

func Test_pollyClient_reuseAndRotate(t *testing.T) {
	b := &countingBuilder{endpoint: "http://127.0.0.1:1"}
	p := &pairSupplier{access: "access-1", secret: "secret-1"}
	h := &audioHandler{awsCreds: p.get, awsBuild: b.build}
	ctx := context.Background()

	c1, err := h.pollyClient(ctx)
	if err != nil || c1 == nil {
		t.Fatalf("Wrong match. expect: a client, got: %v %v", c1, err)
	}
	c2, _ := h.pollyClient(ctx)
	if c2 != c1 || b.calls.Load() != 1 {
		t.Errorf("Wrong match. expect: the same client and one build, got: %d builds", b.calls.Load())
	}

	// a changed secret builds a new client with the new pair
	p.set("access-1", "secret-2", nil)
	c3, _ := h.pollyClient(ctx)
	if c3 == c1 || b.calls.Load() != 2 || b.lastPair.Load() != "access-1/secret-2" {
		t.Errorf("Wrong match. expect: a new client for the new secret, got: %d builds, pair %v", b.calls.Load(), b.lastPair.Load())
	}

	// a changed access key builds a new client too
	p.set("access-2", "secret-2", nil)
	c4, _ := h.pollyClient(ctx)
	if c4 == c3 || b.calls.Load() != 3 || b.lastPair.Load() != "access-2/secret-2" {
		t.Errorf("Wrong match. expect: a new client for the new access key, got: %d builds, pair %v", b.calls.Load(), b.lastPair.Load())
	}

	// going back to the first pair builds again: only the last pair is kept
	p.set("access-1", "secret-1", nil)
	if c5, _ := h.pollyClient(ctx); c5 == c4 || b.calls.Load() != 4 {
		t.Errorf("Wrong match. expect: a build for the old pair, got: %d builds", b.calls.Load())
	}
}

// Two empty keys build a client and never return a nil client.
func Test_pollyClient_emptyPair(t *testing.T) {
	b := &countingBuilder{endpoint: "http://127.0.0.1:1"}
	h := &audioHandler{awsCreds: func(context.Context) (string, string, error) { return "", "", nil }, awsBuild: b.build}

	c1, err := h.pollyClient(context.Background())
	if err != nil || c1 == nil {
		t.Fatalf("Wrong match. expect: a client, got: %v %v", c1, err)
	}
	if c2, _ := h.pollyClient(context.Background()); c2 != c1 || b.calls.Load() != 1 {
		t.Errorf("Wrong match. expect: the same client, got: %d builds", b.calls.Load())
	}
}

func Test_pollyClient_nilCreds(t *testing.T) {
	h := &audioHandler{}

	c, err := h.pollyClient(context.Background())
	if err == nil || c != nil {
		t.Errorf("Wrong match. expect: an error and no client, got: %v %v", c, err)
	}
}

// A nil builder uses the real constructor, and nothing panics.
func Test_pollyClient_nilBuild(t *testing.T) {
	h := &audioHandler{awsCreds: func(context.Context) (string, string, error) { return "fake-access", "fake-secret", nil }}

	c, err := h.pollyClient(context.Background())
	if err != nil || c == nil {
		t.Errorf("Wrong match. expect: a client, got: %v %v", c, err)
	}
}

func Test_pollyClient_buildError(t *testing.T) {
	errBuild := errors.New("broken aws config")
	h := &audioHandler{
		awsCreds: func(context.Context) (string, string, error) { return "a", "s", nil },
		awsBuild: func(context.Context, string, string) (*polly.Client, error) { return nil, errBuild },
	}

	c, err := h.pollyClient(context.Background())
	if !errors.Is(err, errBuild) || c != nil {
		t.Errorf("Wrong match. expect: the build error and no client, got: %v %v", c, err)
	}
}

func Test_pollyClient_suppliedError(t *testing.T) {
	errSupply := errors.New("keys not readable")
	b := &countingBuilder{endpoint: "http://127.0.0.1:1"}
	h := &audioHandler{awsCreds: func(context.Context) (string, string, error) { return "", "", errSupply }, awsBuild: b.build}

	if c, err := h.pollyClient(context.Background()); !errors.Is(err, errSupply) || c != nil || b.calls.Load() != 0 {
		t.Errorf("Wrong match. expect: the supplier error and no build, got: %v %v %d", c, err, b.calls.Load())
	}
}

func Test_pollyClient_concurrent(t *testing.T) {
	b := &countingBuilder{endpoint: "http://127.0.0.1:1"}
	p := &pairSupplier{access: "a", secret: "s"}
	h := &audioHandler{awsCreds: p.get, awsBuild: b.build}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%8 == 0 {
				p.set("a", "s"+string(rune('0'+i%5)), nil)
			}
			if c, err := h.pollyClient(context.Background()); err != nil || c == nil {
				t.Errorf("Wrong match. expect: a client, got: %v %v", c, err)
			}
		}(i)
	}
	wg.Wait()
}

// A key change between two requests is used by the second request: the request carries the new
// key pair (checked at the fake server through the signature header).
func Test_awsAudioCreate_keyChangeNextRequest(t *testing.T) {
	var mu sync.Mutex
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auths = append(auths, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write([]byte{1, 2, 3, 4})
	}))
	defer srv.Close()

	p := &pairSupplier{access: "first-access", secret: "first-secret"}
	h := &audioHandler{awsCreds: p.get, awsBuild: func(_ context.Context, access, secret string) (*polly.Client, error) {
		return newLocalPollyClientWithKeys(srv.URL, access, secret), nil
	}}
	callID := uuid.FromStringOrNil("9e4f3000-0000-4000-8000-000000000002")

	for _, pair := range [][2]string{{"first-access", "first-secret"}, {"second-access", "second-secret"}} {
		p.set(pair[0], pair[1], nil)
		path := filepath.Join(t.TempDir(), "speech.wav")
		if err := h.awsAudioCreate(context.Background(), callID, "<speak>hi</speak>", "en-US", "Joanna", path); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(auths) != 2 || !strings.Contains(auths[0], "Credential=first-access/") || !strings.Contains(auths[1], "Credential=second-access/") {
		t.Errorf("Wrong match. expect: the key of each request in its signature, got: %v", auths)
	}
}

// A failed read stops the request before any call, and the error holds no key text.
func Test_awsAudioCreate_supplierError(t *testing.T) {
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called.Store(true) }))
	defer srv.Close()

	const secretText = "sentinel-secret-text"
	h := newLocalPollyHandler(srv.URL)
	h.awsCreds = func(context.Context) (string, string, error) {
		return "", "", errors.New("the TTS provider key could not be read")
	}

	err := h.awsAudioCreate(context.Background(), uuid.Nil, "<speak>hi</speak>", "en-US", "Joanna", filepath.Join(t.TempDir(), "x.wav"))
	if err == nil || called.Load() || strings.Contains(err.Error(), secretText) {
		t.Errorf("Wrong match. expect: an error without a call, got: %v (called: %v)", err, called.Load())
	}
}
