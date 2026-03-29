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
}

type paddleHandler struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	priceMap   map[string]account.PlanType
}

// NewPaddleHandler creates a new PaddleHandler.
func NewPaddleHandler(apiKey string, priceIDBasic string, priceIDProfessional string) PaddleHandler {
	priceMap := make(map[string]account.PlanType)
	if priceIDBasic != "" {
		priceMap[priceIDBasic] = account.PlanTypeBasic
	}
	if priceIDProfessional != "" {
		priceMap[priceIDProfessional] = account.PlanTypeProfessional
	}

	return &paddleHandler{
		apiKey:  apiKey,
		baseURL: defaultPaddleBaseURL,
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
	req.Header.Set("Authorization", "Bearer "+h.apiKey)

	log.Infof("Calling Paddle API for portal session. paddle_customer_id: %s", paddleCustomerID)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("paddle API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
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
