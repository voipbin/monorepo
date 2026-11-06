package messagehandler

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"monorepo/bin-message-manager/models/telnyx"
)

// newTestMediaServer starts a TLS server whose certificate is valid for telnyx.com and media.telnyx.com and returns
// the server and a client transport that sends every connection to it, whatever the address, and trusts only its
// certificate. No connection leaves the machine.
func newTestMediaServer(t *testing.T, handler http.Handler) (*httptest.Server, *http.Transport) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "media.telnyx.com"},
		DNSNames:     []string{"media.telnyx.com", "telnyx.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	pool := x509.NewCertPool()
	pool.AddCert(cert)

	srvAddr := srv.Listener.Addr().String()
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network string, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, srvAddr)
		},
		TLSClientConfig:    &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		DisableKeepAlives:  true,
		DisableCompression: true,
	}
	t.Cleanup(tr.CloseIdleConnections)

	return srv, tr
}

func newTestFetcher(tr *http.Transport) *mediaFetcher {
	return &mediaFetcher{
		client:          newMediaHTTPClient(tr),
		allowedSuffixes: []string{"telnyx.com"},
		maxFileSize:     1024,
		fileTimeout:     2 * time.Second,
	}
}

func Test_mediaHostAllowed(t *testing.T) {
	tests := []struct {
		host   string
		expect bool
	}{
		{"telnyx.com", true},
		{"media.telnyx.com", true},
		{"a.b.telnyx.com", true},
		{"MEDIA.TELNYX.COM", true},
		{"media.telnyx.com.", true},
		{"evil-telnyx.com", false},
		{"telnyx.com.example.net", false},
		{"xtelnyx.com", false},
		{"telnyx.com.evil", false},
		{"", false},
		{"127.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := mediaHostAllowed(tt.host, []string{"telnyx.com"}); got != tt.expect {
				t.Errorf("Wrong match. host: %q, expect: %v, got: %v", tt.host, tt.expect, got)
			}
		})
	}
}

func Test_parseMediaURL(t *testing.T) {
	tests := []struct {
		raw    string
		expect bool
	}{
		{"https://media.telnyx.com/example/photo.jpg", true},
		{"https://media.telnyx.com:443/example/photo.jpg", true},
		{"https://media.telnyx.com/example/photo.jpg?token=abc", true},
		{"http://media.telnyx.com/example/photo.jpg", false},
		{"https://media.telnyx.com:8443/example/photo.jpg", false},
		{"https://user:pass@media.telnyx.com/example/photo.jpg", false},
		{"https://user@media.telnyx.com/example/photo.jpg", false},
		{"https://media.telnyx.com.example.net/photo.jpg", false},
		{"https://example.net/media.telnyx.com", false},
		{"ftp://media.telnyx.com/photo.jpg", false},
		{"//media.telnyx.com/photo.jpg", false},
		{"mailto:media@telnyx.com", false},
		{"https:media.telnyx.com", false},
		{"", false},
		{"://bad", false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if _, got := parseMediaURL(tt.raw, []string{"telnyx.com"}); got != tt.expect {
				t.Errorf("Wrong match. url: %q, expect: %v, got: %v", tt.raw, tt.expect, got)
			}
		})
	}
}

func Test_isPublicAddress(t *testing.T) {
	tests := []struct {
		addr   string
		expect bool
	}{
		{"8.8.8.8", true},
		{"93.184.216.34", true},
		{"2606:4700:4700::1111", true},
		{"127.0.0.1", false},
		{"::1", false},
		{"10.1.2.3", false},
		{"172.16.0.1", false},
		{"172.31.255.255", false},
		{"192.168.1.1", false},
		{"169.254.169.254", false},
		{"169.254.0.1", false},
		{"0.0.0.0", false},
		{"0.1.2.3", false},
		{"::", false},
		{"100.64.0.1", false},
		{"100.127.255.255", false},
		{"192.0.0.1", false},
		{"192.0.2.1", false},
		{"192.88.99.1", false},
		{"::ffff:100.64.0.1", false},
		{"::ffff:198.18.0.1", false},
		{"::ffff:192.0.0.1", false},
		{"198.18.0.1", false},
		{"198.51.100.1", false},
		{"203.0.113.1", false},
		{"224.0.0.1", false},
		{"255.255.255.255", false},
		{"240.0.0.1", false},
		{"::ffff:127.0.0.1", false},
		{"::ffff:10.0.0.1", false},
		{"::ffff:8.8.8.8", true},
		{"64:ff9b::7f00:1", false},
		{"64:ff9b:1::1", false},
		{"2002:7f00:1::1", false},
		{"2001:0:4136:e378:8000:63bf:3fff:fdd2", false},
		{"2001:db8::1", false},
		{"fc00::1", false},
		{"fd12:3456::1", false},
		{"fe80::1", false},
		{"fec0::1", false},
		{"ff02::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			if got := isPublicAddress(netip.MustParseAddr(tt.addr)); got != tt.expect {
				t.Errorf("Wrong match. addr: %s, expect: %v, got: %v", tt.addr, tt.expect, got)
			}
		})
	}
}

