package engine_dialogflow_handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/engine_dialogflow"

	dialogflow "cloud.google.com/go/dialogflow/apiv2"
	dialogflowpb "cloud.google.com/go/dialogflow/apiv2/dialogflowpb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"monorepo/bin-ai-manager/models/message"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *engineDialogflowHandler) MessageSend(ctx context.Context, cc *aicall.AIcall, m *message.Message) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "MessageSend",
		"aicall":  cc,
		"message": m,
	})

	var data engine_dialogflow.EngineDialogflow
	tmpData, err := json.Marshal(cc.AIEngineData)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the ai engine data")
	}

	if errUnmarshal := json.Unmarshal(tmpData, &data); errUnmarshal != nil {
		return nil, errors.Wrapf(errUnmarshal, "could not unmarshal the ai engine data")
	}

	decodedCredential, err := DecodeBase64(data.CredentialBase64)
	if err != nil {
		return nil, errors.Wrapf(err, "could not decode the credential base64")
	}

	creds, err := google.CredentialsFromJSON(ctx, decodedCredential, dialogflow.DefaultAuthScopes()...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create credentials from json")
	}

	endpoint := GetEndpointAddress(data.Region)
	log.WithFields(logrus.Fields{
		"endpoint": endpoint,
	}).Debugf("Checking session data.")

	// Create a Dialogflow client with the credentials in-memory
	client, err := dialogflow.NewSessionsClient(ctx,
		option.WithTokenSource(creds.TokenSource),
		option.WithEndpoint(endpoint),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the dialogflow client")
	}

	req := h.getRequest(&data, cc, m)
	log.Debugf("Checking value. request: %v", req)

	resp, err := h.send(ctx, client, req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the request")
	}
	log.Debugf("Received response. response: %v", resp)

	content := ""
	if resp.GetQueryResult() == nil {
		content = ""
	} else if resp.GetQueryResult().GetFulfillmentText() != "" { // Check if FulfillmentText is present
		content = resp.GetQueryResult().GetFulfillmentText()
	} else if len(resp.GetQueryResult().GetFulfillmentMessages()) > 0 { // Check for Fulfillment Messages
		for _, message := range resp.GetQueryResult().GetFulfillmentMessages() {
			if text := message.GetText(); text != nil && len(text.GetText()) > 0 {
				content = text.GetText()[0]
				break
			}
		}
	}

	res := &message.Message{
		Role:    message.RoleAssistant,
		Content: content,
	}
	return res, nil
}

func (h *engineDialogflowHandler) getRequest(engineData *engine_dialogflow.EngineDialogflow, cc *aicall.AIcall, message *message.Message) *dialogflowpb.DetectIntentRequest {
	sessionPath := fmt.Sprintf("projects/%s/agent/sessions/%s", engineData.ProjectID, cc.ID)
	if cc.AIEngineModel == ai.EngineModelDialogflowCX {
		sessionPath = fmt.Sprintf("projects/%s/locations/%s/agents/%s/sessions/%s", engineData.ProjectID, engineData.Region, engineData.AgentID, cc.ID)
	}

	lang := GetLanguage(cc.Language)
	res := &dialogflowpb.DetectIntentRequest{
		Session: sessionPath,
		QueryInput: &dialogflowpb.QueryInput{
			Input: &dialogflowpb.QueryInput_Text{
				Text: &dialogflowpb.TextInput{
					Text:         message.Content,
					LanguageCode: lang,
				},
			},
		},
	}

	return res
}

// DecodeBase64 decodes a base64-encoded string
func DecodeBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

func GetEndpointAddress(region engine_dialogflow.Region) string {
	if region == engine_dialogflow.RegionNone || region == engine_dialogflow.RegionGlobal {
		return "dialogflow.googleapis.com:443"
	}

	res := fmt.Sprintf("%s-dialogflow.googleapis.com:443", region)
	return res
}

func GetLanguage(locale string) string {
	if len(locale) < 2 {
		return ""
	}
	return locale[:2]
}
