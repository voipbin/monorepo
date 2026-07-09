package telnyxclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// allTelnyxDestinations is the full set of ISO 3166-1 alpha-2 codes we request,
// including overseas territories and special regions with PSTN service.
// Telnyx will reject codes the account has not been approved for; those are
// stripped and the request is retried automatically.
var allTelnyxDestinations = []string{
	"AD", "AE", "AF", "AG", "AL", "AM", "AO", "AR", "AT", "AU", "AW", "AZ",
	"BA", "BB", "BD", "BE", "BF", "BG", "BH", "BI", "BJ", "BL", "BN", "BO",
	"BQ", "BR", "BS", "BT", "BW", "BY", "BZ", "CA", "CC", "CD", "CF", "CG",
	"CH", "CI", "CK", "CL", "CM", "CN", "CO", "CR", "CU", "CV", "CW", "CX",
	"CY", "CZ", "DE", "DJ", "DK", "DM", "DO", "DZ", "EC", "EE", "EG", "EH",
	"ER", "ES", "ET", "FI", "FJ", "FK", "FM", "FO", "FR", "GA", "GB", "GD",
	"GE", "GF", "GG", "GH", "GI", "GL", "GM", "GN", "GP", "GQ", "GR", "GT",
	"GU", "GW", "GY", "HK", "HN", "HR", "HT", "HU", "ID", "IE", "IL", "IM",
	"IN", "IQ", "IR", "IS", "IT", "JE", "JM", "JO", "JP", "KE", "KG", "KH",
	"KI", "KM", "KN", "KP", "KR", "KW", "KY", "KZ", "LA", "LB", "LC", "LI",
	"LK", "LR", "LS", "LT", "LU", "LV", "LY", "MA", "MC", "MD", "ME", "MF",
	"MG", "MH", "MK", "ML", "MM", "MN", "MO", "MP", "MQ", "MR", "MS", "MT",
	"MU", "MV", "MW", "MX", "MY", "MZ", "NA", "NC", "NE", "NF", "NG", "NI",
	"NL", "NO", "NP", "NR", "NU", "NZ", "OM", "PA", "PE", "PF", "PG", "PH",
	"PK", "PL", "PM", "PR", "PS", "PT", "PW", "PY", "QA", "RE", "RO", "RS",
	"RU", "RW", "SA", "SB", "SC", "SD", "SE", "SG", "SH", "SI", "SJ", "SK",
	"SL", "SM", "SN", "SO", "SR", "SS", "ST", "SV", "SX", "SY", "SZ", "TC",
	"TD", "TG", "TH", "TJ", "TK", "TL", "TM", "TN", "TO", "TR", "TT", "TV",
	"TW", "TZ", "UA", "UG", "US", "UY", "UZ", "VA", "VC", "VE", "VG", "VI",
	"VN", "VU", "WF", "WS", "XK", "YE", "YT", "ZA", "ZM", "ZW",
}

