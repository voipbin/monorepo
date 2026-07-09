package providerhandler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/telnyxclient"
)

const telnyxSIPHostname = "sip.telnyx.com"

// generateCredentialSecret returns a random alphanumeric string suitable for
// a Telnyx FQDN connection's SIP credential user_name/password. Telnyx
// rejects punctuation/whitespace in these fields ("Must contain only letters
// and numbers; no spacing allowed" — confirmed empirically against the
// Telnyx API), so the alphabet is restricted to letters and digits.
func generateCredentialSecret(n int) (string, error) {
	// base64 URL-safe encoding without padding only uses [A-Za-z0-9_-];
	// stripping '_'/'-' leaves a purely alphanumeric string. Over-request
	// bytes so the stripped result still has at least n characters.
	buf := make([]byte, n*2)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(buf)
	out := make([]byte, 0, n)
	for i := 0; i < len(encoded) && len(out) < n; i++ {
		c := encoded[i]
		if c == '_' || c == '-' {
			continue
		}
		out = append(out, c)
	}
	if len(out) < n {
		return "", fmt.Errorf("could not generate %d alphanumeric characters", n)
	}
	return string(out), nil
}

// sanitizeCredentialUserName strips everything but letters/digits from name
// so it is safe to use as (part of) a Telnyx FQDN connection's user_name.
// Telnyx rejects punctuation/whitespace in user_name (e.g. hyphens, spaces),
// which provider names commonly contain (confirmed empirically: a request
// with a hyphenated user_name returns 422 "Must contain only letters and
// numbers; no spacing allowed").
func sanitizeCredentialUserName(name string) string {
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		}
	}
	return string(out)
}

// credentialUserNamePrefix is prepended to the sanitized provider name to
// form the Telnyx FQDN connection's user_name.
const credentialUserNamePrefix = "voipbinrm"

// telnyxUserNameMaxLen is Telnyx's hard limit on the FQDN connection
// user_name field (confirmed empirically: a 90-character user_name returns
// 422 "is too long (maximum is 32 characters)").
const telnyxUserNameMaxLen = 32

