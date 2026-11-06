package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_media_fetcher.go -source media_fetcher.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/telnyx"
)

// Limits of the media of a received message. The values are chosen small on purpose: the whole media is held in
// memory while it is downloaded and while it is sent to the storage-manager.
const (
	inboundMediaMaxCount     = 10
	inboundMediaMaxFileSize  = int64(2 * 1024 * 1024)
	inboundMediaMaxTotalSize = int64(5 * 1024 * 1024)
	inboundMediaFileTimeout  = 20 * time.Second
)

// mediaAllowedHostSuffixes lists the hosts the media of a received message may be downloaded from: the host itself
// or any sub domain. The provider documentation shows media.telnyx.com only. This is an assumption: if the real
// host differs, the media is skipped (result host) and this line is the one to change.
var mediaAllowedHostSuffixes = []string{"telnyx.com"}

// mediaContentTypes lists the accepted content types with the extension of the stored file.
var mediaContentTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"audio/mpeg":      ".mp3",
	"audio/wav":       ".wav",
	"audio/x-wav":     ".wav",
	"audio/wave":      ".wav",
	"application/pdf": ".pdf",
	"text/vcard":      ".vcf",
	"text/x-vcard":    ".vcf",
}

// FetchResult is the closed set of outcomes of one media of a received message. It is the label of the counter.
type FetchResult string

// list of FetchResult
const (
	FetchResultOK       FetchResult = "ok"
	FetchResultStored   FetchResult = "stored"
	FetchResultParse    FetchResult = "parse"
	FetchResultType     FetchResult = "type"
	FetchResultHost     FetchResult = "host"
	FetchResultSize     FetchResult = "size"
	FetchResultAuth     FetchResult = "auth"
	FetchResultStatus   FetchResult = "status"
	FetchResultNetwork  FetchResult = "network"
	FetchResultTimeout  FetchResult = "timeout"
	FetchResultStorage  FetchResult = "storage"
	FetchResultOverflow FetchResult = "overflow"
	FetchResultBusy     FetchResult = "busy"
)

// FetchedMedia is the downloaded media of a received message.
type FetchedMedia struct {
	Content     []byte
	ContentType string
	Filename    string // generated name, never taken from the sender
}

// MediaFetcher downloads the media of a received message.
type MediaFetcher interface {
	// Fetch downloads the given media. budget is the number of bytes left for the whole message.
	// On success it returns the media and FetchResultOK, otherwise nil and the reason.
	Fetch(ctx context.Context, m telnyx.PayloadMedia, index int, budget int64) (*FetchedMedia, FetchResult)
}

type mediaFetcher struct {
	client          *http.Client
	allowedSuffixes []string
	maxFileSize     int64
	fileTimeout     time.Duration
}

// NewMediaFetcher returns the production MediaFetcher.
func NewMediaFetcher() MediaFetcher {
	return &mediaFetcher{
		client:          newMediaHTTPClient(newMediaTransport(publicAddressControl)),
		allowedSuffixes: mediaAllowedHostSuffixes,
		maxFileSize:     inboundMediaMaxFileSize,
		fileTimeout:     inboundMediaFileTimeout,
	}
}

func newMediaHTTPClient(transport *http.Transport) *http.Client {
	return &http.Client{
		Transport: transport,
		// a redirect is never followed: a 3xx answer is a failure.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// newMediaTransport returns the transport of the media downloads: no proxy, default TLS verification, no
// connection reuse, no decompression, and a dialer that refuses addresses by the given control.
func newMediaTransport(control func(network, address string, c syscall.RawConn) error) *http.Transport {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: control,
	}

	return &http.Transport{
		Proxy:                  nil,
		DialContext:            dialer.DialContext,
		TLSHandshakeTimeout:    10 * time.Second,
		ResponseHeaderTimeout:  15 * time.Second,
		MaxResponseHeaderBytes: 64 * 1024,
		DisableKeepAlives:      true,
		DisableCompression:     true,
		ForceAttemptHTTP2:      true,
	}
}

// errNotPublicAddress is returned by the dialer when the address that would be connected is not a public one.
var errNotPublicAddress = errors.New("the address is not a public address")

// blockedPrefixes lists the address ranges that are never connected, on top of the ones netip knows.
var blockedPrefixes = func() []netip.Prefix {
	list := []string{
		"0.0.0.0/8",
		"100.64.0.0/10",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.88.99.0/24",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"240.0.0.0/4",
		"64:ff9b::/96",
		"64:ff9b:1::/48",
		"2002::/16",
		"2001::/32",
		"2001:db8::/32",
		"fec0::/10",
	}

	res := make([]netip.Prefix, 0, len(list))
	for _, s := range list {
		res = append(res, netip.MustParsePrefix(s))
	}
	return res
}()

// isPublicAddress returns true if the given address is a public unicast address.
func isPublicAddress(addr netip.Addr) bool {
	addr = addr.Unmap()

	if !addr.IsValid() || addr.IsLoopback() || addr.IsPrivate() || addr.IsUnspecified() || addr.IsMulticast() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsInterfaceLocalMulticast() {
		return false
	}

	for _, p := range blockedPrefixes {
		if p.Contains(addr) {
			return false
		}
	}

	return true
}

// publicAddressControl is the Control of the dialer: it runs on the address that is actually connected, after the
// name resolution, so a name that resolves to an internal address and a DNS change between a check and the
// connection are both refused.
func publicAddressControl(network string, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return errNotPublicAddress
	}

	addr, err := netip.ParseAddr(host)
	if err != nil || !isPublicAddress(addr) {
		return errNotPublicAddress
	}

	return nil
}

