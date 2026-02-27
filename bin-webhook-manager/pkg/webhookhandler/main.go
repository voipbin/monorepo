package webhookhandler

//go:generate mockgen -package webhookhandler -destination ./mock_webhookhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
)

// WebhookHandler is interface for webhook handle
type WebhookHandler interface {
	SendWebhookToCustomer(ctx context.Context, customerID uuid.UUID, dataType webhook.DataType, data json.RawMessage) error
	SendWebhookToURI(ctx context.Context, customerID uuid.UUID, uri string, method webhook.MethodType, dataType webhook.DataType, data json.RawMessage) error
}

// webhookHandler structure for service handle
type webhookHandler struct {
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	accoutHandler accounthandler.AccountHandler

	httpClient *http.Client
}

// newSafeHTTPClient returns an *http.Client hardened against SSRF.
func newSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address %s: %w", addr, err)
			}

			// Resolve DNS and check each IP against the block list (prevents DNS rebinding).
			ips, err := net.LookupIP(host)
			if err != nil {
				return nil, fmt.Errorf("cannot resolve %s: %w", host, err)
			}

			for _, ip := range ips {
				if isPrivateIP(ip) {
					return nil, fmt.Errorf("connection to private/reserved IP blocked: %s -> %s", host, ip)
				}
			}

			// Dial the first resolved public IP.
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   5,
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects (max 3)")
			}
			if err := validateWebhookURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect target blocked: %w", err)
			}
			return nil
		},
	}
}

// NewWebhookHandler returns new webhook handler
func NewWebhookHandler(db dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, messageTargetHandler accounthandler.AccountHandler) WebhookHandler {

	h := &webhookHandler{
		db:            db,
		notifyHandler: notifyHandler,

		accoutHandler: messageTargetHandler,

		httpClient: newSafeHTTPClient(),
	}

	return h
}
