package requestexternal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/pkg/requestexternal/models/request"
	"monorepo/bin-number-manager/pkg/requestexternal/models/response"
	"monorepo/bin-number-manager/pkg/requestexternal/models/telnyx"
)

var telnyxHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

func (h *requestExternal) TelnyxAvailableNumberGets(token, countryCode, locality, administrativeArea string, limit uint) ([]*telnyx.AvailableNumber, error) {

	// create a request uri
	uri := "https://api.telnyx.com/v2/available_phone_numbers"

	if countryCode == "" {
		return nil, fmt.Errorf("no country given")
	}
	uri = fmt.Sprintf("%s?filter[country_code]=%s", uri, countryCode)

	if locality != "" {
		uri = fmt.Sprintf("%s&filter[locality]=%s", uri, locality)
	}
	if administrativeArea != "" {
		uri = fmt.Sprintf("%s&filter[administrative_area]=%s", uri, administrativeArea)
	}

	if limit <= 0 || limit > 10 {
		limit = 10
	}
	uri = fmt.Sprintf("%s&filter[limit]=%d", uri, limit)

	// create a request
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a http request. err: %v", err)
	}

	client := telnyxHTTPClient

	authToken := fmt.Sprintf("Bearer %s", token)
	req.Header.Add("Authorization", authToken)

	// send a request go provider
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get correct response. err: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not get correct response. status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the response body. err: %v", err)
	}

	// parse
	resParse := response.TelnyxV2ResponseAvailableNumbersGet{}
	if err := json.Unmarshal(body, &resParse); err != nil {
		return nil, err
	}

	res := []*telnyx.AvailableNumber{}
	for i := range resParse.Data {
		res = append(res, &resParse.Data[i])
	}

	return res, nil
}

// TelnyxNumberOrdersPost sends the post request to the telnyx number_orders
func (h *requestExternal) TelnyxNumberOrdersPost(token string, numbers []string, connectionID, profileID string) (*telnyx.OrderNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TelnyxNumberOrdersPost",
		"numbers":       numbers,
		"connection_id": connectionID,
		"profile_id":    profileID,
	})

	// create a request uri
	uri := "https://api.telnyx.com/v2/number_orders"

	reqData := request.TelnyxV2DataNumberOrdersPost{
		ConnectionID:       connectionID,
		MessagingProfileID: profileID,
	}
	for _, number := range numbers {
		tmp := request.TelnyxPhoneNumber{
			PhoneNumber: number,
		}
		reqData.PhoneNumbers = append(reqData.PhoneNumbers, tmp)
	}
	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}
	log.WithField("request_data", m).Debugf("Generated request data.")

	// create a request
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(m))
	if err != nil {
		return nil, fmt.Errorf("could not create a http request. err: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := telnyxHTTPClient
	authToken := fmt.Sprintf("Bearer %s", token)
	req.Header.Add("Authorization", authToken)

	// send a request go provider
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get correct response. err: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not get correct response. status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the response body. err: %v", err)
	}

	// parse
	res := response.TelnyxV2ResponseNumberOrdersPost{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	return &res.Data, nil
}

// TelnyxPhoneNumbersGetByNumber returns number info of the given number
func (h *requestExternal) TelnyxPhoneNumbersGetByNumber(token string, number string) (*telnyx.PhoneNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "TelnyxPhoneNumbersGetByNumber",
		"number": number,
	})

	// create a request uri
	uri := fmt.Sprintf("https://api.telnyx.com/v2/phone_numbers?filter[phone_number]=%s", number)

	// create a request
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a http request. err: %v", err)
	}

	client := telnyxHTTPClient
	authToken := fmt.Sprintf("Bearer %s", token)
	req.Header.Add("Authorization", authToken)

	// send a request go provider
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get correct response. err: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	log.WithField("response", resp).Debugf("Received response.")

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not get correct response. status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the response body. err: %v", err)
	}

	// parse
	resParse := response.TelnyxV2ResponsePhoneNumbersGet{}
	if err := json.Unmarshal(body, &resParse); err != nil {
		return nil, err
	}

	if len(resParse.Data) == 0 {
		return nil, fmt.Errorf("no phone number found for number: %s", number)
	}
	return &resParse.Data[0], nil
}

