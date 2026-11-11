package requestexternal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/messagebird"
)

// messagebirdMessagesURI is the endpoint of the sms request; a variable so that a test can point it to a local server.
var messagebirdMessagesURI = "https://rest.messagebird.com/messages"

// MessagebirdSendMessage sends request to the messagebird to send the message.
func (h *requestExternal) MessagebirdSendMessage(ctx context.Context, sender string, destinations []string, text string) (*messagebird.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "MessagebirdSendMessage",
		"recipients": len(destinations),
	})

	token, errToken := h.authtokenMessagebird(ctx)
	if errToken != nil {
		log.Errorf("Could not get the messagebird token. err: %v", errToken)
		return nil, fmt.Errorf("%w: messagebird: %w", ErrTokenUnavailable, errToken)
	}

	uri := messagebirdMessagesURI

	data := url.Values{}
	recipents := strings.Join(destinations, ",")
	data.Set("recipients", recipents)
	data.Set("originator", sender)
	data.Set("body", text)

	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// We use ABSURDLY large keys, and should probably not.
		TLSHandshakeTimeout: 60 * time.Second,
	}

	client := &http.Client{
		Transport: t,
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, strings.NewReader(data.Encode())) // URL-encoded payload
	if err != nil {
		log.Errorf("Could not create a request. err: %v", err)
		return nil, errors.Wrap(err, "Could not create a request.")
	}

	authtoken := fmt.Sprintf("AccessKey %s", token)
	r.Header.Add("Authorization", authtoken)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	response, err := client.Do(r)
	if err != nil {
		log.Errorf("Could not send the request. err: %v", err)
		return nil, errors.Wrap(err, "Could not send the request.")
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		pe := &ProviderError{Provider: providerMessagebird, StatusCode: response.StatusCode, Detail: providerErrorDetail(response.Body)}
		log.Errorf("Could not send the message request. err: %v", pe)
		return nil, pe
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Could not receive the response correctly. err: %v", err)
		return nil, errors.Wrap(err, "Could not receive the response correctly.")
	}

	res := messagebird.Message{}
	if errParse := json.Unmarshal(body, &res); errParse != nil {
		log.Errorf("Could not parse the response. err: %v", errParse)
		return nil, errors.Wrapf(errParse, "could not parse the response correctly")
	}

	return &res, nil
}
