package requesthandler

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/rabbitmq"
)

// AstBridgeCreate sends the bridge create request
func (r *requestHandler) AstBridgeCreate(asteriskID, bridgeID, bridgeName string, bridgeTypes []bridge.Type) error {
	url := fmt.Sprint("/ari/bridges")

	type Data struct {
		Type     string `json:"type"`
		BridgeID string `json:"bridgeId"`
		Name     string `json:"name"`
	}

	tmpList := []string{}
	for _, bridgeType := range bridgeTypes {
		tmpList = append(tmpList, string(bridgeType))
	}

	m, err := json.Marshal(Data{
		strings.Join(tmpList, ","),
		bridgeID,
		bridgeName,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, resourceAstBridges, requestTimeoutDefault, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstBridgeDelete sends the bridge delete request
func (r *requestHandler) AstBridgeDelete(asteriskID, bridgeID string) error {
	url := fmt.Sprintf("/ari/bridges/%s", bridgeID)

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodDelete, resourceAstBridges, requestTimeoutDefault, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstBridgeGet sends the bridge get request
func (r *requestHandler) AstBridgeGet(asteriskID, bridgeID string) (*bridge.Bridge, error) {
	url := fmt.Sprintf("/ari/bridges/%s", bridgeID)

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodGet, resourceAstBridges, requestTimeoutDefault, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	tmpBridge, err := ari.ParseBridge([]byte(res.Data))
	if err != nil {
		return nil, err
	}

	bridge := bridge.NewBridgeByARIBridge(tmpBridge)
	return bridge, nil
}

// AstBridgeAddChannel sends the request for adding the channel to the bridge
func (r *requestHandler) AstBridgeAddChannel(asteriskID, bridgeID, channelID, role string, absorbDTMF, mute bool) error {
	url := fmt.Sprintf("/ari/bridges/%s/addChannel", bridgeID)

	type Data struct {
		ChannelID  string `json:"channel"`
		Role       string `json:"role,omitempty"`
		AbsorbDTMF bool   `json:"absorbDTMF,omitempty"`
		Mute       bool   `json:"mute,omitempty"`
	}

	m, err := json.Marshal(Data{
		channelID,
		role,
		absorbDTMF,
		mute,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, resourceAstBridgesAddChannel, requestTimeoutDefault, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstBridgeRemoveChannel sends the request for removing the channel from the bridge
func (r *requestHandler) AstBridgeRemoveChannel(asteriskID, bridgeID, channelID string) error {
	url := fmt.Sprintf("/ari/bridges/%s/removeChannel", bridgeID)

	type Data struct {
		ChannelID string `json:"channel"`
	}

	m, err := json.Marshal(Data{
		channelID,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, resourceAstBridgesRemoveChannel, requestTimeoutDefault, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
