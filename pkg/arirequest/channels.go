package arirequest

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// ChannelAnswer sends the channel answer request
func (r *requestHandler) ChannelAnswer(asteriskID, channelID string) error {
	url := fmt.Sprintf("/ari/channels/%s/answer", channelID)

	res, err := r.requester.SendARIRequest(asteriskID, url, reqPost, requestTimeout, "", "")
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// ChannelContinue sends the continue request
func (r *requestHandler) ChannelContinue(asteriskID, channelID, context, ext string, pri int, label string) error {
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

	res, err := r.requester.SendARIRequest(asteriskID, url, reqPost, requestTimeout, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// ChannelContinue sends the continue request
func (r *requestHandler) ChannelHangup(asteriskID, channelID string, code int) error {
	url := fmt.Sprintf("/ari/channels/%s", channelID)

	type Data struct {
		ReasonCode string `json:"reason_code"`
	}

	m, err := json.Marshal(Data{
		strconv.Itoa(code),
	})
	if err != nil {
		return err
	}

	res, err := r.requester.SendARIRequest(asteriskID, url, reqDelete, requestTimeout, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// ChannelVariableSet sends the variable set request
func (r *requestHandler) ChannelVariableSet(asteriskID, channelID, variable, value string) error {
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

	res, err := r.requester.SendARIRequest(asteriskID, url, reqPost, requestTimeout, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
