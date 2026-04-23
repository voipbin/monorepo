package providerhandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/telnyxclient"
)

const telnyxSIPHostname = "sip.telnyx.com"

// Setup is the public entry point. It constructs a TelnyxClient from apiKey and delegates to setupWithClient.
func (h *providerHandler) Setup(ctx context.Context, carrier, name, detail, apiKey string) (*provider.Provider, error) {
	client := telnyxclient.NewTelnyxClient(apiKey)
	return h.setupWithClient(ctx, carrier, name, detail, client)
}

// setupWithClient is the testable core — accepts an injected TelnyxClient.
func (h *providerHandler) setupWithClient(ctx context.Context, carrier, name, detail string, client telnyxclient.TelnyxClient) (*provider.Provider, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "setupWithClient",
		"carrier": carrier,
		"name":    name,
	})

	if carrier != "telnyx" {
		return nil, fmt.Errorf("unsupported carrier: %s", carrier)
	}

	if err := client.ValidateKey(ctx); err != nil {
		log.Infof("Telnyx key validation failed. err: %v", err)
		return nil, err
	}

	connID, err := client.CreateCredentialConnection(ctx, name)
	if err != nil {
		log.Errorf("Could not create Telnyx credential connection. err: %v", err)
		return nil, fmt.Errorf("telnyx create connection failed: %w", err)
	}
	log.Debugf("Created Telnyx credential connection. conn_id: %s", connID)

	// IP-based auth: only the SIP hostname is stored — no credentials.
	res, err := h.Create(ctx, provider.TypeSIP, telnyxSIPHostname, "", "", map[string]string{}, name, detail)
	if err != nil {
		log.Errorf("Could not create provider record. Attempting Telnyx cleanup. err: %v", err)
		if cleanupErr := client.DeleteCredentialConnection(ctx, connID); cleanupErr != nil {
			log.Errorf("Telnyx cleanup failed. conn_id: %s, err: %v", connID, cleanupErr)
		} else {
			log.Infof("Telnyx connection deleted during cleanup. conn_id: %s", connID)
		}
		return nil, fmt.Errorf("provider create failed: %w", err)
	}

	return res, nil
}

// isTelnyxInvalidKey reports whether err is a Telnyx key validation failure.
func isTelnyxInvalidKey(err error) bool {
	return errors.Is(err, telnyxclient.ErrInvalidKey)
}
