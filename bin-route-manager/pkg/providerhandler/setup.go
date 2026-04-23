package providerhandler

import (
	"context"
	"fmt"
	"time"

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

	// Step 1: validate the API key
	if err := client.ValidateKey(ctx); err != nil {
		log.Infof("Telnyx key validation failed. err: %v", err)
		return nil, err
	}

	// Step 2: create outbound voice profile
	profileID, err := client.CreateOutboundVoiceProfile(ctx, name)
	if err != nil {
		log.Errorf("Could not create Telnyx outbound voice profile. err: %v", err)
		return nil, fmt.Errorf("telnyx create voice profile failed: %w", err)
	}
	log.Debugf("Created Telnyx outbound voice profile. profile_id: %s", profileID)

	// Step 3: create IP connection linked to the voice profile
	connID, err := client.CreateIPConnection(ctx, name, profileID)
	if err != nil {
		log.Errorf("Could not create Telnyx IP connection. err: %v", err)
		h.telnyxCleanup(log, client, "", "", profileID)
		return nil, fmt.Errorf("telnyx create ip connection failed: %w", err)
	}
	log.Debugf("Created Telnyx IP connection. conn_id: %s", connID)

	// Step 4: register our SIP load balancer IP on the connection
	ipID, err := client.RegisterIP(ctx, connID, h.sipLBIP, h.sipLBPort)
	if err != nil {
		log.Errorf("Could not register SIP LB IP on Telnyx connection. ip: %s, err: %v", h.sipLBIP, err)
		h.telnyxCleanup(log, client, "", connID, profileID)
		return nil, fmt.Errorf("telnyx register ip failed: %w", err)
	}
	log.Debugf("Registered SIP LB IP on Telnyx connection. ip_id: %s", ipID)

	// Step 5: create the VoIPBin provider record
	res, err := h.Create(ctx, provider.TypeSIP, telnyxSIPHostname, "", "", map[string]string{}, name, detail)
	if err != nil {
		log.Errorf("Could not create provider record. Attempting Telnyx cleanup. err: %v", err)
		h.telnyxCleanup(log, client, ipID, connID, profileID)
		return nil, fmt.Errorf("provider create failed: %w", err)
	}

	return res, nil
}

// telnyxCleanup attempts to delete Telnyx resources created before a failure.
// Empty IDs are skipped. Errors are logged but not returned — the caller's
// original error takes precedence.
func (h *providerHandler) telnyxCleanup(log *logrus.Entry, client telnyxclient.TelnyxClient, ipID, connID, profileID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if ipID != "" {
		if err := client.DeleteIP(ctx, ipID); err != nil {
			log.Errorf("Telnyx IP cleanup failed. ip_id: %s, err: %v", ipID, err)
		} else {
			log.Infof("Telnyx IP deleted during cleanup. ip_id: %s", ipID)
		}
	}

	if connID != "" {
		if err := client.DeleteIPConnection(ctx, connID); err != nil {
			log.Errorf("Telnyx IP connection cleanup failed. conn_id: %s, err: %v", connID, err)
		} else {
			log.Infof("Telnyx IP connection deleted during cleanup. conn_id: %s", connID)
		}
	}

	if profileID != "" {
		if err := client.DeleteOutboundVoiceProfile(ctx, profileID); err != nil {
			log.Errorf("Telnyx voice profile cleanup failed. profile_id: %s, err: %v", profileID, err)
		} else {
			log.Infof("Telnyx voice profile deleted during cleanup. profile_id: %s", profileID)
		}
	}
}