// mediaHostAllowed returns true if the host is one of the suffixes or a sub domain of one.
func mediaHostAllowed(host string, suffixes []string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" {
		return false
	}

	for _, s := range suffixes {
		if host == s || strings.HasSuffix(host, "."+s) {
			return true
		}
	}

	return false
}

// parseMediaURL returns the url if it is an https url without user info, on the default port, of an allowed host.
func parseMediaURL(raw string, suffixes []string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" {
		return nil, false
	}

	if port := u.Port(); port != "" && port != "443" {
		return nil, false
	}

	if !mediaHostAllowed(u.Hostname(), suffixes) {
		return nil, false
	}

	return u, true
}

// parseMediaContentType returns the extension of the stored file for the given content type.
func parseMediaContentType(contentType string) (string, string, bool) {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", "", false
	}

	mt = strings.ToLower(mt)
	ext, ok := mediaContentTypes[mt]
	if !ok {
		return "", "", false
	}

	return mt, ext, true
}

// Fetch downloads the given media.
func (f *mediaFetcher) Fetch(ctx context.Context, m telnyx.PayloadMedia, index int, budget int64) (*FetchedMedia, FetchResult) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Fetch",
		"index": index,
	})

	contentType, ext, ok := parseMediaContentType(m.ContentType)
	if !ok {
		return nil, FetchResultType
	}

	u, ok := parseMediaURL(m.URL, f.allowedSuffixes)
	if !ok {
		// the host name is not a secret, the path and the query may be one. It is logged to be able to correct the
		// allow list.
		host := "unparsable"
		if tmp, err := url.Parse(m.URL); err == nil && tmp.Hostname() != "" {
			host = tmp.Hostname()
		}
		log.Warnf("The media is not on an allowed host or scheme. host: %s", host)
		return nil, FetchResultHost
	}
	log = log.WithField("host", u.Hostname())

	limit := f.maxFileSize
	if budget < limit {
		limit = budget
	}
	if limit <= 0 || m.Size > limit {
		return nil, FetchResultSize
	}

	ctx, cancel := context.WithTimeout(ctx, f.fileTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, FetchResultNetwork
	}
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, f.classifyError(ctx, log, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		log.Warnf("The media download needs authentication. status: %d", resp.StatusCode)
		return nil, FetchResultAuth
	case resp.StatusCode != http.StatusOK:
		log.Infof("The media download was refused. status: %d", resp.StatusCode)
		return nil, FetchResultStatus
	}

	if enc := resp.Header.Get("Content-Encoding"); enc != "" && !strings.EqualFold(enc, "identity") {
		return nil, FetchResultStatus
	}
	if resp.ContentLength > limit {
		return nil, FetchResultSize
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, f.classifyError(ctx, log, err)
	}
	if int64(len(content)) > limit {
		return nil, FetchResultSize
	}
	if len(content) == 0 {
		return nil, FetchResultStatus
	}

	return &FetchedMedia{
		Content:     content,
		ContentType: contentType,
		Filename:    fmt.Sprintf("media-%d%s", index, ext),
	}, FetchResultOK
}

func (f *mediaFetcher) classifyError(ctx context.Context, log *logrus.Entry, err error) FetchResult {
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) {
		return FetchResultTimeout
	}

	// the error text of the http client carries the url, so it is not logged.
	log.Warnf("Could not download the media. not_public_address: %v", errors.Is(err, errNotPublicAddress) || strings.Contains(err.Error(), errNotPublicAddress.Error()))
	return FetchResultNetwork
}
