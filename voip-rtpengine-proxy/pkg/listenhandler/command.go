package listenhandler

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/voip-rtpengine-proxy/models/command"
)

var regUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// processCommandPost handles POST /v1/commands.
// Routes commands by type: ng to RTPEngine, exec/kill to the process manager.
func (h *listenHandler) processCommandPost(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processCommandPost")

	var raw map[string]interface{}
	if err := json.Unmarshal(m.Data, &raw); err != nil {
		log.WithError(err).Debug("Failed to unmarshal command")
		return simpleResponse(400), nil
	}

	typeStr, ok := raw["type"].(string)
	if !ok || typeStr == "" {
		log.Debug("Missing or invalid required field: type")
		return simpleResponse(400), nil
	}

	cmd := command.Command{
		Type: command.Type(typeStr),
		Data: raw,
	}

	// Extract optional typed fields.
	if id, ok := raw["id"].(string); ok {
		cmd.ID = id
	}
	if c, ok := raw["command"].(string); ok {
		cmd.Command = c
	}
	if rawParams, ok := raw["parameters"].([]interface{}); ok {
		for _, p := range rawParams {
			s, ok := p.(string)
			if !ok {
				log.Debug("Non-string parameter in parameters array")
				return simpleResponse(400), nil
			}
			cmd.Parameters = append(cmd.Parameters, s)
		}
	}

	switch cmd.Type {
	case command.TypeExec:
		return h.processExec(cmd)
	case command.TypeKill:
		return h.processKill(cmd)
	case command.TypeNG:
		return h.processNG(cmd)
	default:
		log.WithField("type", typeStr).Debug("Unknown command type")
		return simpleResponse(400), nil
	}
}

// processExec handles exec commands by starting a tcpdump process via the process manager.
func (h *listenHandler) processExec(cmd command.Command) (*sock.Response, error) {
	log := logrus.WithField("func", "processExec")

	if cmd.ID == "" || !regUUID.MatchString(cmd.ID) {
		log.Debug("Missing or invalid required field: id (must be UUID)")
		return simpleResponse(400), nil
	}

	if cmd.Command == "" {
		log.Debug("Missing or invalid required field: command")
		return simpleResponse(400), nil
	}

	log.WithFields(logrus.Fields{"id": cmd.ID, "command": cmd.Command}).Debug("Starting process")

	if err := h.procMgr.Exec(cmd.ID, cmd.Command, cmd.Parameters); err != nil {
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
func (h *listenHandler) processKill(cmd command.Command) (*sock.Response, error) {
	log := logrus.WithField("func", "processKill")

	if cmd.ID == "" || !regUUID.MatchString(cmd.ID) {
		log.Debug("Missing or invalid required field: id (must be UUID)")
		return simpleResponse(400), nil
	}

	log.WithField("id", cmd.ID).Debug("Killing process")

	pcapPath, err := h.procMgr.Kill(cmd.ID)
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

// processNG handles NG protocol commands by forwarding them to RTPEngine.
func (h *listenHandler) processNG(cmd command.Command) (*sock.Response, error) {
	log := logrus.WithField("func", "processNG")

	if cmd.Command == "" {
		log.Debug("Missing or invalid required field: command")
		return simpleResponse(400), nil
	}

	log.WithField("command", cmd.Command).Debug("Sending NG command")

	result, err := h.ngClient.Send(cmd.Data)
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
