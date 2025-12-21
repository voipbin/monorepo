package listenhandler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"monorepo/bin-common-handler/models/sock"
)

// ariSendRequestToAsterisk sends the request to the Asterisk's ARI.
// returns status_code, response_message, error
func (h *listenHandler) ariSendRequestToAsterisk(m *sock.Request) (int, []byte, error) {
	url := fmt.Sprintf("http://%s%s", h.ariAddr, m.URI)

	req, err := http.NewRequest(string(m.Method), url, bytes.NewReader(m.Data))
	if err != nil {
		return 0, nil, err
	}

	// basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(h.ariAccount))
	req.Header.Add("Authorization", "Basic "+auth)

	// content-type
	if m.DataType != "" {
		req.Header.Set("Content-Type", m.DataType)
	}

	// send
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	res, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, res, nil
}

func (h *listenHandler) listenHandlerARI(request *sock.Request) (*sock.Response, error) {
	// send the request to Asterisk
	statusCode, resData, err := h.ariSendRequestToAsterisk(request)
	if err != nil {
		return nil, err
	}

	response := &sock.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       resData,
	}

	return response, nil
}
