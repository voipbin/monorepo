package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
	"net/http"
)

// parseAMIResponseData parses response data.
func (r *requestHandler) parseAMIResponseData(data []byte) error {

	var resData map[string]string
	if err := json.Unmarshal(data, &resData); err != nil {
		return err
	}

	if resData["Response"] != "Success" {
		return fmt.Errorf("could not execute the ami action. response: %s", resData["Response"])
	}

	return nil
}

// Send the Redirect AMI action request.
//
// Action: Redirect
// ActionID: <value>
// Channel: <value>
// Context: <value>
// Exten: <value>
// Priority: <value>
// ExtraChannel: <value>
// ExtraExten: <value>
// ExtraContext: <value>
// ExtraPriority: <value>
func (r *requestHandler) AstAMIRedirect(ctx context.Context, asteriskID, channelID, context, exten, priority string) error {
	url := "/ami/action"

	data := map[string]string{
		"Action":   "Redirect",
		"Channel":  channelID,
		"Context":  context,
		"Exten":    exten,
		"Priority": priority,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/ami", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	// the proxy answers an unroutable or failed request with a status code and no body.
	// report the status code instead of failing to parse the empty body.
	if res == nil || res.StatusCode != http.StatusOK {
		code := 0
		if res != nil {
			code = res.StatusCode
		}
		return fmt.Errorf("could not send the ami request. status code: %d", code)
	}

	if err := r.parseAMIResponseData(res.Data); err != nil {
		return err
	}

	return nil
}
