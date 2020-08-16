package ari

import "encoding/json"

// Bridge ARI event types
type Bridge struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	BridgeType  string `json:"bridge_type"`
	Technology  string `json:"technology"`
	BridgeClass string `json:"bridge_class"`
	Creator     string `json:"creator"`

	VideoMode     string `json:"video_mode"`
	VideoSourceID string `json:"video_source_id"`

	Channels []string `json:"channels"`

	CreationTime string `json:"creationtime"`
}

// BridgeCreated ARI event struct
type BridgeCreated struct {
	Event
	Bridge Bridge `json:"bridge"`
}

// BridgeDestroyed ARI event struct
type BridgeDestroyed struct {
	Event
	Bridge Bridge `json:"bridge"`
}

// ParseBridge parses message into Brdige struct
func ParseBridge(message []byte) (*Bridge, error) {
	res := &Bridge{}

	err := json.Unmarshal(message, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
