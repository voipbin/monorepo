package listenhandler

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
	"monorepo/bin-common-handler/models/sock"
)

var regUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// processCommandPost handles POST /v1/commands.
// Routes process management commands (exec/kill) to the process manager,
// and all other commands to the RTPEngine NG protocol client.
func (h *listenHandler) processCommandPost(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processCommandPost")

	var cmd map[string]interface{}
	if err := json.Unmarshal(m.Data, &cmd); err != nil {
		log.WithError(err).Debug("Failed to unmarshal command")
		return simpleResponse(400), nil
	}

	// Check if this is a process management command (exec/kill).
	if typeStr, ok := cmd["type"].(string); ok {
		switch typeStr {
		case "exec":
			return h.processExec(cmd)
		case "kill":
			return h.processKill(cmd)
		}
	}

	// Otherwise, it's an NG protocol command (query, offer, answer, delete, etc.).
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

// processExec handles exec commands by starting a tcpdump process via the process manager.
func (h *listenHandler) processExec(cmd map[string]interface{}) (*sock.Response, error) {
	log := logrus.WithField("func", "processExec")

	id, ok := cmd["id"].(string)
	if !ok || id == "" || !regUUID.MatchString(id) {
		log.Debug("Missing or invalid required field: id (must be UUID)")
		return simpleResponse(400), nil
	}

	command, ok := cmd["command"].(string)
	if !ok || command == "" {
		log.Debug("Missing or invalid required field: command")
		return simpleResponse(400), nil
	}

	var params []string
	if rawParams, ok := cmd["parameters"].([]interface{}); ok {
		for _, p := range rawParams {
			s, ok := p.(string)
			if !ok {
				log.Debug("Non-string parameter in parameters array")
				return simpleResponse(400), nil
			}
			params = append(params, s)
		}
	}

	log.WithFields(logrus.Fields{"id": id, "command": command}).Debug("Starting process")

	if err := h.procMgr.Exec(id, command, params); err != nil {
		log.WithError(err).Warn("Exec failed")
		errData, _ := json.Marshal(map[string]string{
			"result":       "error",
			"error-reason": fmt.Sprintf("%v", err),
		})
		return &sock.Response{StatusCode: 500, Data: json.RawMessage(errData)}, nil
	}

	data, _ := json.Marshal(map[string]string{"result": "ok"})
	return &sock.Response{StatusCode: 200, Data: json.RawMessage(data)}, nil
}

// processKill handles kill commands by stopping a running process via the process manager.
func (h *listenHandler) processKill(cmd map[string]interface{}) (*sock.Response, error) {
	log := logrus.WithField("func", "processKill")

	id, ok := cmd["id"].(string)
	if !ok || id == "" || !regUUID.MatchString(id) {
		log.Debug("Missing or invalid required field: id (must be UUID)")
		return simpleResponse(400), nil
	}

	log.WithField("id", id).Debug("Killing process")

	pcapPath, err := h.procMgr.Kill(id)
	if err != nil {
		log.WithError(err).Warn("Kill failed")
		errData, _ := json.Marshal(map[string]string{
			"result":       "error",
			"error-reason": fmt.Sprintf("%v", err),
		})
		return &sock.Response{StatusCode: 500, Data: json.RawMessage(errData)}, nil
	}

	data, _ := json.Marshal(map[string]interface{}{
		"result":    "ok",
		"pcap_path": pcapPath,
	})
	return &sock.Response{StatusCode: 200, Data: json.RawMessage(data)}, nil
}
