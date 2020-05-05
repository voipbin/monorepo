package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// BridgeType type
type BridgeType string

// List of BirdgeTypes
const (
	BridgeTypeMixing     BridgeType = "mixing"
	BirdgeTypeHolding    BridgeType = "holding"
	BirdgeTypeDTMFEvents BridgeType = "dtmf_events"
	BridgeTypeProxyMedia BridgeType = "proxy_media"
	BridgeTypeVideoSFU   BridgeType = "video_sfu"
)

// AstBridgeCreate sends the bridge create request
func (r *requestHandler) AstBridgeCreate(asteriskID, bridgeID, bridgeName string, bridgeType BridgeType) error {
	url := fmt.Sprint("/ari/bridges")

	type Data struct {
		Type     string `json:"type"`
		BridgeID string `json:"bridgeId"`
		Name     string `json:"name"`
	}

	m, err := json.Marshal(Data{
		string(bridgeType),
		bridgeID,
		bridgeName,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, requestTimeoutDefault, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstBridgeDelete sends the bridge delete request
func (r *requestHandler) AstBridgeDelete(asteriskID, bridgeID string) error {
	url := fmt.Sprintf("/ari/bridges/%s", bridgeID)

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodDelete, requestTimeoutDefault, ContentTypeJSON, string(""))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, requestTimeoutDefault, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
