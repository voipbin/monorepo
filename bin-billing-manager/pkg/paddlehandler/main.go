package paddlehandler

//go:generate mockgen -package paddlehandler -destination mock_main.go -source main.go

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"monorepo/bin-billing-manager/models/account"

	"github.com/sirupsen/logrus"
)

const (
	defaultPaddleBaseURL = "https://api.paddle.com"
	defaultTimeout       = 10 * time.Second
)

// PaddleHandler defines the interface for Paddle API operations.
type PaddleHandler interface {
	CreatePortalSession(ctx context.Context, paddleCustomerID string) (string, error)
	GetPlanTypeByPriceID(priceID string) (account.PlanType, error)

	// VerifyWebhook checks the signature header of a Paddle webhook against the body that was
	// received. It returns nil only when the signature is valid. ErrSignatureInvalid means that the
	// signature was rejected, ErrCredentialNotConfigured means that no webhook secret is configured
	// (the webhook is rejected), and ErrKeyUnavailable means that the secret could not be read.
	VerifyWebhook(ctx context.Context, signature string, body []byte) error
}

// KeyFunc hands a Paddle credential to the handler at the time of use. An empty value with no
// error means "not configured" (the callers answer with ErrCredentialNotConfigured). An error
// means that the credential could not be read, and it is never replaced by an empty value.
type KeyFunc = func(ctx context.Context) (string, error)

type paddleHandler struct {
	apiKey        KeyFunc
	webhookSecret KeyFunc
	baseURL       string
	httpClient    *http.Client
	priceMap      map[string]account.PlanType
}

// NewPaddleHandler creates a new PaddleHandler. The two credentials are functions that are called
// for every use, so a changed key is used by the next call and nothing is kept in the handler.
func NewPaddleHandler(apiKey KeyFunc, webhookSecret KeyFunc, priceIDBasic string, priceIDProfessional string) PaddleHandler {
	priceMap := make(map[string]account.PlanType)
	if priceIDBasic != "" {
		priceMap[priceIDBasic] = account.PlanTypeBasic
	}
	if priceIDProfessional != "" {
		priceMap[priceIDProfessional] = account.PlanTypeProfessional
	}

	return &paddleHandler{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		baseURL:       defaultPaddleBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		priceMap: priceMap,
	}
}

// portalSessionRequest is the request body for Paddle's customer portal sessions API.
type portalSessionRequest struct {
	CustomerID string `json:"customer_id"`
}

// portalSessionResponse is the response from Paddle's customer portal sessions API.
type portalSessionResponse struct {
	Data struct {
		ID   string `json:"id"`
		URLs struct {
			General struct {
				Overview string `json:"overview"`
			} `json:"general"`
		} `json:"urls"`
	} `json:"data"`
	Error *struct {
		Type   string `json:"type"`
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"error"`
}

// CreatePortalSession calls Paddle API to generate a customer portal session URL.
func (h *paddleHandler) CreatePortalSession(ctx context.Context, paddleCustomerID string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "CreatePortalSession",
		"paddle_customer_id": paddleCustomerID,
	})

	if paddleCustomerID == "" {
		return "", fmt.Errorf("paddle_customer_id is empty")
	}

	// The key is read for this call only, and a key that can not be read stops the call before
	// any request leaves: it is never replaced by an empty key or by another source.
	apiKey, err := h.readKey(ctx, h.apiKey)
	if err != nil {
		return "", err
	}
	if apiKey == "" {
		return "", fmt.Errorf("%w: api key", ErrCredentialNotConfigured)
	}

	reqBody := portalSessionRequest{
		CustomerID: paddleCustomerID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("could not marshal request: %w", err)
	}

	url := h.baseURL + "/customer-portal-sessions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Infof("Calling Paddle API for portal session. paddle_customer_id: %s", paddleCustomerID)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("paddle API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	const maxResponseSize = 1 << 20 // 1 MB
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", fmt.Errorf("could not read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		log.Errorf("Paddle API returned non-201 status. status: %d, body: %s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("paddle API returned status %d", resp.StatusCode)
	}

	var portalResp portalSessionResponse
	if err := json.Unmarshal(respBody, &portalResp); err != nil {
		return "", fmt.Errorf("could not parse portal session response: %w", err)
	}

	portalURL := portalResp.Data.URLs.General.Overview
	if portalURL == "" {
		return "", fmt.Errorf("paddle returned empty portal URL")
	}

	log.Infof("Portal session created. paddle_customer_id: %s", paddleCustomerID)
	return portalURL, nil
}

// GetPlanTypeByPriceID returns the plan type for the given Paddle price ID.
func (h *paddleHandler) GetPlanTypeByPriceID(priceID string) (account.PlanType, error) {
	if priceID == "" {
		return "", fmt.Errorf("price_id is empty")
	}

	planType, ok := h.priceMap[priceID]
	if !ok {
		return "", fmt.Errorf("unknown price_id: %s", priceID)
	}

	return planType, nil
}
