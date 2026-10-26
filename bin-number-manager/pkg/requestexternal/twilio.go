package requestexternal

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"monorepo/bin-number-manager/pkg/requestexternal/models/twilio"
)

// twilioBaseURL is the base URL of the twilio REST API. Tests replace it.
var twilioBaseURL = "https://api.twilio.com"

// twilioHTTPClient never follows a redirect: a redirected POST could lose its body, and the
// answer is classified by the status as it is.
var twilioHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

const (
	twilioAPIVersion = "2010-04-01"

	// maxTwilioErrorBody is the most bytes read from an error answer.
	maxTwilioErrorBody = 4096

	// maxTwilioBody is the most bytes read from a success answer.
	maxTwilioBody = 1 << 20
)

// twilioErrorBody is the error answer of twilio.
type twilioErrorBody struct {
	Code int `json:"code"`
}

// twilioSend sends a request of the account and returns the body of an answer with one of the
// okStatus. Any other answer is an *HTTPStatusError. The error of the transport is returned
// without its URL, because the URL holds the account sid.
func (h *requestExternal) twilioSend(method, sid, token, path string, query, form url.Values, okStatus ...int) ([]byte, error) {
	uri := fmt.Sprintf("%s/%s/Accounts/%s/%s", twilioBaseURL, twilioAPIVersion, url.PathEscape(sid), path)
	if len(query) > 0 {
		uri = uri + "?" + query.Encode()
	}

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequestWithContext(context.Background(), method, uri, body)
	if err != nil {
		return nil, fmt.Errorf("could not create a http request")
	}
	req.SetBasicAuth(sid, token)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := twilioHTTPClient.Do(req)
	if err != nil {
		var urlErr *url.Error
		if stderrors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("could not get correct response. err: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	for _, ok := range okStatus {
		if resp.StatusCode == ok {
			res, errRead := io.ReadAll(io.LimitReader(resp.Body, maxTwilioBody))
			if errRead != nil {
				return nil, fmt.Errorf("could not read the response body. err: %w", errRead)
			}
			return res, nil
		}
	}

	statusErr := &HTTPStatusError{StatusCode: resp.StatusCode}
	if resp.StatusCode >= http.StatusBadRequest {
		if raw, errRead := io.ReadAll(io.LimitReader(resp.Body, maxTwilioErrorBody)); errRead == nil {
			parsed := twilioErrorBody{}
			if errParse := json.Unmarshal(raw, &parsed); errParse == nil {
				statusErr.ErrorCode = parsed.Code
			}
		}
	}

	return nil, statusErr
}

// TwilioAvailableNumberGets returns the local numbers of the country that can be purchased
// without an address.
func (h *requestExternal) TwilioAvailableNumberGets(sid, token, countryCode string, limit uint) ([]*twilio.AvailableNumber, error) {
	if countryCode == "" {
		return nil, fmt.Errorf("no country given")
	}

	if limit <= 0 || limit > 10 {
		limit = 10
	}

	query := url.Values{}
	query.Set("PageSize", fmt.Sprintf("%d", limit))
	query.Set("VoiceEnabled", "true")
	query.Set("ExcludeAllAddressRequired", "true")

	body, err := h.twilioSend(http.MethodGet, sid, token, fmt.Sprintf("AvailablePhoneNumbers/%s/Local.json", url.PathEscape(countryCode)), query, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}

	parsed := twilio.ResponseAvailableNumbers{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("could not parse the response. err: %w", err)
	}

	res := []*twilio.AvailableNumber{}
	for i := range parsed.AvailablePhoneNumbers {
		res = append(res, &parsed.AvailablePhoneNumbers[i])
	}

	return res, nil
}

// TwilioIncomingPhoneNumbersPost purchases the number. The trunk (if given) receives the calls of the number.
func (h *requestExternal) TwilioIncomingPhoneNumbersPost(sid, token, num, trunkSID string) (*twilio.IncomingPhoneNumber, error) {
	form := url.Values{}
	form.Set("PhoneNumber", num)
	if trunkSID != "" {
		form.Set("TrunkSid", trunkSID)
	}

	body, err := h.twilioSend(http.MethodPost, sid, token, "IncomingPhoneNumbers.json", nil, form, http.StatusCreated, http.StatusOK)
	if err != nil {
		return nil, err
	}

	res := &twilio.IncomingPhoneNumber{}
	if err := json.Unmarshal(body, res); err != nil {
		return nil, fmt.Errorf("could not parse the response. err: %w", err)
	}

	return res, nil
}

// TwilioIncomingPhoneNumbersIDDelete releases the number of the id (the sid of the number) from the account.
func (h *requestExternal) TwilioIncomingPhoneNumbersIDDelete(sid, token, id string) error {
	if _, err := h.twilioSend(http.MethodDelete, sid, token, fmt.Sprintf("IncomingPhoneNumbers/%s.json", url.PathEscape(id)), nil, nil, http.StatusNoContent, http.StatusOK); err != nil {
		return err
	}

	return nil
}
