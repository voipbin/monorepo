package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	cmari "monorepo/bin-call-manager/models/ari"
	cmchannel "monorepo/bin-call-manager/models/channel"
	"monorepo/bin-common-handler/models/sock"
)

// AstChannelAnswer sends the channel answer request
func (r *requestHandler) AstChannelAnswer(ctx context.Context, asteriskID, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/answer", channelID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/answer", requestTimeoutDefault, 0, "", nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelContinue sends the continue request
func (r *requestHandler) AstChannelContinue(ctx context.Context, asteriskID, channelID, context, ext string, pri int, label string) error {
	url := fmt.Sprintf("/ari/channels/%s/continue", channelID)

	type Data struct {
		Context   string `json:"context"`
		Extension string `json:"extension"`
		Priority  int    `json:"priority"`
		Label     string `json:"label"`
	}

	m, err := json.Marshal(Data{
		context,
		ext,
		pri,
		label,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/continue", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelContinue sends the continue request
// delay: milliseconds
func (r *requestHandler) AstChannelHangup(ctx context.Context, asteriskID, channelID string, code cmari.ChannelCause, delay int) error {
	url := fmt.Sprintf("/ari/channels/%s", channelID)

	type Data struct {
		ReasonCode string `json:"reason_code"`
	}

	m, err := json.Marshal(Data{
		strconv.Itoa(int(code)),
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/channels/hangup", requestTimeoutDefault, delay, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res == nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelVariableGet sends the variable get request
func (r *requestHandler) AstChannelVariableGet(ctx context.Context, asteriskID, channelID, variable string) (string, error) {
	url := fmt.Sprintf("/ari/channels/%s/variable?variable=%s", channelID, variable)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodGet, "ast/channels/var", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return "", err
	case tmp.StatusCode > 299:
		return "", fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res struct {
		Value string `json:"value"`
	}

	if errUnmarshal := json.Unmarshal([]byte(tmp.Data), &res); errUnmarshal != nil {
		return "", err
	}

	return res.Value, nil
}

// AstChannelVariableSet sends the variable set request
func (r *requestHandler) AstChannelVariableSet(ctx context.Context, asteriskID, channelID, variable, value string) error {
	url := fmt.Sprintf("/ari/channels/%s/variable", channelID)

	type Data struct {
		Variable string `json:"variable"`
		Value    string `json:"value"`
	}

	m, err := json.Marshal(Data{
		variable,
		value,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/var", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelCreate sends the request for create a channel
func (r *requestHandler) AstChannelCreate(ctx context.Context, asteriskID, channelID, appArgs, endpoint, otherChannelID, originator, formats string, variables map[string]string) (*cmchannel.Channel, error) {
	uri := "/ari/channels/create"

	type Data struct {
		Endpoint       string            `json:"endpoint,omitempty"`
		App            string            `json:"app"`
		AppArgs        string            `json:"appArgs,omitempty"`
		ChannelID      string            `json:"channelId,omitempty"`
		OtherChannelID string            `json:"otherChannelId,omitempty"`
		Originator     string            `json:"originator,omitempty"`
		Formats        string            `json:"formats,omitempty"`
		Variables      map[string]string `json:"variables,omitempty"`
	}

	m, err := json.Marshal(Data{
		endpoint,
		defaultAstStasisApp,
		appArgs,
		channelID,
		otherChannelID,
		originator,
		formats,
		variables,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAst(ctx, asteriskID, uri, sock.RequestMethodPost, "ast/channels", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	tmpChannel, err := cmari.ParseChannel([]byte(tmp.Data))
	if err != nil {
		return nil, err
	}

	res := cmchannel.NewChannelByARIChannel(tmpChannel)
	return res, nil
}

// AstChannelCreateSnoop sends the request for create a snoop channel
func (r *requestHandler) AstChannelCreateSnoop(ctx context.Context, asteriskID, channelID, snoopID, appArgs string, spy, whisper cmchannel.SnoopDirection) (*cmchannel.Channel, error) {
	url := fmt.Sprintf("/ari/channels/%s/snoop", channelID)

	type Data struct {
		Spy     cmchannel.SnoopDirection `json:"spy,omitempty"`
		Whisper cmchannel.SnoopDirection `json:"whisper,omitempty"`
		App     string                   `json:"app"`
		AppArgs string                   `json:"appArgs,omitempty"`
		SnoopID string                   `json:"snoopId"`
	}

	m, err := json.Marshal(Data{
		spy,
		whisper,
		defaultAstStasisApp,
		appArgs,
		snoopID,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/snoop", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	tmpChannel, err := cmari.ParseChannel([]byte(tmp.Data))
	if err != nil {
		return nil, err
	}

	res := cmchannel.NewChannelByARIChannel(tmpChannel)
	return res, nil
}

// AstChannelGet gets the Asterisk's channel defail
func (r *requestHandler) AstChannelGet(ctx context.Context, asteriskID, channelID string) (*cmchannel.Channel, error) {
	url := fmt.Sprintf("/ari/channels/%s", channelID)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodGet, "ast/channels", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	tmpChannel, err := cmari.ParseChannel([]byte(tmp.Data))
	if err != nil {
		return nil, err
	}

	res := cmchannel.NewChannelByARIChannel(tmpChannel)
	return res, nil
}

// AstChannelDTMF sends the dtmf request
func (r *requestHandler) AstChannelDTMF(ctx context.Context, asteriskID, channelID string, digit string, duration, before, between, after int) error {
	url := fmt.Sprintf("/ari/channels/%s/dtmf", channelID)

	type Data struct {
		DTMF     string `json:"dtmf"`
		Duration int    `json:"duration"`
		Before   int    `json:"before"`
		Between  int    `json:"between"`
		After    int    `json:"after"`
	}

	m, err := json.Marshal(Data{
		digit,
		duration,
		before,
		between,
		after,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/hangup", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelDial dials the channel to the endpoint
func (r *requestHandler) AstChannelDial(ctx context.Context, asteriskID, channelID, caller string, timeout int) error {
	url := fmt.Sprintf("/ari/channels/%s/dial", channelID)

	type Data struct {
		Caller  string `json:"caller,omitempty"`
		Timeout int    `json:"timeout,omitempty"`
	}

	m, err := json.Marshal(Data{
		caller,
		timeout,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/dial", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelPlay plays the music file on the channel
// The channelID will be used for playbackId as well.
// medias string must be stared with sound:, recording:, number:, digits:, characters:, tone:.
func (r *requestHandler) AstChannelPlay(
	ctx context.Context,
	asteriskID string,
	channelID string,
	medias []string,
	language string,
	offsetms int,
	skipms int,
	playbackID string,
) (*cmari.Playback, error) {
	url := fmt.Sprintf("/ari/channels/%s/play", channelID)

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

	resp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/play", 10000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case resp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", resp.StatusCode)
	}

	res, err := cmari.ParsePlayback([]byte(resp.Data))
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AstChannelRecord records the given the channel
func (r *requestHandler) AstChannelRecord(ctx context.Context, asteriskID string, channelID string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error {
	url := fmt.Sprintf("/ari/channels/%s/record", channelID)

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

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/record", 10000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelExternalMedia creates a external media.
func (r *requestHandler) AstChannelExternalMedia(ctx context.Context, asteriskID string, channelID string, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string, data string, variables map[string]string) (*cmchannel.Channel, error) {

	url := "/ari/channels/externalMedia"

	type Data struct {
		ChannelID      string            `json:"channel_id"`
		App            string            `json:"app"`
		ExternalHost   string            `json:"external_host"`
		Encapsulation  string            `json:"encapsulation,omitempty"`
		Transport      string            `json:"transport,omitempty"`
		ConnectionType string            `json:"connection_type,omitempty"`
		Format         string            `json:"format"`
		Direction      string            `json:"direction,omitempty"`
		Data           string            `json:"data,omitempty"`
		Variables      map[string]string `json:"variables,omitempty"`
	}

	m, err := json.Marshal(Data{
		ChannelID:      channelID,
		App:            defaultAstStasisApp,
		ExternalHost:   externalHost,
		Encapsulation:  encapsulation,
		Transport:      transport,
		ConnectionType: connectionType,
		Format:         format,
		Direction:      direction,
		Data:           data,
		Variables:      variables,
	})
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/externalmedia", 10000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	tmpCh, err := cmari.ParseChannel([]byte(res.Data))
	if err != nil {
		return nil, err
	}

	ch := cmchannel.NewChannelByARIChannel(tmpCh)
	return ch, nil
}

// AstChannelRing ring the given channel.
func (r *requestHandler) AstChannelRing(ctx context.Context, asteriskID string, channelID string) error {

	url := fmt.Sprintf("/ari/channels/%s/ring", channelID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/externalmedia", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelHold holds the given the channel
func (r *requestHandler) AstChannelHoldOn(ctx context.Context, asteriskID string, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/hold", channelID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/hold", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelUnhold unholds the given the channel
func (r *requestHandler) AstChannelHoldOff(ctx context.Context, asteriskID string, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/hold", channelID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/channels/hold", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelMusicOnHoldOn puts the given the channel to the music on hold
func (r *requestHandler) AstChannelMusicOnHoldOn(ctx context.Context, asteriskID string, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/moh", channelID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/moh", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelMusicOnHoldOff puts out the given the channel from the music on hold
func (r *requestHandler) AstChannelMusicOnHoldOff(ctx context.Context, asteriskID string, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/moh", channelID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/channels/moh", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelSilenceOn puts the given the channel on the silence
func (r *requestHandler) AstChannelSilenceOn(ctx context.Context, asteriskID string, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/silence", channelID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/silence", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelSilenceOff puts out the given the channel from silence
func (r *requestHandler) AstChannelSilenceOff(ctx context.Context, asteriskID string, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/silence", channelID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/channels/silence", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelMuteOn puts the given the channel on the mute
// direction: Direction in which to mute audio(both, in, out)
func (r *requestHandler) AstChannelMuteOn(ctx context.Context, asteriskID string, channelID string, direction string) error {
	url := fmt.Sprintf("/ari/channels/%s/mute", channelID)

	type Data struct {
		Direction string `json:"direction"`
	}

	m, err := json.Marshal(Data{
		Direction: direction,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/channels/mute", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelMuteOff puts out the given the channel from mute
// direction: Direction in which to unmute audio
func (r *requestHandler) AstChannelMuteOff(ctx context.Context, asteriskID string, channelID string, direction string) error {
	url := fmt.Sprintf("/ari/channels/%s/mute", channelID)

	type Data struct {
		Direction string `json:"direction"`
	}

	m, err := json.Marshal(Data{
		Direction: direction,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/channels/mute", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
