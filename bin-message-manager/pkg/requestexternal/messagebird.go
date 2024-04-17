package requestexternal

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/messagebird"
)

// MessagebirdSendMessage sends request to the messagebird to send the message.
func (h *requestExternal) MessagebirdSendMessage(sender string, destinations []string, text string) (*messagebird.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "MessagebirdSendMessage",
		"sender":       sender,
		"destinations": destinations,
		"text":         text,
	})

	uri := "https://rest.messagebird.com/messages"

	data := url.Values{}
	recipents := strings.Join(destinations, ",")
	data.Set("recipients", recipents)
	data.Set("originator", sender)
	data.Set("body", text)

	t := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		// We use ABSURDLY large keys, and should probably not.
		TLSHandshakeTimeout: 60 * time.Second,
	}

	client := &http.Client{
		Transport: t,
	}
	r, err := http.NewRequest("POST", uri, strings.NewReader(data.Encode())) // URL-encoded payload
	if err != nil {
		log.Errorf("Could not create a request. err: %v", err)
		return nil, errors.Wrap(err, "Could not create a request.")
	}

	r.Header.Add("Authorization", messagebirdAuth)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	response, err := client.Do(r)
	if err != nil {
		log.Errorf("Could not send the request. err: %v", err)
		return nil, errors.Wrap(err, "Could not send the request.")
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Could not receive the response correctly. err: %v", err)
		return nil, errors.Wrap(err, "Could not receive the response correctly.")
	}

	res := messagebird.Message{}
	if errParse := json.Unmarshal(body, &res); errParse != nil {
		log.Errorf("Could not parse the response. err: %v", errParse)
	}

	return &res, nil
}
