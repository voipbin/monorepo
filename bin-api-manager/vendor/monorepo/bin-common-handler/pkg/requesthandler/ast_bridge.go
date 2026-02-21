package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cmari "monorepo/bin-call-manager/models/ari"
	cmbridge "monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-common-handler/models/sock"
)

// AstBridgeCreate sends the bridge create request
func (r *requestHandler) AstBridgeCreate(ctx context.Context, asteriskID, bridgeID, bridgeName string, bridgeTypes []cmbridge.Type) error {
	url := "/ari/bridges"

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

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/bridges", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstBridgeDelete sends the bridge delete request
func (r *requestHandler) AstBridgeDelete(ctx context.Context, asteriskID, bridgeID string) error {
	url := fmt.Sprintf("/ari/bridges/%s", bridgeID)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/bridges", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstBridgeGet sends the bridge get request
func (r *requestHandler) AstBridgeGet(ctx context.Context, asteriskID, bridgeID string) (*cmbridge.Bridge, error) {
	url := fmt.Sprintf("/ari/bridges/%s", bridgeID)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodGet, "ast/bridges", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var tmpBridge cmari.Bridge
	if errParse := parseResponse(tmp, &tmpBridge); errParse != nil {
		return nil, errParse
	}

	bridge := cmbridge.NewBridgeByARIBridge(&tmpBridge)
	return bridge, nil
}

// AstBridgeAddChannel sends the request for adding the channel to the bridge
func (r *requestHandler) AstBridgeAddChannel(ctx context.Context, asteriskID, bridgeID, channelID, role string, absorbDTMF, mute bool) error {
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

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/bridges/addchannel", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstBridgeRemoveChannel sends the request for removing the channel from the bridge
func (r *requestHandler) AstBridgeRemoveChannel(ctx context.Context, asteriskID, bridgeID, channelID string) error {
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

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/bridges/removechannel", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstBridgeRecord sends the request for recording the given bridge
func (r *requestHandler) AstBridgeRecord(ctx context.Context, asteriskID string, bridgeID string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error {
	url := fmt.Sprintf("/ari/bridges/%s/record", bridgeID)

	type Data struct {
		Name               string `json:"name"`
		Format             string `json:"format"`
		MaxDurationSeconds int    `json:"maxDurationSeconds"`
		MaxSilenceSeconds  int    `json:"maxSilenceSeconds"`
		Beep               bool   `json:"beep"`
		TerminateOn        string `json:"terminateOn"`
		IfExists           string `json:"ifExists"`
	}

	m, err := json.Marshal(Data{
		Name:               filename,
		Format:             format,
		MaxDurationSeconds: duration,
		MaxSilenceSeconds:  silence,
		Beep:               beep,
		TerminateOn:        endKey,
		IfExists:           ifExists,
	})
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/bridges/record", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstBridgePlay sends the request for playing media to the given bridge
func (r *requestHandler) AstBridgePlay(
	ctx context.Context,
	asteriskID string,
	bridgeID string,
	medias []string,
	language string,
	offsetms int,
	skipms int,
	playbackID string,
) (*cmari.Playback, error) {
	url := fmt.Sprintf("/ari/bridges/%s/play", bridgeID)

	payload := struct {
		Media      []string `json:"media"`
		PlaybackID string   `json:"playbackId,omitempty"`
		Lang       string   `json:"lang,omitempty"`
		Offsetms   int      `json:"offsetms,omitempty"`
		Skipms     int      `json:"skipms,omitempty"`
	}{
		Media:      medias,
		PlaybackID: playbackID,
		Lang:       language,
		Offsetms:   offsetms,
		Skipms:     skipms,
	}

	m, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/bridges/play", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmari.Playback
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