// TelnyxPhoneNumbersIDGet gets the phone number info from the telnyx and return
func (h *requestExternal) TelnyxPhoneNumbersIDGet(token, id string) (*telnyx.PhoneNumber, error) {
	// create a request uri
	uri := fmt.Sprintf("https://api.telnyx.com/v2/phone_numbers/%s", id)

	// create a request
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a http request. err: %v", err)
	}

	client := telnyxHTTPClient
	authToken := fmt.Sprintf("Bearer %s", token)
	req.Header.Add("Authorization", authToken)

	// send a request go provider
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get correct response. err: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not get correct response. status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the response body. err: %v", err)
	}

	// parse
	resParse := response.TelnyxV2ResponsePhoneNumbersIDGet{}
	if err := json.Unmarshal(body, &resParse); err != nil {
		return nil, err
	}

	return &resParse.Data, nil
}

// TelnyxPhoneNumbersIDDelete delets the phone number of the given id
func (h *requestExternal) TelnyxPhoneNumbersIDDelete(token, id string) (*telnyx.PhoneNumber, error) {
	// create a request uri
	uri := fmt.Sprintf("https://api.telnyx.com/v2/phone_numbers/%s", id)

	// create a request
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a http request. err: %v", err)
	}

	client := telnyxHTTPClient
	authToken := fmt.Sprintf("Bearer %s", token)
	req.Header.Add("Authorization", authToken)

	// send a request go provider
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get correct response. err: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not get correct response. status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the response body. err: %v", err)
	}

	// parse
	resParse := response.TelnyxV2ResponsePhoneNumbersIDDelete{}
	if err := json.Unmarshal(body, &resParse); err != nil {
		return nil, err
	}

	return &resParse.Data, nil
}

// TelnyxPhoneNumbersGet gets the phone number info from the telnyx and return
func (h *requestExternal) TelnyxPhoneNumbersGet(token string, size uint, tag, number string) ([]*telnyx.PhoneNumber, error) {
	// create a request uri
	if size <= 0 {
		size = 10
	}
	uri := fmt.Sprintf("https://api.telnyx.com/v2/phone_numbers?page[size]=%d", size)

	if tag != "" {
		uri = fmt.Sprintf("%s&filter[tag]=%s", uri, tag)
	}
	if number != "" {
		uri = fmt.Sprintf("%s&filter[phone_number]=%s", uri, number)
	}

	// create a request
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a http request. err: %v", err)
	}

	client := telnyxHTTPClient
	authToken := fmt.Sprintf("Bearer %s", token)
	req.Header.Add("Authorization", authToken)

	// send a request go provider
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get correct response. err: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not get correct response. status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the response body. err: %v", err)
	}

	// parse
	resParse := response.TelnyxV2ResponsePhoneNumbersGet{}
	if err := json.Unmarshal(body, &resParse); err != nil {
		return nil, err
	}

	res := []*telnyx.PhoneNumber{}
	for _, tmp := range resParse.Data {
		res = append(res, &tmp)
	}

	return res, nil
}

// TelnyxPhoneNumbersIDUpdate updates the number info of the given id
func (h *requestExternal) TelnyxPhoneNumbersIDUpdate(token, id string, data map[string]interface{}) (*telnyx.PhoneNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "TelnyxPhoneNumbersIDUpdate",
		"id":   id,
		"data": data,
	})

	// create a request uri
	uri := fmt.Sprintf("https://api.telnyx.com/v2/phone_numbers/%s", id)

	// create data body
	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	log.WithField("body", m).Debugf("Sending a request to the telnyx. id: %s", id)

	// create a request
	req, err := http.NewRequest("PATCH", uri, bytes.NewBuffer(m))
	if err != nil {
		return nil, fmt.Errorf("could not create a http request. err: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := telnyxHTTPClient
	authToken := fmt.Sprintf("Bearer %s", token)
	req.Header.Add("Authorization", authToken)

	// send a request go provider
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get correct response. err: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not get correct response. status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the response body. err: %v", err)
	}
	log.WithField("response", body).Debugf("Received response.")

	// return body, nil
	// parse
	resParse := response.TelnyxV2ResponsePhoneNumbersIDPPatch{}
	if err := json.Unmarshal(body, &resParse); err != nil {
		return nil, err
	}

	return &resParse.Data, nil
}
