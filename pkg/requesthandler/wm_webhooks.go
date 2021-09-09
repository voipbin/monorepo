package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	request "gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/listenhandler/models/request"
)

func (r *requestHandler) WMWebhookPost(webhookMethod, webhookURI, dataType, messageType string, messageData []byte) error {

	uri := fmt.Sprintf("/v1/webhooks")

	m, err := json.Marshal(request.V1DataWebhooksPost{
		Method:     webhook.MethodType(webhookMethod),
		WebhookURI: webhookURI,
		DataType:   webhook.DataType(dataType),
		Data: request.WebhookData{
			Type: messageType,
			Data: messageData,
		},
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestWebhook(uri, rabbitmqhandler.RequestMethodPost, resourceWebhookWebhooks, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("could not find action")
	}

	return nil
}

// // SMRecordingGet sends a request to storage-manager
// // to getting a recording download link.
// // it returns download link if it succeed.
// func (r *requestHandler) WMWebhookPost(reqMethod, reqURI, reqDataType string, reqData []byte) error {
// 	uri := fmt.Sprintf("/v1/webhooks")

// 	method := wbwebhook.MethodTypePOST
// 	if strings.ToUpper(reqMethod) == "POST" {
// 		method = wbwebhook.MethodTypePOST
// 	} else if strings.ToUpper(reqMethod) == "GET" {
// 		method = wbwebhook.MethodTypeGET
// 	} else if strings.ToUpper(reqMethod) == "PUT" {
// 		method = wbwebhook.MethodTypePUT
// 	} else if strings.ToUpper(reqMethod) == "DELETE" {
// 		method = wbwebhook.MethodTypeDELETE
// 	} else {
// 		return fmt.Errorf("not support method")
// 	}

// 	dataType := wbwebhook.DataTypeEmpty
// 	if strings.ToLower(reqDataType) == "application/json" {
// 		dataType = wbwebhook.DataTypeJSON
// 	}

// 	tmpData := &wbrequest.V1DataWebhooksPost{
// 		Webhook: wbwebhook.Webhook{
// 			Method:     method,
// 			WebhookURI: reqURI,
// 			DataType:   dataType,
// 			Data:       reqData,
// 		},
// 	}

// 	requestData, err := json.Marshal(tmpData)
// 	if err != nil {
// 		return err
// 	}

// 	// sending a webhook would take a longer time.
// 	// so, we are putting enough timeout here.
// 	res, err := r.sendRequestWebhook(uri, rabbitmqhandler.RequestMethodPost, resourceWebhookWebhooks, 60, 0, ContentTypeJSON, requestData)
// 	switch {
// 	case err != nil:
// 		return err
// 	case res == nil:
// 		// not found
// 		return fmt.Errorf("response code: %d", 404)
// 	case res.StatusCode > 299:
// 		return fmt.Errorf("response code: %d", res.StatusCode)
// 	}

// 	return nil
// }
