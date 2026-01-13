package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/listenhandler/models/request"
)

// convertAgentFilters converts URL query filters to typed agent filters
func convertAgentFilters(rawFilters map[string]string) map[agent.Field]any {
	filters := make(map[agent.Field]any)
	for k, v := range rawFilters {
		switch k {
		case "customer_id":
			filters[agent.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "deleted":
			filters[agent.FieldDeleted] = v == "true"
		case "status":
			filters[agent.FieldStatus] = agent.Status(v)
		default:
			filters[agent.Field(k)] = v
		}
	}
	return filters
}

// processV1AgentsGet handles GET /v1/agents request
func (h *listenHandler) processV1AgentsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// parse the filters and convert to typed filters
	rawFilters := h.utilHandler.URLParseFilters(u)
	filters := convertAgentFilters(rawFilters)

	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AgentsGet",
		"size":    pageSize,
		"token":   pageToken,
		"filters": filters,
	})

	tmp, err := h.agentHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDGet handles Get /v1/agents/<agent-id> request
func (h *listenHandler) processV1AgentsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsIDGet",
		"agent_id": id,
	})
	log.Debug("Executing processV1AgentsIDGet.")

	tmp, err := h.agentHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get an agent info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsPost handles Post /v1/agents request
func (h *listenHandler) processV1AgentsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1AgentsPost",
	})
	log.Debug("Executing processV1AgentsPost.")

	var reqData request.V1DataAgentsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"customer_id": reqData.CustomerID,
		"username":    reqData.Username,
		"permission":  reqData.Permission,
	})
	log.Debug("Creating an agent.")

	// create an agent
	tmp, err := h.agentHandler.Create(
		ctx,
		reqData.CustomerID,
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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDDelete handles Delete /v1/agents/<agent_id> request
func (h *listenHandler) processV1AgentsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsIDDelete",
		"agent_id": id,
	})
	log.Debug("Executing processV1AgentsIDDelete.")

	tmp, err := h.agentHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the agent info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsGetByCustomerIDAddressPost handles Post /v1/agents/get_by_customer_id_address request
func (h *listenHandler) processV1AgentsGetByCustomerIDAddressPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1AgentsGetByCustomerIDAddressPost",
	})
	log.Debug("Executing processV1AgentsGetByCustomerIDAddressPost.")

	var reqData request.V1DataAgentsGetByCustomerIDAddressPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"customer_id": reqData.CustomerID,
		"address":     reqData.Address,
	})
	log.Debug("Getting an agent.")

	// create an agent
	tmp, err := h.agentHandler.GetByCustomerIDAndAddress(
		ctx,
		reqData.CustomerID,
		&reqData.Address,
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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsUsernameLogin handles Post /v1/agents/<agent_username>/login request
func (h *listenHandler) processV1AgentsUsernameLogin(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	username := uriItems[3]
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsUsernameLogin",
		"username": username,
	})
	log.Debug("Executing processV1AgentsUsernameLogin.")

	var reqData request.V1DataAgentsUsernameLoginPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.Login(ctx, username, reqData.Password)
	if err != nil {
		log.Errorf("Could not login the agent info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDAddressesPut handles Put /v1/agents/<agent_id>/addresses request
func (h *listenHandler) processV1AgentsIDAddressesPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsIDAddressesPut",
		"agent_id": id,
	})
	log.Debug("Executing processV1AgentsIDAddressesPut.")

	var reqData request.V1DataAgentsIDAddressesPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.UpdateAddresses(ctx, id, reqData.Addresses)
	if err != nil {
		log.Errorf("Could not update the agent's addresses info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDPut handles Put /v1/agents/<agent_id> request
func (h *listenHandler) processV1AgentsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsIDPut",
		"agent_id": id,
	})
	log.Debug("Executing processV1AgentsIDPut.")

	var reqData request.V1DataAgentsIDPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.UpdateBasicInfo(ctx, id, reqData.Name, reqData.Detail, agent.RingMethod(reqData.RingMethod))
	if err != nil {
		log.Errorf("Could not update the agent's basic info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDStatusPut handles Put /v1/agents/<agent_id>/status request
func (h *listenHandler) processV1AgentsIDStatusPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsIDStatusPut",
		"agent_id": id,
	})
	log.Debug("Executing processV1AgentsIDStatusPut.")

	var reqData request.V1DataAgentsIDStatusPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.UpdateStatus(ctx, id, agent.Status(reqData.Status))
	if err != nil {
		log.Errorf("Could not update the agent's status info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDPasswordPut handles Put /v1/agents/<agent_id>/password request
func (h *listenHandler) processV1AgentsIDPasswordPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsIDPasswordPut",
		"agent_id": id,
	})
	log.Debug("Executing processV1AgentsIDPasswordPut.")

	var req request.V1DataAgentsIDPasswordPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.UpdatePassword(ctx, id, req.Password)
	if err != nil {
		log.Errorf("Could not update the agent's status info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDTagIDsPut handles Put /v1/agents/<agent_id>/tag_ids request
func (h *listenHandler) processV1AgentsIDTagIDsPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsIDTagIDsPut",
		"agent_id": id,
	})
	log.Debug("Executing processV1AgentsIDTagIDsPut.")

	var reqData request.V1DataAgentsIDTagIDsPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.UpdateTagIDs(ctx, id, reqData.TagIDs)
	if err != nil {
		log.Errorf("Could not update the agent's tag_ids info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AgentsIDPermissionPut handles Put /v1/agents/<agent_id>/permission request
func (h *listenHandler) processV1AgentsIDPermissionPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1AgentsIDPermissionPut",
		"agent_id": id,
	})
	log.Debug("Executing processV1AgentsIDPermissionPut.")

	var reqData request.V1DataAgentsIDPermissionPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.UpdatePermission(ctx, id, agent.Permission(reqData.Permission))
	if err != nil {
		log.Errorf("Could not update the agent's permission info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