func Test_publicAddressControl(t *testing.T) {
	if err := publicAddressControl("tcp", "8.8.8.8:443", nil); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	for _, addr := range []string{"127.0.0.1:443", "[::1]:443", "10.0.0.1:443", "not-an-address", "media.telnyx.com:443", ""} {
		if err := publicAddressControl("tcp", addr, nil); err == nil {
			t.Errorf("Wrong match. address: %q, expect: refused, got: ok", addr)
		}
	}
}

func Test_parseMediaContentType(t *testing.T) {
	tests := []struct {
		in        string
		expectCT  string
		expectExt string
		expectOK  bool
	}{
		{"image/jpeg", "image/jpeg", ".jpg", true},
		{"IMAGE/PNG", "image/png", ".png", true},
		{"image/gif", "image/gif", ".gif", true},
		{"audio/mpeg", "audio/mpeg", ".mp3", true},
		{"audio/x-wav", "audio/x-wav", ".wav", true},
		{"audio/wave", "audio/wave", ".wav", true},
		{"application/pdf; charset=binary", "application/pdf", ".pdf", true},
		{"text/x-vcard", "text/x-vcard", ".vcf", true},
		{"text/html", "", "", false},
		{"application/octet-stream", "", "", false},
		{"image/svg+xml", "", "", false},
		{"video/mp4", "", "", false},
		{"", "", "", false},
		{"not a type", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			ct, ext, ok := parseMediaContentType(tt.in)
			if ok != tt.expectOK || ct != tt.expectCT || ext != tt.expectExt {
				t.Errorf("Wrong match. expect: %q %q %v, got: %q %q %v", tt.expectCT, tt.expectExt, tt.expectOK, ct, ext, ok)
			}
		})
	}
}

func Test_NewMediaFetcher_wiring(t *testing.T) {
	f, ok := NewMediaFetcher().(*mediaFetcher)
	if !ok {
		t.Fatalf("Wrong match. expect: *mediaFetcher")
	}

	if len(f.allowedSuffixes) != 1 || f.allowedSuffixes[0] != "telnyx.com" {
		t.Errorf("Wrong match. allowed suffixes: %v", f.allowedSuffixes)
	}
	if f.maxFileSize != 2*1024*1024 || f.fileTimeout != 20*time.Second {
		t.Errorf("Wrong match. limits: %d, %v", f.maxFileSize, f.fileTimeout)
	}
	if inboundMediaMaxTotalSize != 5*1024*1024 || inboundMediaMaxCount != 10 || inboundMediaMaxIngest != 2 || inboundMediaIngestTimeout != 90*time.Second {
		t.Errorf("Wrong match. the documented constants changed")
	}

	if f.client.CheckRedirect == nil {
		t.Errorf("Wrong match. expect: redirects are refused")
	}
	tr, ok := f.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Wrong match. expect: *http.Transport")
	}
	if tr.Proxy != nil || !tr.DisableKeepAlives || !tr.DisableCompression || tr.TLSClientConfig != nil {
		t.Errorf("Wrong match. transport: proxy set: %v, keepalive off: %v, compression off: %v, tls config: %v", tr.Proxy != nil, tr.DisableKeepAlives, tr.DisableCompression, tr.TLSClientConfig)
	}
}

// Test_productionTransport_refusesLoopback proves that the production dial control refuses a server on the
// loopback address, the one the other tests have to reach through a test dialer.
func Test_productionTransport_refusesLoopback(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Wrong match. the request must never arrive")
	}))
	defer srv.Close()

	client := newMediaHTTPClient(newMediaTransport(publicAddressControl))
	resp, err := client.Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatalf("Wrong match. expect: refused, got: ok")
	}
	if !strings.Contains(err.Error(), errNotPublicAddress.Error()) {
		t.Errorf("Wrong match. expect: %v, got: %v", errNotPublicAddress, err)
	}
}

