package requestexternal

// func TestTelnyxAvailableNumberGets(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	h := requestExternal{}

// 	tests := []struct {
// 		name               string
// 		country            string
// 		locality           string
// 		administrativeArea string
// 		limit              int
// 	}{
// 		{
// 			"normal us",
// 			"us",
// 			"",
// 			"",
// 			1,
// 		},
// 		{
// 			"multiple numbers us",
// 			"us",
// 			"",
// 			"",
// 			3,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			res, err := h.TelnyxAvailableNumberGets(testToken, tt.country, tt.locality, tt.administrativeArea, uint(tt.limit))
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if len(res) != tt.limit {
// 				t.Errorf("Wrong match. expect: %d, got: %d", tt.limit, len(res))
// 			}

// 			time.Sleep(time.Second * 1)
// 		})
// 	}
// }

// func Test_TelnyxPhoneNumbersIDGet(t *testing.T) {

// 	tests := []struct {
// 		name      string
// 		id        string
// 		expectRes *telnyx.PhoneNumber
// 	}{
// 		{
// 			"normal us number",
// 			"1748688147379652251",
// 			&telnyx.PhoneNumber{
// 				ID:          "1748688147379652251",
// 				RecordType:  "phone_number",
// 				PhoneNumber: "+14703298699",
// 				Status:      "active",
// 				Tags: []string{
// 					"CustomerID_5e4a0680-804e-11ec-8477-2fea5968d85b",
// 					"NumberID_12f3058e-2eb8-11ec-bd0a-bf95e97c5f5c",
// 				},
// 				ConnectionID:          "2054833017033065613",
// 				ConnectionName:        "voipbin prod",
// 				CustomerReference:     "",
// 				ExternalPin:           "",
// 				T38FaxGatewayEnabled:  true,
// 				PurchasedAt:           "2021-10-16T17:31:11Z",
// 				BillingGroupID:        "",
// 				EmergencyEnabled:      false,
// 				EmergencyAddressID:    "",
// 				EmergencyStatus:       "disabled",
// 				CallForwardingEnabled: true,
// 				CNAMListingEnabled:    false,
// 				CallRecordingEnabled:  false,
// 				PhoneNumberType:       "local",
// 				MessagingProfileID:    testProfileID,
// 				MessagingProfileName:  "voipbin production",
// 				NumberBlockID:         "",
// 				CreatedAt:             "2021-10-16T17:31:11.737Z",
// 				NumberLevelRouting:    "disabled",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			h := requestExternal{}

// 			res, err := h.TelnyxPhoneNumbersIDGet(testToken, tt.id)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			tt.expectRes.UpdatedAt = res.UpdatedAt
// 			if !reflect.DeepEqual(tt.expectRes, res) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

// func Test_TelnyxPhoneNumbersGetByNumber(t *testing.T) {

// 	tests := []struct {
// 		name      string
// 		number    string
// 		expectRes *telnyx.PhoneNumber
// 	}{
// 		{
// 			"normal us number",
// 			"14703298699",
// 			&telnyx.PhoneNumber{
// 				ID:          "1748688147379652251",
// 				RecordType:  "phone_number",
// 				PhoneNumber: "+14703298699",
// 				Status:      "active",
// 				Tags: []string{
// 					"CustomerID_5e4a0680-804e-11ec-8477-2fea5968d85b",
// 					"NumberID_12f3058e-2eb8-11ec-bd0a-bf95e97c5f5c",
// 				},
// 				ConnectionID:          "2054833017033065613",
// 				ConnectionName:        "voipbin prod",
// 				CustomerReference:     "",
// 				ExternalPin:           "",
// 				T38FaxGatewayEnabled:  true,
// 				PurchasedAt:           "2021-10-16T17:31:11Z",
// 				BillingGroupID:        "",
// 				EmergencyEnabled:      false,
// 				EmergencyAddressID:    "",
// 				EmergencyStatus:       "disabled",
// 				CallForwardingEnabled: true,
// 				CNAMListingEnabled:    false,
// 				CallRecordingEnabled:  false,
// 				PhoneNumberType:       "local",
// 				MessagingProfileID:    testProfileID,
// 				MessagingProfileName:  "voipbin production",
// 				NumberBlockID:         "",
// 				CreatedAt:             "2021-10-16T17:31:11.737Z",
// 				NumberLevelRouting:    "disabled",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			h := requestExternal{}

// 			res, err := h.TelnyxPhoneNumbersGetByNumber(testToken, tt.number)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			tt.expectRes.UpdatedAt = res.UpdatedAt
// 			if reflect.DeepEqual(tt.expectRes, res) != true {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

// // Number creation/delete test
// // Careful!!! this test does actually create the number and delete
// // it takes a actual money!
// func TestTelnyxPhoneNumbersCreateDelete(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		token string
// 		// id        string
// 		// expectRes *telnyx.PhoneNumber
// 	}{
// 		{
// 			name: "normal",

// 			token: "",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			h := requestExternal{}

// 			// get available numbers
// 			availNumbers, err := h.TelnyxAvailableNumberGets(tt.token, "US", "", "", 1)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 			t.Errorf("available numbers: %v", availNumbers[0])

// 			// // order numbers
// 			// orderNumber, err := reqHandler.TelnyxNumberOrdersPost([]string{availNumbers[0].PhoneNumber})
// 			// if err != nil {
// 			// 	t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			// }

// 			// tmp, err := h.TelnyxPhoneNumbersGet(tt.token, 1, "", availNumbers[0].PhoneNumber)
// 			// if err != nil {
// 			// 	t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			// }
// 			// t.Errorf("res: %v", tmp)

// 			// // delete numbers
// 			// if _, err := h.TelnyxPhoneNumbersIDDelete(tmp[0].ID); err != nil {
// 			// 	t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			// }
// 		})
// 	}
// }

// func Test_TelnyxPhoneNumbersIDUpdate(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		id   string
// 		data map[string]interface{}

// 		expectRes *telnyx.PhoneNumber
// 	}{
// 		{
// 			name: "normal",

// 			id: "1748688147379652251",
// 			data: map[string]interface{}{
// 				"tags": []string{
// 					"CustomerID_5e4a0680-804e-11ec-8477-2fea5968d85b",
// 					"NumberID_12f3058e-2eb8-11ec-bd0a-bf95e97c5f5c",
// 				},
// 			},

// 			expectRes: &telnyx.PhoneNumber{
// 				ID:          "1748688147379652251",
// 				RecordType:  "phone_number",
// 				PhoneNumber: "+14703298699",
// 				Status:      "active",
// 				Tags: []string{
// 					"CustomerID_5e4a0680-804e-11ec-8477-2fea5968d85b",
// 					"NumberID_12f3058e-2eb8-11ec-bd0a-bf95e97c5f5c",
// 				},
// 				ConnectionID:          "2054833017033065613",
// 				ConnectionName:        "voipbin prod",
// 				CustomerReference:     "",
// 				ExternalPin:           "",
// 				T38FaxGatewayEnabled:  true,
// 				PurchasedAt:           "2021-10-16T17:31:11Z",
// 				BillingGroupID:        "",
// 				EmergencyEnabled:      false,
// 				EmergencyAddressID:    "",
// 				EmergencyStatus:       "disabled",
// 				CallForwardingEnabled: true,
// 				CNAMListingEnabled:    false,
// 				CallRecordingEnabled:  false,
// 				PhoneNumberType:       "local",
// 				MessagingProfileID:    testProfileID,
// 				MessagingProfileName:  "voipbin production",
// 				NumberBlockID:         "",
// 				CreatedAt:             "2021-10-16T17:31:11.737Z",
// 				NumberLevelRouting:    "disabled",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			h := requestExternal{}

// 			res, err := h.TelnyxPhoneNumbersIDUpdate(testToken, tt.id, tt.data)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res.UpdatedAt = ""
// 			if !reflect.DeepEqual(tt.expectRes, res) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

// func Test_tmp(t *testing.T) {

// 	id := "1748688147379652251"
// 	// data := map[string]interface{}{
// 	// 	"tags": []string{
// 	// 		"CustomerID_5e4a0680-804e-11ec-8477-2fea5968d85b",
// 	// 		"NumberID_12f3058e-2eb8-11ec-bd0a-bf95e97c5f5c",
// 	// 	}}

// 	// create a request uri
// 	uri := fmt.Sprintf("https://api.telnyx.com/v2/phone_numbers/%s", id)

// 	// create a request
// 	req, err := http.NewRequest("GET", uri, nil)
// 	if err != nil {
// 		t.Errorf("could not create a http request. err: %v", err)
// 	}

// 	client := &http.Client{}
// 	authToken := fmt.Sprintf("Bearer %s", testToken)
// 	req.Header.Add("Authorization", authToken)

// 	// // create a request uri
// 	// uri := fmt.Sprintf("https://api.telnyx.com/v2/phone_numbers/%s", id)

// 	// // create data body
// 	// m, err := json.Marshal(data)
// 	// if err != nil {
// 	// 	t.Errorf("err: %v", err)
// 	// }

// 	// // create a request
// 	// req, err := http.NewRequest("PATCH", uri, bytes.NewBuffer(m))
// 	// if err != nil {
// 	// 	t.Errorf("could not create a http request. err: %v", err)
// 	// }
// 	// req.Header.Set("Content-Type", "application/json")

// 	// client := &http.Client{}
// 	// authToken := fmt.Sprintf("Bearer %s", testToken)
// 	// req.Header.Add("Authorization", authToken)

// 	// send a request go provider
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		t.Errorf("could not get correct response. err: %v", err)
// 	}
// 	defer func() {
// _ = resp.Body.Close()
// }()

// 	if resp.StatusCode != 200 {
// 		t.Errorf("could not get correct response. status: %d", resp.StatusCode)
// 	}

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		t.Errorf("could not read the response body. err: %v", err)
// 	}

// 	t.Errorf("Response. body: %v", body)

// 	// tests := []struct {
// 	// 	name string

// 	// 	id   string
// 	// 	data map[string]interface{}

// 	// 	expectRes *telnyx.PhoneNumber
// 	// }{
// 	// 	{
// 	// 		name: "normal",

// 	// 		id: "1748688147379652251",
// 	// 		data: map[string]interface{}{
// 	// 			"tags": []string{
// 	// 				"tag1",
// 	// 				"tag2",
// 	// 			},
// 	// 		},
// 	// 		// "14703298699",

// 	// 		// &telnyx.PhoneNumber{
// 	// 		// 	ID:                    "1748688147379652251",
// 	// 		// 	RecordType:            "phone_number",
// 	// 		// 	PhoneNumber:           "+14703298699",
// 	// 		// 	Status:                "active",
// 	// 		// 	Tags:                  []string{},
// 	// 		// 	ConnectionID:          "2054833017033065613",
// 	// 		// 	CustomerReference:     "",
// 	// 		// 	ExternalPin:           "",
// 	// 		// 	T38FaxGatewayEnabled:  true,
// 	// 		// 	PurchasedAt:           "2021-10-16T17:31:11Z",
// 	// 		// 	BillingGroupID:        "",
// 	// 		// 	EmergencyEnabled:      false,
// 	// 		// 	EmergencyAddressID:    "",
// 	// 		// 	CallForwardingEnabled: true,
// 	// 		// 	CNAMListingEnabled:    false,
// 	// 		// 	CallRecordingEnabled:  false,
// 	// 		// 	MessagingProfileID:    testProfileID,
// 	// 		// 	MessagingProfileName:  "",
// 	// 		// 	NumberBlockID:         "",
// 	// 		// 	CreatedAt:             "2021-10-16T17:31:11.737Z",
// 	// 		// },
// 	// 	},
// 	// }

// 	// for _, tt := range tests {
// 	// 	t.Run(tt.name, func(t *testing.T) {
// 	// 		mc := gomock.NewController(t)
// 	// 		defer mc.Finish()

// 	// 		h := requestExternal{}

// 	// 		res, err := h.TelnyxPhoneNumbersIDUpdate(testToken, tt.id, tt.data)
// 	// 		if err != nil {
// 	// 			t.Errorf("Wrong match. expect: ok, got: %v", err)
// 	// 		}

// 	// 		t.Errorf("response data: %v", res)

// 	// 		// tt.expectRes.UpdatedAt = res.UpdatedAt
// 	// 		// tt.expectRes.MessagingProfileName = res.MessagingProfileName
// 	// 		// if reflect.DeepEqual(tt.expectRes, res) != true {
// 	// 		// 	t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
// 	// 		// }
// 	// 	})
// 	// }
// }
