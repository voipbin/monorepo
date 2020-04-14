package rpc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	ariAddr    string //The asterisk-proxy connects to this asterisk ari service address
	ariAccount string //The asterisk-proxy uses this asterisk ari account info. id:password
)

// logger for global
// var log = logrus.New()

// Request defines RPC message request
type Request struct {
	URL      string `json:"url"`
	Method   string `json:"method"`
	DataType string `json:"data_type"`
	Data     string `json:"data,omitempty"`
}

// Response defines RPC message response
type Response struct {
	StatusCode int    `json:"status_code"`
	Data       string `json:"data"`
}

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
func parse(req []byte) (*Request, error) {
	res := &Request{}
	err := json.Unmarshal(req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// sendRequest sends a request to the Asterisk ARI.
func (m *Request) sendRequest() (int, string, error) {
	url := fmt.Sprintf("http://%s%s", ariAddr, m.URL)
	log.WithFields(log.Fields{
		"request": m,
	}).Debug("Sending ARI request.")

	req, err := http.NewRequest(m.Method, url, strings.NewReader(m.Data))
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
func RequestHandler(request string) (string, error) {
	// parse
	m, err := parse([]byte(request))
	if err != nil {
		return "", err
	}

	// send the request to Asterisk
	statusCode, resData, err := m.sendRequest()
	if err != nil {
		return "", err
	}

	response := Response{
		statusCode,
		resData,
	}

	res, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(res), nil
}
