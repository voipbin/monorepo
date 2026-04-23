package telnyxclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

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

func (c *telnyxClient) CreateOutboundVoiceProfile(ctx context.Context, name string) (string, error) {
	body, err := json.Marshal(map[string]string{
		"name":         name,
		"traffic_type": "conversational",
		"service_plan": "global",
	})
	if err != nil {
		return "", fmt.Errorf("marshal create voice profile request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/outbound_voice_profiles",
		bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build create voice profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create voice profile request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidKey
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("telnyx create voice profile returned unexpected status %d", resp.StatusCode)
	}

	var res idResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode create voice profile response: %w", err)
	}
	return res.Data.ID, nil
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

func (c *telnyxClient) CreateIPConnection(ctx context.Context, name string, profileID string) (string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"connection_name": name,
		"outbound": map[string]string{
			"outbound_voice_profile_id": profileID,
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal create ip connection request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ip_connections",
		bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build create ip connection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create ip connection request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidKey
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("telnyx create ip connection returned unexpected status %d", resp.StatusCode)
	}

	var res idResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode create ip connection response: %w", err)
	}
	return res.Data.ID, nil
}

func (c *telnyxClient) DeleteIPConnection(ctx context.Context, connID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/ip_connections/%s", c.baseURL, connID), nil)
	if err != nil {
		return fmt.Errorf("build delete ip connection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete ip connection request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("telnyx delete ip connection returned unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *telnyxClient) RegisterIP(ctx context.Context, connID string, ipAddress string, port int) (string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"connection_id": connID,
		"ip_address":    ipAddress,
		"port":          port,
	})
	if err != nil {
		return "", fmt.Errorf("marshal register ip request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ips",
		bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build register ip request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("register ip request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidKey
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("telnyx register ip returned unexpected status %d", resp.StatusCode)
	}

	var res idResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode register ip response: %w", err)
	}
	return res.Data.ID, nil
}

func (c *telnyxClient) DeleteIP(ctx context.Context, ipID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/ips/%s", c.baseURL, ipID), nil)
	if err != nil {
		return fmt.Errorf("build delete ip request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete ip request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("telnyx delete ip returned unexpected status %d", resp.StatusCode)
	}
	return nil
}
