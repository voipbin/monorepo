package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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
	url := "/ami"

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

	res, err := r.sendRequestAst(ctx, asteriskID, url, rabbitmqhandler.RequestMethodPost, resourceAstAMI, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if err := r.parseAMIResponseData(res.Data); err != nil {
		return err
	}

	return nil
}
