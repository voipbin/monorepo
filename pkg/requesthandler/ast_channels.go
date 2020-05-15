package requesthandler

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// AstChannelAnswer sends the channel answer request
func (r *requestHandler) AstChannelAnswer(asteriskID, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/answer", channelID)

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, resourceAstChannelsAnswer, requestTimeoutDefault, "", "")
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, resourceAstChannelsContinue, requestTimeoutDefault, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodDelete, resourceAstChannelsHangup, requestTimeoutDefault, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, resourceAstChannelsVar, requestTimeoutDefault, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
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

	res, err := r.sendRequestAst(asteriskID, url, rabbitmq.RequestMethodPost, resourceAstChannelsSnoop, requestTimeoutDefault, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
