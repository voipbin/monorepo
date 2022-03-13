package requestexternal

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/messagebird"
)

// MessagebirdSendMessage sends request to the messagebird to send the message.
func (h *requestExternal) MessagebirdSendMessage(sender string, destinations []string, text string) (*messagebird.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "MessagebirdSendMessage",
		"sender": sender,
	})

	uri := "https://rest.messagebird.com/messages"

	data := url.Values{}
	recipents := strings.Join(destinations, ",")
	data.Set("recipients", recipents)
	data.Set("originator", sender)
	data.Set("body", text)

	client := &http.Client{}
	r, err := http.NewRequest("POST", uri, strings.NewReader(data.Encode())) // URL-encoded payload
	if err != nil {
		log.Fatal(err)
	}

	r.Header.Add("Authorization", messagebirdAuth)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	response, err := client.Do(r)
	if err != nil {
		log.Errorf("Could not send the request to the messagebird. err: %v", err)
		return nil, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Could not receive the response correctly. err: %v", err)
		return nil, err
	}

	res := messagebird.Message{}
	if errParse := json.Unmarshal(body, &res); errParse != nil {
		log.Errorf("Could not parse the response. err: %v", errParse)
	}

	return &res, nil
}
