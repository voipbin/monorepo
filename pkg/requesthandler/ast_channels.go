package requesthandler

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// AstChannelAnswer sends the channel answer request
func (r *requestHandler) AstChannelAnswer(asteriskID, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/answer", channelID)

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsAnswer, requestTimeoutDefault, 0, "", nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelContinue sends the continue request
func (r *requestHandler) AstChannelContinue(asteriskID, channelID, context, ext string, pri int, label string) error {
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsContinue, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelContinue sends the continue request
func (r *requestHandler) AstChannelHangup(asteriskID, channelID string, code ari.ChannelCause) error {
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodDelete, resourceAstChannelsHangup, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelVariableSet sends the variable set request
func (r *requestHandler) AstChannelVariableSet(asteriskID, channelID, variable, value string) error {
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsVar, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelCreate sends the request for create a channel
func (r *requestHandler) AstChannelCreate(asteriskID, channelID, appArgs, endpoint, otherChannelID, originator, formats string, variables map[string]string) error {
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
		return err
	}

	res, err := r.sendRequestAst(asteriskID, uri, rabbitmqhandler.RequestMethodPost, resourceAstChannels, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelCreateSnoop sends the request for create a snoop channel
func (r *requestHandler) AstChannelCreateSnoop(asteriskID, channelID, snoopID, appArgs string, spy, whisper channel.SnoopDirection) error {
	url := fmt.Sprintf("/ari/channels/%s/snoop", channelID)

	type Data struct {
		Spy     channel.SnoopDirection `json:"spy,omitempty"`
		Whisper channel.SnoopDirection `json:"whisper,omitempty"`
		App     string                 `json:"app"`
		AppArgs string                 `json:"appArgs,omitempty"`
		SnoopID string                 `json:"snoopId"`
	}

	m, err := json.Marshal(Data{
		spy,
		whisper,
		defaultAstStasisApp,
		appArgs,
		snoopID,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsSnoop, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelGet gets the Asterisk's channel defail
func (r *requestHandler) AstChannelGet(asteriskID, channelID string) (*channel.Channel, error) {
	url := fmt.Sprintf("/ari/channels/%s", channelID)

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodGet, resourceAstChannels, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	tmpChannel, err := ari.ParseChannel([]byte(res.Data))
	if err != nil {
		return nil, err
	}

	channel := channel.NewChannelByARIChannel(tmpChannel)
	return channel, nil
}

// AstChannelDTMF sends the dtmf request
func (r *requestHandler) AstChannelDTMF(asteriskID, channelID string, digit string, duration, before, between, after int) error {
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsHangup, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelDial dials the channel to the endpoint
func (r *requestHandler) AstChannelDial(asteriskID, channelID, caller string, timeout int) error {
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsDial, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
func (r *requestHandler) AstChannelPlay(asteriskID string, channelID string, actionID uuid.UUID, medias []string, lang string) error {
	url := fmt.Sprintf("/ari/channels/%s/play", channelID)

	type Data struct {
		Media      []string `json:"media"`
		PlaybackID string   `json:"playbackId"`
		Language   string   `json:"lang,omitempty"`
	}

	m, err := json.Marshal(Data{
		medias,
		actionID.String(),
		lang,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsPlay, 10, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelRecord records the given the channel
func (r *requestHandler) AstChannelRecord(asteriskID string, channelID string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error {
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsRecord, 10, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstChannelExternalMedia creates a external media.
func (r *requestHandler) AstChannelExternalMedia(asteriskID string, channelID string, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string, data string, variables map[string]string) (*channel.Channel, error) {

	url := fmt.Sprintf("/ari/channels/externalMedia")

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
		Variables:      variables,
	})
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestAst(asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstChannelsExternalMedia, 10, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	tmpCh, err := ari.ParseChannel([]byte(res.Data))
	if err != nil {
		return nil, err
	}

	ch := channel.NewChannelByARIChannel(tmpCh)
	return ch, nil
}