// buildCredentialUserName combines the fixed prefix with a sanitized,
// length-capped provider name so the result never exceeds Telnyx's 32
// character user_name limit regardless of how long the provider name is.
func buildCredentialUserName(name string) string {
	sanitized := sanitizeCredentialUserName(name)
	maxSanitizedLen := telnyxUserNameMaxLen - len(credentialUserNamePrefix)
	if maxSanitizedLen < 0 {
		maxSanitizedLen = 0
	}
	if len(sanitized) > maxSanitizedLen {
		sanitized = sanitized[:maxSanitizedLen]
	}
	return credentialUserNamePrefix + sanitized
}

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

	if h.sipGatewayFQDNForPSTN == "" {
		return nil, fmt.Errorf("no PSTN SIP gateway FQDN configured; set EXTERNAL_SIP_GATEWAY_FQDN_FOR_PSTN")
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

	// Step 3: create an FQDN connection linked to the voice profile. An FQDN
	// connection (as opposed to an IP connection) presents inbound calls with
	// the FQDN as the SIP request-URI domain instead of a raw IP address,
	// which Kamailio's domain validation (request_from_external_init_connection)
	// requires. Telnyx requires the connection's credential authentication to
	// be fully configured before an outbound_voice_profile can be attached, so
	// a random credential is generated here; it is never used for actual
	// authentication since inbound routing is FQDN/DNS-based and outbound
	// dialing uses IP-based tech-prefix auth, but Telnyx still requires the
	// fields to be populated.
	credUser := buildCredentialUserName(name)
	credPassword, err := generateCredentialSecret(32)
	if err != nil {
		log.Errorf("Could not generate Telnyx connection credential. err: %v", err)
		h.telnyxCleanup(log, client, "", "", profileID)
		return nil, fmt.Errorf("generate telnyx credential failed: %w", err)
	}
	connID, err := client.CreateFQDNConnection(ctx, name, profileID, credUser, credPassword)
	if err != nil {
		log.Errorf("Could not create Telnyx FQDN connection. err: %v", err)
		h.telnyxCleanup(log, client, "", "", profileID)
		return nil, fmt.Errorf("telnyx create fqdn connection failed: %w", err)
	}
	log.Debugf("Created Telnyx FQDN connection. conn_id: %s", connID)

	// Step 4: register our public PSTN SIP gateway FQDN on the connection.
	_, portStr, parseErr := net.SplitHostPort(h.sipGatewayFQDNForPSTN)
	if parseErr != nil {
		log.Errorf("Invalid PSTN SIP gateway FQDN. fqdn: %s, err: %v", h.sipGatewayFQDNForPSTN, parseErr)
		h.telnyxCleanup(log, client, "", connID, profileID)
		return nil, fmt.Errorf("invalid pstn sip gateway fqdn %q: %w", h.sipGatewayFQDNForPSTN, parseErr)
	}
	port, convErr := strconv.Atoi(portStr)
	if convErr != nil {
		log.Errorf("Invalid PSTN SIP gateway port. fqdn: %s, err: %v", h.sipGatewayFQDNForPSTN, convErr)
		h.telnyxCleanup(log, client, "", connID, profileID)
		return nil, fmt.Errorf("invalid pstn sip gateway port in %q: %w", h.sipGatewayFQDNForPSTN, convErr)
	}
	fqdnHost, _, _ := net.SplitHostPort(h.sipGatewayFQDNForPSTN)
	fqdnID, regErr := client.RegisterFQDN(ctx, connID, fqdnHost, port)
	if regErr != nil {
		log.Errorf("Could not register PSTN SIP gateway FQDN. fqdn: %s, err: %v", h.sipGatewayFQDNForPSTN, regErr)
		h.telnyxCleanup(log, client, "", connID, profileID)
		return nil, fmt.Errorf("telnyx register fqdn %q failed: %w", h.sipGatewayFQDNForPSTN, regErr)
	}
	log.Debugf("Registered PSTN SIP gateway FQDN. fqdn: %s, fqdn_id: %s", h.sipGatewayFQDNForPSTN, fqdnID)

	// Step 5: create the VoIPBin provider record
	res, err := h.Create(ctx, provider.TypeSIP, telnyxSIPHostname, "", "", map[string]string{}, name, detail, "")
	if err != nil {
		log.Errorf("Could not create provider record. Attempting Telnyx cleanup. err: %v", err)
		h.telnyxCleanup(log, client, fqdnID, connID, profileID)
		return nil, fmt.Errorf("provider create failed: %w", err)
	}

	// Step 6: store Telnyx resource IDs as metadata for future reference.
	// Failure here is non-fatal: setup already succeeded, the metadata is for
	// visibility/reference only, and forcing a rollback would orphan the
	// Telnyx-side resources we just created. Log and return the provider
	// record we already have from Step 5.
	metadata := map[string]interface{}{
		"telnyx_profile_id":    profileID,
		"telnyx_connection_id": connID,
		"telnyx_fqdn_id":       fqdnID,
		"telnyx_fqdn":          h.sipGatewayFQDNForPSTN,
	}
	if errMeta := h.db.ProviderUpdate(ctx, res.ID, map[provider.Field]any{
		provider.FieldMetadata: metadata,
	}); errMeta != nil {
		log.Warnf("Could not save Telnyx metadata on provider. provider_id: %s, err: %v", res.ID, errMeta)
		return res, nil
	}

	updated, err := h.db.ProviderGet(ctx, res.ID)
	if err != nil {
		log.Warnf("Could not re-fetch provider after metadata update. Returning pre-update record. err: %v", err)
		return res, nil
	}
	return updated, nil
}

// telnyxCleanup attempts to delete Telnyx resources created before a failure.
// Empty IDs are skipped. Errors are logged but not returned — the caller's
// original error takes precedence.
func (h *providerHandler) telnyxCleanup(log *logrus.Entry, client telnyxclient.TelnyxClient, fqdnID, connID, profileID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if fqdnID != "" {
		if err := client.DeleteFQDN(ctx, fqdnID); err != nil {
			log.Errorf("Telnyx FQDN cleanup failed. fqdn_id: %s, err: %v", fqdnID, err)
		} else {
			log.Infof("Telnyx FQDN deleted during cleanup. fqdn_id: %s", fqdnID)
		}
	}

	if connID != "" {
		if err := client.DeleteFQDNConnection(ctx, connID); err != nil {
			log.Errorf("Telnyx FQDN connection cleanup failed. conn_id: %s, err: %v", connID, err)
		} else {
			log.Infof("Telnyx FQDN connection deleted during cleanup. conn_id: %s", connID)
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
