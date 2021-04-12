package requesthandler

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	wbwebhook "gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	wbrequest "gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/listenhandler/models/request"
)

// SMRecordingGet sends a request to storage-manager
// to getting a recording download link.
// it returns download link if it succeed.
func (r *requestHandler) WMWebhookPost(reqMethod, reqURI, reqDataType string, reqData []byte) error {
	uri := fmt.Sprintf("/v1/webhooks")

	method := wbwebhook.MethodTypePOST
	if strings.ToUpper(reqMethod) == "POST" {
		method = wbwebhook.MethodTypePOST
	} else if strings.ToUpper(reqMethod) == "GET" {
		method = wbwebhook.MethodTypeGET
	} else if strings.ToUpper(reqMethod) == "PUT" {
		method = wbwebhook.MethodTypePUT
	} else if strings.ToUpper(reqMethod) == "DELETE" {
		method = wbwebhook.MethodTypeDELETE
	} else {
		return fmt.Errorf("not support method")
	}

	dataType := wbwebhook.DataTypeEmpty
	if strings.ToLower(reqDataType) == "application/json" {
		dataType = wbwebhook.DataTypeJSON
	}

	tmpData := &wbrequest.V1DataWebhooksPost{
		Webhook: wbwebhook.Webhook{
			Method:     method,
			WebhookURI: reqURI,
			DataType:   dataType,
			Data:       reqData,
		},
	}

	requestData, err := json.Marshal(tmpData)
	if err != nil {
		return err
	}

	res, err := r.sendRequestStorage(uri, rabbitmqhandler.RequestMethodPost, resourceStorageRecording, 60, 0, ContentTypeJSON, requestData)
	switch {
	case err != nil:
		return err
	case res == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}