func (c *telnyxClient) ValidateKey(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/whoami", nil)
	if err != nil {
		return fmt.Errorf("build whoami request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("whoami request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrInvalidKey
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telnyx whoami returned unexpected status %d", resp.StatusCode)
	}
	return nil
}

type idResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

type telnyxErrorResponse struct {
	Errors []struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func (c *telnyxClient) CreateOutboundVoiceProfile(ctx context.Context, name string) (string, error) {
	destinations := make([]string, len(allTelnyxDestinations))
	copy(destinations, allTelnyxDestinations)

	for {
		id, rejected, err := c.tryCreateOutboundVoiceProfile(ctx, name, destinations)
		if err != nil {
			return "", err
		}
		if id != "" {
			return id, nil
		}
		// Remove rejected countries and retry.
		keep := destinations[:0]
		rejectedSet := make(map[string]bool, len(rejected))
		for _, r := range rejected {
			rejectedSet[r] = true
		}
		for _, d := range destinations {
			if !rejectedSet[d] {
				keep = append(keep, d)
			}
		}
		if len(keep) == len(destinations) {
			return "", fmt.Errorf("telnyx create voice profile failed with unapproved countries but none could be identified")
		}
		destinations = keep
	}
}

// tryCreateOutboundVoiceProfile attempts creation. Returns (id, nil, nil) on success,
// ("", rejectedCodes, nil) when Telnyx rejects specific country codes, or ("", nil, err) on other failures.
func (c *telnyxClient) tryCreateOutboundVoiceProfile(ctx context.Context, name string, destinations []string) (string, []string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"name":                     name,
		"traffic_type":             "conversational",
		"service_plan":             "global",
		"whitelisted_destinations": destinations,
	})
	if err != nil {
		return "", nil, fmt.Errorf("marshal create voice profile request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/outbound_voice_profiles",
		bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("build create voice profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("create voice profile request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", nil, ErrInvalidKey
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		var res idResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return "", nil, fmt.Errorf("decode create voice profile response: %w", err)
		}
		return res.Data.ID, nil, nil
	}
	if resp.StatusCode == http.StatusUnprocessableEntity {
		respBody, _ := io.ReadAll(resp.Body)
		var errResp telnyxErrorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil {
			for _, e := range errResp.Errors {
				if rejected := parseRejectedCountries(e.Detail); len(rejected) > 0 {
					return "", rejected, nil
				}
			}
		}
		return "", nil, fmt.Errorf("telnyx create voice profile unprocessable: %s", string(respBody))
	}
	return "", nil, fmt.Errorf("telnyx create voice profile returned unexpected status %d", resp.StatusCode)
}

// parseRejectedCountries extracts 2-letter country codes from a Telnyx error detail like:
// "You must have your account approved ... to use the following countries: US, CA, ..."
func parseRejectedCountries(detail string) []string {
	const marker = "following countries: "
	idx := strings.Index(detail, marker)
	if idx == -1 {
		return nil
	}
	parts := strings.Split(detail[idx+len(marker):], ", ")
	codes := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) == 2 {
			codes = append(codes, p)
		}
	}
	return codes
}

func (c *telnyxClient) DeleteOutboundVoiceProfile(ctx context.Context, profileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/outbound_voice_profiles/%s", c.baseURL, profileID), nil)
	if err != nil {
		return fmt.Errorf("build delete voice profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete voice profile request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("telnyx delete voice profile returned unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *telnyxClient) CreateFQDNConnection(ctx context.Context, name, profileID, userName, password string) (string, error) {
	// Telnyx requires an FQDN connection to have credential authentication
	// fully configured (user_name + password) before an outbound_voice_profile
	// can be attached — confirmed empirically against the Telnyx API (a PATCH
	// setting only outbound_voice_profile_id on a fresh FQDN connection with no
	// credentials returns 422 "must be fully configured before assigning an
	// outbound profile"). All three are therefore sent together in a single
	// create call.
	body, err := json.Marshal(map[string]interface{}{
		"connection_name": name,
		"user_name":       userName,
		"password":        password,
		"inbound": map[string]string{
			// dnis_number_format must match the format VoIPBin stores numbers
			// in (E.164 with a leading '+'). The Telnyx default ("e164", no
			// leading '+') causes number-manager lookups for the called
			// number to silently return zero results.
			"ani_number_format":  "+E.164",
			"dnis_number_format": "+e164",
		},
		"outbound": map[string]string{
			"outbound_voice_profile_id": profileID,
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal create fqdn connection request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/fqdn_connections",
		bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build create fqdn connection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create fqdn connection request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidKey
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("telnyx create fqdn connection returned status %d: %s", resp.StatusCode, string(errBody))
	}

	var res idResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode create fqdn connection response: %w", err)
	}
	return res.Data.ID, nil
}

func (c *telnyxClient) DeleteFQDNConnection(ctx context.Context, connID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/fqdn_connections/%s", c.baseURL, connID), nil)
	if err != nil {
		return fmt.Errorf("build delete fqdn connection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete fqdn connection request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("telnyx delete fqdn connection returned unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *telnyxClient) RegisterFQDN(ctx context.Context, connID string, fqdn string, port int) (string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"connection_id":   connID,
		"fqdn":            fqdn,
		"port":            port,
		"dns_record_type": "a",
	})
	if err != nil {
		return "", fmt.Errorf("marshal register fqdn request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/fqdns",
		bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build register fqdn request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("register fqdn request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidKey
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("telnyx register fqdn returned status %d: %s", resp.StatusCode, string(errBody))
	}

	var res idResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode register fqdn response: %w", err)
	}
	return res.Data.ID, nil
}

func (c *telnyxClient) DeleteFQDN(ctx context.Context, fqdnID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/fqdns/%s", c.baseURL, fqdnID), nil)
	if err != nil {
		return fmt.Errorf("build delete fqdn request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete fqdn request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("telnyx delete fqdn returned unexpected status %d", resp.StatusCode)
	}
	return nil
}
