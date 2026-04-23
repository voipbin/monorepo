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

type credentialConnectionResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (c *telnyxClient) CreateCredentialConnection(ctx context.Context, name string) (string, error) {
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return "", fmt.Errorf("marshal create connection request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/credential_connections",
		bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build create connection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create connection request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidKey
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("telnyx create connection returned unexpected status %d", resp.StatusCode)
	}

	var res credentialConnectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode create connection response: %w", err)
	}
	return res.Data.ID, nil
}

func (c *telnyxClient) DeleteCredentialConnection(ctx context.Context, connID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/credential_connections/%s", c.baseURL, connID), nil)
	if err != nil {
		return fmt.Errorf("build delete connection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete connection request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("telnyx delete connection returned unexpected status %d", resp.StatusCode)
	}
	return nil
}
