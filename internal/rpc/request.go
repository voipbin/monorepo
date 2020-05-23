package rpc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/rabbitmq"
)

var (
	ariAddr    string //The asterisk-proxy connects to this asterisk ari service address
	ariAccount string //The asterisk-proxy uses this asterisk ari account info. id:password
)

func init() {
	ariAddr = "localhost:8088"
	ariAccount = "asterisk:asterisk"
}

// Initiate initiates rpc package
func Initiate(addr, account string) {
	ariAddr = addr
	ariAccount = account

	log.Debugf("Initiated rpc.")
}

// parse parses []byte to Message
func parse(req []byte) (*rabbitmq.Request, error) {
	res := &rabbitmq.Request{}
	err := json.Unmarshal(req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// sendRequest sends a request to the Asterisk ARI.
func sendRequest(m *rabbitmq.Request) (int, string, error) {
	url := fmt.Sprintf("http://%s%s", ariAddr, m.URI)
	log.WithFields(log.Fields{
		"request": m,
	}).Debug("Sending ARI request.")

	req, err := http.NewRequest(string(m.Method), url, strings.NewReader(m.Data))
	if err != nil {
		return 0, "", err
	}

	// basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(ariAccount))
	req.Header.Add("Authorization", "Basic "+auth)

	// content-type
	if m.DataType != "" {
		req.Header.Set("Content-Type", m.DataType)
	}

	// send
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}

	res, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, "", err
	}

	return resp.StatusCode, string(res), nil
}

// RequestHandler handles RPC request message
func RequestHandler(request *rabbitmq.Request) (*rabbitmq.Response, error) {
	// send the request to Asterisk
	// statusCode, resData, err := sendRequest(m)
	statusCode, resData, err := sendRequest(request)
	if err != nil {
		return nil, err
	}

	response := &rabbitmq.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       resData,
	}

	return response, nil
}
