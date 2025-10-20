package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	amagent "monorepo/bin-agent-manager/models/agent"
	amrequest "monorepo/bin-agent-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
)

// AgentV1AgentCreate sends a request to agent-manager
// to creating an Agent.
// it returns created call if it succeed.
// timeout: milliseconds
func (r *requestHandler) AgentV1AgentCreate(
	ctx context.Context,
	timeout int,
	customerID uuid.UUID,
	username string,
	password string,
	name string,
	detail string,
	ringMethod amagent.RingMethod,
	permission amagent.Permission,
	tagIDs []uuid.UUID,
	addresses []commonaddress.Address,
) (*amagent.Agent, error) {
	uri := "/v1/agents"

	data := &amrequest.V1DataAgentsPost{
		CustomerID: customerID,
		Username:   username,
		Password:   password,

		Name:       name,
		Detail:     detail,
		RingMethod: string(ringMethod),
		Permission: uint64(permission),
		TagIDs:     tagIDs,
		Addresses:  addresses,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPost, "agent/agents", timeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentGet sends a request to agent-manager
// to getting an agent.
// it returns an agent if it succeed.
func (r *requestHandler) AgentV1AgentGet(ctx context.Context, agentID uuid.UUID) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s", agentID)

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodGet, "agent/agents/<agent-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentGet sends a request to agent-manager
// to getting an agent.
// it returns an agent if it succeed.
func (r *requestHandler) AgentV1AgentGetByCustomerIDAndAddress(ctx context.Context, timeout int, customerID uuid.UUID, addr commonaddress.Address) (*amagent.Agent, error) {
	uri := "/v1/agents/get_by_customer_id_address"

	data := &amrequest.V1DataAgentsGetByCustomerIDAddressPost{
		CustomerID: customerID,
		Address:    addr,
	}
	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPost, "agent/agents/get_by_customer_id_address", timeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentGets sends a request to agent-manager
// to getting a list of agent info.
// it returns detail list of agent info if it succeed.
func (r *requestHandler) AgentV1AgentGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodGet, "agent/agents", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AgentV1AgentGetsByTagIDs sends a request to agent-manager
// to getting a list of agent info.
// it returns detail list of agent info if it succeed.
func (r *requestHandler) AgentV1AgentGetsByTagIDs(ctx context.Context, customerID uuid.UUID, tagIDs []uuid.UUID) ([]amagent.Agent, error) {

	if len(tagIDs) == 0 {
		return nil, fmt.Errorf("no tag id given")
	}

	tagStr := tagIDs[0].String()
	for _, tag := range tagIDs[1:] {
		tagStr = fmt.Sprintf("%s,%s", tagStr, tag.String())
	}

	uri := fmt.Sprintf("/v1/agents?customer_id=%s&tag_ids=%s", customerID, tagStr)

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodGet, "agent/agents", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AgentV1AgentGetsByTagIDs sends a request to agent-manager
// to getting a list of agent info.
// it returns detail list of agent info if it succeed.
func (r *requestHandler) AgentV1AgentGetsByTagIDsAndStatus(ctx context.Context, customerID uuid.UUID, tagIDs []uuid.UUID, status amagent.Status) ([]amagent.Agent, error) {

	if len(tagIDs) == 0 {
		return nil, fmt.Errorf("no tag id given")
	}

	tagStr := tagIDs[0].String()
	for _, tag := range tagIDs[1:] {
		tagStr = fmt.Sprintf("%s,%s", tagStr, tag.String())
	}

	uri := fmt.Sprintf("/v1/agents?customer_id=%s&tag_ids=%s&status=%s", customerID, tagStr, status)

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodGet, "agent/agents", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AgentV1AgentDelete sends a request to agent-manager
// to delete the agent.
// it returns error if something went wrong.
func (r *requestHandler) AgentV1AgentDelete(ctx context.Context, id uuid.UUID) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s", id)

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodDelete, "agent/agents/<agent-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentLogin sends a request to agent-manager
// to login the agent
// it returns error if something went wrong.
func (r *requestHandler) AgentV1AgentUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/addresses", id)

	data := &amrequest.V1DataAgentsIDAddressesPut{
		Addresses: addresses,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPut, "agent/agents/<agent-id>/addresses", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentUpdatePassword sends a request to agent-manager
// to update the agent's password
// it returns error if something went wrong.
// timeout: milliseconds
func (r *requestHandler) AgentV1AgentUpdatePassword(ctx context.Context, timeout int, id uuid.UUID, password string) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/password", id)

	data := &amrequest.V1DataAgentsIDPasswordPut{
		Password: password,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPut, "agent/agents/<agent-id>/password", timeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentUpdate sends a request to agent-manager
// to update teh agent basic info
// it returns error if something went wrong.
func (r *requestHandler) AgentV1AgentUpdate(ctx context.Context, id uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s", id)

	data := &amrequest.V1DataAgentsIDPut{
		Name:       name,
		Detail:     detail,
		RingMethod: string(ringMethod),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPut, "agent/agents/<agent-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentUpdate sends a request to agent-manager
// to update teh agent's tag_ids info
// it returns error if something went wrong.
func (r *requestHandler) AgentV1AgentUpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/tag_ids", id)

	data := &amrequest.V1DataAgentsIDTagIDsPut{
		TagIDs: tagIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPut, "agent/agents/<agent-id>/tag_ids", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentUpdateStatus sends a request to agent-manager
// to update teh agent's status info
// it returns error if something went wrong.
func (r *requestHandler) AgentV1AgentUpdateStatus(ctx context.Context, id uuid.UUID, status amagent.Status) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/status", id)

	data := &amrequest.V1DataAgentsIDStatusPut{
		Status: string(status),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPut, "agent/agents/<agent-id>/status", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AgentV1AgentUpdatePermission sends a request to agent-manager
// to update the agent permission
// it returns error if something went wrong.
func (r *requestHandler) AgentV1AgentUpdatePermission(ctx context.Context, id uuid.UUID, permission amagent.Permission) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/permission", id)

	data := &amrequest.V1DataAgentsIDPermissionPut{
		Permission: uint64(permission),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPut, "agent/agents/<agent-id>/permission", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