func Test_mediaFetcher_Fetch(t *testing.T) {
	jpeg := []byte("fake-jpeg-content")

	tests := []struct {
		name string

		media  telnyx.PayloadMedia
		budget int64

		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)

		expectResult  FetchResult
		expectContent []byte
		expectName    string
		expectCalls   int32
	}{
		{
			name:   "ok",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/example/photo.jpg", ContentType: "image/jpeg", Size: int64(len(jpeg))},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
					t.Errorf("Wrong match. no credential may be sent: %v", r.Header)
				}
				if r.Header.Get("Accept-Encoding") != "identity" {
					t.Errorf("Wrong match. expect: identity, got: %q", r.Header.Get("Accept-Encoding"))
				}
				if r.Method != http.MethodGet || r.URL.Path != "/example/photo.jpg" {
					t.Errorf("Wrong match. got: %s %s", r.Method, r.URL.Path)
				}
				_, _ = w.Write(jpeg)
			},
			expectResult:  FetchResultOK,
			expectContent: jpeg,
			expectName:    "media-3.jpg",
			expectCalls:   1,
		},
		{
			name:   "size unknown is accepted",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "application/pdf"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(jpeg)
			},
			expectResult:  FetchResultOK,
			expectContent: jpeg,
			expectName:    "media-3.pdf",
			expectCalls:   1,
		},
		{
			name:   "content type not allowed, no request",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "text/html"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
			},
			expectResult: FetchResultType,
		},
		{
			name:         "content type missing, no request",
			media:        telnyx.PayloadMedia{URL: "https://media.telnyx.com/a"},
			budget:       5000,
			handler:      func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			expectResult: FetchResultType,
		},
		{
			name:         "host not allowed, no request",
			media:        telnyx.PayloadMedia{URL: "https://media.example.net/a", ContentType: "image/jpeg"},
			budget:       5000,
			handler:      func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			expectResult: FetchResultHost,
		},
		{
			name:         "scheme not allowed, no request",
			media:        telnyx.PayloadMedia{URL: "http://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget:       5000,
			handler:      func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			expectResult: FetchResultHost,
		},
		{
			name:         "declared size over the file limit, no request",
			media:        telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg", Size: 1025},
			budget:       5000,
			handler:      func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			expectResult: FetchResultSize,
		},
		{
			name:         "declared size over the budget, no request",
			media:        telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg", Size: 600},
			budget:       500,
			handler:      func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			expectResult: FetchResultSize,
		},
		{
			name:         "no budget left, no request",
			media:        telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget:       0,
			handler:      func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			expectResult: FetchResultSize,
		},
		{
			name:   "content length over the limit",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg", Size: 10},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "2048")
				w.WriteHeader(http.StatusOK)
			},
			expectResult: FetchResultSize,
			expectCalls:  1,
		},
		{
			name:   "body over the limit without a length",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg", Size: 10},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Transfer-Encoding", "chunked")
				_, _ = w.Write(bytes.Repeat([]byte("a"), 512))
				w.(http.Flusher).Flush()
				_, _ = w.Write(bytes.Repeat([]byte("a"), 2048))
			},
			expectResult: FetchResultSize,
			expectCalls:  1,
		},
		{
			name:   "body exactly at the limit",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(bytes.Repeat([]byte("a"), 1024))
			},
			expectResult:  FetchResultOK,
			expectContent: bytes.Repeat([]byte("a"), 1024),
			expectName:    "media-3.jpg",
			expectCalls:   1,
		},
		{
			name:   "empty body",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectResult: FetchResultStatus,
			expectCalls:  1,
		},
		{
			name:   "redirect is not followed",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/a" {
					t.Errorf("Wrong match. the redirect was followed to %s", r.URL.Path)
				}
				http.Redirect(w, r, "https://media.telnyx.com/b", http.StatusFound)
			},
			expectResult: FetchResultStatus,
			expectCalls:  1,
		},
		{
			name:   "401",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			expectResult: FetchResultAuth,
			expectCalls:  1,
		},
		{
			name:   "403",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			expectResult: FetchResultAuth,
			expectCalls:  1,
		},
		{
			name:   "404",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectResult: FetchResultStatus,
			expectCalls:  1,
		},
		{
			name:   "500",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectResult: FetchResultStatus,
			expectCalls:  1,
		},
		{
			name:   "compressed response",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Encoding", "gzip")
				_, _ = w.Write(jpeg)
			},
			expectResult: FetchResultStatus,
			expectCalls:  1,
		},
		{
			name:   "slow body",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = w.Write([]byte("a"))
				w.(http.Flusher).Flush()
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
				}
			},
			expectResult: FetchResultTimeout,
			expectCalls:  1,
		},
		{
			name:   "connection closed in the middle",
			media:  telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"},
			budget: 5000,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = w.Write([]byte("a"))
				w.(http.Flusher).Flush()
				panic(http.ErrAbortHandler)
			},
			expectResult: FetchResultNetwork,
			expectCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int32
			_, tr := newTestMediaServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&calls, 1)
				tt.handler(t, w, r)
			}))
			f := newTestFetcher(tr)
			f.fileTimeout = 500 * time.Millisecond

			res, result := f.Fetch(context.Background(), tt.media, 3, tt.budget)
			if result != tt.expectResult {
				t.Fatalf("Wrong match. expect: %s, got: %s", tt.expectResult, result)
			}
			if got := atomic.LoadInt32(&calls); got != tt.expectCalls {
				t.Errorf("Wrong match. expect calls: %d, got: %d", tt.expectCalls, got)
			}

			if tt.expectResult != FetchResultOK {
				if res != nil {
					t.Errorf("Wrong match. expect: nil media, got: %v", res)
				}
				return
			}
			if !bytes.Equal(res.Content, tt.expectContent) || res.Filename != tt.expectName {
				t.Errorf("Wrong match. expect: %q %s, got: %q %s", tt.expectContent, tt.expectName, res.Content, res.Filename)
			}
		})
	}
}

