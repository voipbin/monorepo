package requestexternal

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // the signature of the Twilio callbacks is HMAC-SHA1 by the definition of Twilio.
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"monorepo/bin-message-manager/models/twilio"
)

const providerTwilio = "twilio"

// twilioAPIBase is the base URL of the Twilio REST API; a variable so that a test can point it to a local server.
var twilioAPIBase = "https://api.twilio.com"

// TwilioConfig supplies the Twilio settings at the time of use. A nil function means not configured, so the
// zero value is a valid "Twilio is not configured".
type TwilioConfig struct {
	// Credentials returns the Account SID and the Auth Token. Both are empty when they are not configured.
	Credentials func(ctx context.Context) (sid string, token string, err error)

	// CallbackURL returns the valid status callback URL, or an empty string when there is none.
	CallbackURL func(ctx context.Context) string
}

// errors of the check of a Twilio callback. None of them holds a value of the request.
var (
	// ErrTwilioNotConfigured means that the Auth Token or the callback URL is not set, so no callback can be verified.
	ErrTwilioNotConfigured = errors.New("twilio callback verification is not configured")

	// ErrTwilioSignatureMissing means that the request has no signature.
	ErrTwilioSignatureMissing = errors.New("twilio callback signature is missing")

	// ErrTwilioSignatureInvalid means that the signature does not match.
	ErrTwilioSignatureInvalid = errors.New("twilio callback signature is invalid")
)

// credentials returns the Twilio credentials. ok is false when they are not configured.
func (h *requestExternal) twilioCredentials(ctx context.Context) (sid string, token string, ok bool, err error) {
	if h.twilio.Credentials == nil {
		return "", "", false, nil
	}

	sid, token, err = h.twilio.Credentials(ctx)
	if err != nil {
		return "", "", false, err
	}

	return sid, token, sid != "" && token != "", nil
}

// TwilioAvailable tells whether Twilio sending is configured: the Account SID and the Auth Token are both set.
func (h *requestExternal) TwilioAvailable(ctx context.Context) bool {
	_, _, ok, err := h.twilioCredentials(ctx)

	return err == nil && ok
}

// TwilioSendSMS sends one sms through Twilio.
func (h *requestExternal) TwilioSendSMS(ctx context.Context, source string, destination string, text string) (*twilio.MessageResponse, error) {
	return h.twilioSend(ctx, source, destination, text, nil, false)
}

// TwilioSendMMS sends one mms through Twilio. A text only mms (no media) is sent as an mms with SendAsMms.
func (h *requestExternal) TwilioSendMMS(ctx context.Context, source string, destination string, text string, mediaURLs []string) (*twilio.MessageResponse, error) {
	return h.twilioSend(ctx, source, destination, text, mediaURLs, len(mediaURLs) == 0)
}

func (h *requestExternal) twilioSend(ctx context.Context, source string, destination string, text string, mediaURLs []string, asMMS bool) (*twilio.MessageResponse, error) {
	sid, token, ok, errCred := h.twilioCredentials(ctx)
	if errCred != nil {
		return nil, fmt.Errorf("%w: twilio: %w", ErrTokenUnavailable, errCred)
	}
	if !ok {
		return nil, fmt.Errorf("%w: twilio: not configured", ErrTokenUnavailable)
	}

	data := url.Values{}
	data.Set("To", destination)
	data.Set("From", source)
	if text != "" {
		data.Set("Body", text)
	}
	for _, mediaURL := range mediaURLs {
		data.Add("MediaUrl", mediaURL)
	}
	if asMMS {
		data.Set("SendAsMms", "true")
	}
	if h.twilio.CallbackURL != nil {
		if callback := h.twilio.CallbackURL(ctx); callback != "" {
			data.Set("StatusCallback", callback)
		}
	}

	// the Account SID is a part of the path, so it is escaped.
	uri := twilioAPIBase + "/2010-04-01/Accounts/" + url.PathEscape(sid) + "/Messages.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, errWithoutURL("could not create a request", err)
	}
	req.SetBasicAuth(sid, token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// the error of the http client holds the request URL, and the path of the URL holds the Account SID.
		return nil, errWithoutURL("could not send the request", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ProviderError{Provider: providerTwilio, StatusCode: resp.StatusCode, Detail: providerErrorDetail(resp.Body)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errWithoutURL("could not read the response", err)
	}

	var res twilio.MessageResponse
	if errParse := json.Unmarshal(body, &res); errParse != nil {
		return nil, pkgerrors.Wrapf(errParse, "could not parse the response")
	}
	if res.SID == "" {
		return nil, fmt.Errorf("the response has no message id")
	}

	return &res, nil
}

// errWithoutURL wraps the error of a request with the text, and replaces a *url.Error by its cause so that the
// URL of the request (it holds the Account SID) is not in the text.
func errWithoutURL(text string, err error) error {
	var ue *url.Error
	if errors.As(err, &ue) && ue.Err != nil {
		return pkgerrors.Wrap(ue.Err, text)
	}

	return pkgerrors.Wrap(err, text)
}

// TwilioValidateCallback checks the signature of a Twilio status callback. The signed string is the callback URL
// setting, as it was given to Twilio with the request, followed by every form key (in byte order) and its values.
// It returns nil only for a valid signature. It never returns a value of the request.
func (h *requestExternal) TwilioValidateCallback(ctx context.Context, signature string, form url.Values) error {
	_, token, ok, err := h.twilioCredentials(ctx)
	callback := ""
	if h.twilio.CallbackURL != nil {
		callback = h.twilio.CallbackURL(ctx)
	}
	if err != nil || !ok || callback == "" {
		return ErrTwilioNotConfigured
	}

	if signature == "" {
		return ErrTwilioSignatureMissing
	}

	expect := twilioSignature(token, callback, form)
	if !hmac.Equal([]byte(expect), []byte(signature)) {
		return ErrTwilioSignatureInvalid
	}

	return nil
}

// twilioSignature computes the signature of a callback: base64(HMAC-SHA1(token, url + key1 + value1 + key2 + ...)).
func twilioSignature(token string, callbackURL string, form url.Values) string {
	keys := make([]string, 0, len(form))
	for k := range form {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(callbackURL)
	for _, k := range keys {
		values := append([]string(nil), form[k]...)
		sort.Strings(values)
		for _, v := range values {
			sb.WriteString(k)
			sb.WriteString(v)
		}
	}

	mac := hmac.New(sha1.New, []byte(token))
	mac.Write([]byte(sb.String()))

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
