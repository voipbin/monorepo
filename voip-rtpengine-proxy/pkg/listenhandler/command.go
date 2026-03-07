package listenhandler

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"monorepo/bin-common-handler/models/sock"
)

// processCommandPost handles POST /v1/commands.
// Translates the JSON body to an NG protocol command and returns the response.
func (h *listenHandler) processCommandPost(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processCommandPost")

	var cmd map[string]interface{}
	if err := json.Unmarshal(m.Data, &cmd); err != nil {
		log.WithError(err).Debug("Failed to unmarshal command")
		return simpleResponse(400), nil
	}

	cmdStr, ok := cmd["command"].(string)
	if !ok || cmdStr == "" {
		log.Debug("Missing or invalid required field: command")
		return simpleResponse(400), nil
	}

	log.WithField("command", cmd["command"]).Debug("Sending NG command")

	result, err := h.ngClient.Send(cmd)
	if err != nil {
		log.WithError(err).Warn("NG command failed")
		errData, _ := json.Marshal(map[string]string{
			"result":       "error",
			"error-reason": fmt.Sprintf("%v", err),
		})
		return &sock.Response{StatusCode: 500, Data: json.RawMessage(errData)}, nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		log.WithError(err).Error("Failed to marshal NG response")
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, Data: json.RawMessage(data)}, nil
}