// countingTransport counts the bytes the client reads from the response bodies.
type countingTransport struct {
	next  http.RoundTripper
	count atomic.Int64
}

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := c.next.RoundTrip(r)
	if err == nil {
		resp.Body = &countingBody{ReadCloser: resp.Body, count: &c.count}
	}
	return resp, err
}

type countingBody struct {
	io.ReadCloser
	count *atomic.Int64
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.count.Add(int64(n))
	return n, err
}

// Test_mediaFetcher_Fetch_readsAtMostTheLimit checks that the client stops reading the body at the limit plus one
// byte, however much the server sends: the memory bound does not depend on the declared length.
func Test_mediaFetcher_Fetch_readsAtMostTheLimit(t *testing.T) {
	_, tr := newTestMediaServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Transfer-Encoding", "chunked")
		chunk := bytes.Repeat([]byte("a"), 512)
		for i := 0; i < 200; i++ {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			w.(http.Flusher).Flush()
		}
	}))
	ct := &countingTransport{next: tr}
	f := newTestFetcher(tr)
	f.client = newMediaHTTPClient(&http.Transport{})
	f.client.Transport = ct

	media := telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"}
	if _, result := f.Fetch(context.Background(), media, 0, 5000); result != FetchResultSize {
		t.Fatalf("Wrong match. expect: size, got: %s", result)
	}
	if got := ct.count.Load(); got > f.maxFileSize+1 {
		t.Errorf("Wrong match. expect: at most %d bytes read, got: %d", f.maxFileSize+1, got)
	}
}

// Test_mediaFetcher_Fetch_unreachable checks that a refused connection is a network error and that the context of
// the caller wins over the file timeout.
func Test_mediaFetcher_Fetch_unreachable(t *testing.T) {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network string, _ string) (net.Conn, error) {
			return nil, errNotPublicAddress
		},
	}
	f := newTestFetcher(tr)

	media := telnyx.PayloadMedia{URL: "https://media.telnyx.com/a", ContentType: "image/jpeg"}
	if _, result := f.Fetch(context.Background(), media, 0, 5000); result != FetchResultNetwork {
		t.Errorf("Wrong match. expect: network, got: %s", result)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, result := f.Fetch(ctx, media, 0, 5000); result != FetchResultTimeout {
		t.Errorf("Wrong match. expect: timeout, got: %s", result)
	}
}
