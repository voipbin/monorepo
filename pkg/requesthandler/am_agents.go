package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	amrequest "gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/listenhandler/models/request"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// AMV1AgentCreate sends a request to agent-manager
// to creating an Agent.
// it returns created call if it succeed.
// timeout: milliseconds
func (r *requestHandler) AMV1AgentCreate(
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
	addresses []cmaddress.Address,
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

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodPost, resourceAMAgent, timeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AMV1AgentGet sends a request to agent-manager
// to getting an agent.
// it returns an agent if it succeed.
func (r *requestHandler) AMV1AgentGet(ctx context.Context, agentID uuid.UUID) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s", agentID)

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodGet, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CMV1AgentGets sends a request to agent-manager
// to getting a list of agent info.
// it returns detail list of agent info if it succeed.
func (r *requestHandler) AMV1AgentGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodGet, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// AMV1AgentGetsByTagIDs sends a request to agent-manager
// to getting a list of agent info.
// it returns detail list of agent info if it succeed.
func (r *requestHandler) AMV1AgentGetsByTagIDs(ctx context.Context, customerID uuid.UUID, tagIDs []uuid.UUID) ([]amagent.Agent, error) {

	if len(tagIDs) == 0 {
		return nil, fmt.Errorf("no tag id given")
	}

	tagStr := tagIDs[0].String()
	for _, tag := range tagIDs[1:] {
		tagStr = fmt.Sprintf("%s,%s", tagStr, tag.String())
	}

	uri := fmt.Sprintf("/v1/agents?customer_id=%s&tag_ids=%s", customerID, tagStr)

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodGet, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// AMV1AgentGetsByTagIDs sends a request to agent-manager
// to getting a list of agent info.
// it returns detail list of agent info if it succeed.
func (r *requestHandler) AMV1AgentGetsByTagIDsAndStatus(ctx context.Context, customerID uuid.UUID, tagIDs []uuid.UUID, status amagent.Status) ([]amagent.Agent, error) {

	if len(tagIDs) == 0 {
		return nil, fmt.Errorf("no tag id given")
	}

	tagStr := tagIDs[0].String()
	for _, tag := range tagIDs[1:] {
		tagStr = fmt.Sprintf("%s,%s", tagStr, tag.String())
	}

	uri := fmt.Sprintf("/v1/agents?customer_id=%s&tag_ids=%s&status=%s", customerID, tagStr, status)

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodGet, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CMV1AgentDelete sends a request to agent-manager
// to delete the agent.
// it returns error if something went wrong.
func (r *requestHandler) AMV1AgentDelete(ctx context.Context, id uuid.UUID) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s", id)

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodDelete, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CMV1AgentLogin sends a request to agent-manager
// to login the agent
// it returns error if something went wrong.
// timeout: milliseconds
func (r *requestHandler) AMV1AgentLogin(ctx context.Context, timeout int, customerID uuid.UUID, username, password string) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/login", username)

	data := &amrequest.V1DataAgentsUsernameLoginPost{
		CustomerID: customerID,
		Password:   password,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodPost, resourceAMAgent, timeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CMV1AgentLogin sends a request to agent-manager
// to login the agent
// it returns error if something went wrong.
func (r *requestHandler) AMV1AgentUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []cmaddress.Address) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/addresses", id)

	data := &amrequest.V1DataAgentsIDAddressesPut{
		Addresses: addresses,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodPut, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AMV1AgentUpdatePassword sends a request to agent-manager
// to update the agent's password
// it returns error if something went wrong.
// timeout: milliseconds
func (r *requestHandler) AMV1AgentUpdatePassword(ctx context.Context, timeout int, id uuid.UUID, password string) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/password", id)

	data := &amrequest.V1DataAgentsIDPasswordPut{
		Password: password,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodPut, resourceAMAgent, timeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CMV1AgentUpdate sends a request to agent-manager
// to update teh agent basic info
// it returns error if something went wrong.
func (r *requestHandler) AMV1AgentUpdate(ctx context.Context, id uuid.UUID, name, detail, ringMethod string) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s", id)

	data := &amrequest.V1DataAgentsIDPut{
		Name:       name,
		Detail:     detail,
		RingMethod: ringMethod,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodPut, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AMV1AgentUpdate sends a request to agent-manager
// to update teh agent's tag_ids info
// it returns error if something went wrong.
func (r *requestHandler) AMV1AgentUpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/tag_ids", id)

	data := &amrequest.V1DataAgentsIDTagIDsPut{
		TagIDs: tagIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodPut, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AMV1AgentUpdateStatus sends a request to agent-manager
// to update teh agent's status info
// it returns error if something went wrong.
func (r *requestHandler) AMV1AgentUpdateStatus(ctx context.Context, id uuid.UUID, status amagent.Status) (*amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/agents/%s/status", id)

	data := &amrequest.V1DataAgentsIDStatusPut{
		Status: string(status),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodPut, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AMV1AgentDial sends a request to agent-manager
// to dial to the agent.
// it returns error if something went wrong.
func (r *requestHandler) AMV1AgentDial(ctx context.Context, id uuid.UUID, source *cmaddress.Address, confbridgeID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/agents/%s/dial", id)

	data := &amrequest.V1DataAgentsIDDialPost{
		Source:       *source,
		ConfbridgeID: confbridgeID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestAM(uri, rabbitmqhandler.RequestMethodPost, resourceAMAgent, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
