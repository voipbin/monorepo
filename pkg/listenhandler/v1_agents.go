package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/listenhandler/models/request"
)

// processV1AgentsGet handles GET /v1/agents request
func (h *listenHandler) processV1AgentsGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get user_id
	tmpUserID, _ := strconv.Atoi(u.Query().Get("user_id"))
	userID := uint64(tmpUserID)

	// parse tagIDs
	hasTagIDs := u.Query().Has("tag_ids")
	tmpTagIDs := u.Query().Get("tag_ids")
	tagIDs := []uuid.UUID{}
	if tmpTagIDs != "" {
		tagIDs = parseTagIDs(tmpTagIDs)
	}

	hasStatus := u.Query().Has("status")
	tmpStatus := u.Query().Get("status")

	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1AgentsGet",
		"user":  userID,
		"size":  pageSize,
		"token": pageToken,
	})

	var tmpRes []*agent.Agent
	var tmpErr error

	if hasTagIDs && hasStatus {
		tmpRes, tmpErr = h.agentHandler.AgentGetsByTagIDsAndStatus(ctx, userID, tagIDs, agent.Status(tmpStatus))
	} else if hasTagIDs {
		tmpRes, tmpErr = h.agentHandler.AgentGetsByTagIDs(ctx, userID, tagIDs)
	} else {
		tmpRes, tmpErr = h.agentHandler.AgentGets(ctx, userID, pageSize, pageToken)
	}
	if tmpErr != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmpRes)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmpRes, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDGet handles Get /v1/agents/<agent-id> request
func (h *listenHandler) processV1AgentsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1AgentsIDGet",
			"agent_id": id,
		})
	log.Debug("Executing processV1AgentsIDGet.")

	tmp, err := h.agentHandler.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get an agent info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsPost handles Post /v1/agents request
func (h *listenHandler) processV1AgentsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1AgentsPost",
		})
	log.Debug("Executing processV1AgentsPost.")

	var reqData request.V1DataAgentsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"user_id":    reqData.UserID,
		"username":   reqData.Username,
		"permission": reqData.Permission,
	})
	log.Debug("Creating an agent.")

	// create an agent
	tmp, err := h.agentHandler.AgentCreate(
		ctx,
		reqData.UserID,
		reqData.Username,
		reqData.Password,
		reqData.Name,
		reqData.Detail,
		agent.RingMethod(reqData.RingMethod),
		agent.Permission(reqData.Permission),
		reqData.TagIDs,
		reqData.Addresses,
	)
	if err != nil {
		log.Errorf("Could not create an agent info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDDelete handles Delete /v1/agents/<agent_id> request
func (h *listenHandler) processV1AgentsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1AgentsIDDelete",
			"agent_id": id,
		})
	log.Debug("Executing processV1AgentsIDDelete.")

	if err := h.agentHandler.AgentDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the agent info. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1AgentsUsernameLogin handles Post /v1/agents/<agent_username>/login request
func (h *listenHandler) processV1AgentsUsernameLogin(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	username := uriItems[3]
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1AgentsUsernameLogin",
			"username": username,
		})
	log.Debug("Executing processV1AgentsUsernameLogin.")

	var reqData request.V1DataAgentsUsernameLoginPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.AgentLogin(ctx, reqData.UserID, username, reqData.Password)
	if err != nil {
		log.Errorf("Could not login the agent info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDAddressesPut handles Post /v1/agents/<agent_id>/addresses request
func (h *listenHandler) processV1AgentsIDAddressesPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1AgentsIDAddressesPut",
			"agent_id": id,
		})
	log.Debug("Executing processV1AgentsIDAddressesPut.")

	var reqData request.V1DataAgentsIDAddressesPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	if err := h.agentHandler.AgentUpdateAddresses(ctx, id, reqData.Addresses); err != nil {
		log.Errorf("Could not update the agent's addresses info. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1AgentsIDPut handles Post /v1/agents/<agent_id> request
func (h *listenHandler) processV1AgentsIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1AgentsIDPut",
			"agent_id": id,
		})
	log.Debug("Executing processV1AgentsIDPut.")

	var reqData request.V1DataAgentsIDPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	if err := h.agentHandler.AgentUpdateBasicInfo(ctx, id, reqData.Name, reqData.Detail, agent.RingMethod(reqData.RingMethod)); err != nil {
		log.Errorf("Could not update the agent's basic info. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1AgentsIDTagIDsPut handles Post /v1/agents/<agent_id>/tag_ids request
func (h *listenHandler) processV1AgentsIDTagIDsPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1AgentsIDTagIDsPut",
			"agent_id": id,
		})
	log.Debug("Executing processV1AgentsIDTagIDsPut.")

	var reqData request.V1DataAgentsIDTagIDsPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	if err := h.agentHandler.AgentUpdateTagIDs(ctx, id, reqData.TagIDs); err != nil {
		log.Errorf("Could not update the agent's tag_ids info. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1AgentsIDDialPost handles Post /v1/agents/<agent_id>/dial request
func (h *listenHandler) processV1AgentsIDDialPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1AgentsIDDialPost",
			"agent_id": id,
		})
	log.Debug("Executing processV1AgentsIDDialPost.")

	var reqData request.V1DataAgentsIDDialPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// dial
	if err := h.agentHandler.AgentDial(ctx, id, &reqData.Source, reqData.ConfbridgeID); err != nil {
		log.Errorf("Could not dial to the agent. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
