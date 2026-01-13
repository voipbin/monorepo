package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_outdialsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseOutdials []*omoutdial.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials?page_size=10&page_token=2021-03-02%2003%3A23%3A20.995000",

			responseOutdials: []*omoutdial.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("438f0ccc-c64a-11ec-9ac6-b729ca9f28bf"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-02 03:23:20.995000",
			expectRes:       `{"result":[{"id":"438f0ccc-c64a-11ec-9ac6-b729ca9f28bf","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials?page_size=10&page_token=2021-03-02%2003%3A23%3A20.995000",

			responseOutdials: []*omoutdial.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ad4ec08a-c64a-11ec-ad4d-2b9c85718834"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b088244e-c64a-11ec-afb8-2f6ebf108ed8"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c3e247e0-c64a-11ec-b415-c786d3fa957c"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-02 03:23:20.995000",
			expectRes:       `{"result":[{"id":"ad4ec08a-c64a-11ec-ad4d-2b9c85718834","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"b088244e-c64a-11ec-afb8-2f6ebf108ed8","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"c3e247e0-c64a-11ec-b415-c786d3fa957c","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().OutdialGetsByCustomerID(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken.Return(tt.responseOutdials, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseOutdial *omoutdial.WebhookMessage

		expectCampaignID uuid.UUID
		expectName       string
		expectDetail     string
		expectData       string
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials",
			reqBody:  []byte(`{"campaign_id":"5770a50e-1a94-45fc-9ba1-79064573cf06","name":"test name","detail":"test detail","data":"test data"}`),

			responseOutdial: &omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("99b197a5-010e-4f4e-b9fc-aae44e241ddb"),
				},
			},

			expectCampaignID: uuid.FromStringOrNil("5770a50e-1a94-45fc-9ba1-79064573cf06"),
			expectName:       "test name",
			expectDetail:     "test detail",
			expectData:       "test data",
			expectRes:        `{"id":"99b197a5-010e-4f4e-b9fc-aae44e241ddb","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", "/outdials", bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialCreate(req.Context(), &tt.agent, tt.expectCampaignID, tt.expectName, tt.expectDetail, tt.expectData.Return(tt.responseOutdial, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseOutdial *omoutdial.WebhookMessage

		expectOutdialID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/3bb463ed-aa6e-4b64-9fc5-b1fc62096b67",

			responseOutdial: &omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3bb463ed-aa6e-4b64-9fc5-b1fc62096b67"),
				},
			},

			expectOutdialID: uuid.FromStringOrNil("3bb463ed-aa6e-4b64-9fc5-b1fc62096b67"),
			expectRes:       `{"id":"3bb463ed-aa6e-4b64-9fc5-b1fc62096b67","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().OutdialGet(req.Context(), &tt.agent, tt.expectOutdialID.Return(tt.responseOutdial, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseOutdial *omoutdial.WebhookMessage

		expectOutdialID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/37877e5d-ebe3-492d-be8c-54d62e98b4db",

			responseOutdial: &omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("37877e5d-ebe3-492d-be8c-54d62e98b4db"),
				},
			},

			expectOutdialID: uuid.FromStringOrNil("37877e5d-ebe3-492d-be8c-54d62e98b4db"),
			expectRes:       `{"id":"37877e5d-ebe3-492d-be8c-54d62e98b4db","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().OutdialDelete(req.Context(), &tt.agent, tt.expectOutdialID.Return(tt.responseOutdial, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseOutdial *omoutdial.WebhookMessage

		expectOutdialID uuid.UUID
		expectName      string
		expectDetail    string
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/38114323-144a-499c-bfde-f3d8af114a7a",
			reqBody:  []byte(`{"name":"test name","detail":"test detail"}`),

			responseOutdial: &omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("38114323-144a-499c-bfde-f3d8af114a7a"),
				},
			},

			expectOutdialID: uuid.FromStringOrNil("38114323-144a-499c-bfde-f3d8af114a7a"),
			expectName:      "test name",
			expectDetail:    "test detail",
			expectRes:       `{"id":"38114323-144a-499c-bfde-f3d8af114a7a","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", "/outdials/"+tt.expectOutdialID.String(), bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialUpdateBasicInfo(req.Context(), &tt.agent, tt.expectOutdialID, tt.expectName, tt.expectDetail.Return(tt.responseOutdial, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsIDCampaignIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseOutdial *omoutdial.WebhookMessage

		expectOutdialID  uuid.UUID
		expectCampaignID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/607ea822-121d-4a52-ad3f-3f5320445ec8/campaign_id",
			reqBody:  []byte(`{"campaign_id":"caad42fb-8266-4a24-be3f-9963ba14a20a"}`),

			responseOutdial: &omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("607ea822-121d-4a52-ad3f-3f5320445ec8"),
				},
			},

			expectOutdialID:  uuid.FromStringOrNil("607ea822-121d-4a52-ad3f-3f5320445ec8"),
			expectCampaignID: uuid.FromStringOrNil("caad42fb-8266-4a24-be3f-9963ba14a20a"),
			expectRes:        `{"id":"607ea822-121d-4a52-ad3f-3f5320445ec8","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialUpdateCampaignID(req.Context(), &tt.agent, tt.expectOutdialID, tt.expectCampaignID.Return(tt.responseOutdial, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsIDDataPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseOutdial *omoutdial.WebhookMessage

		expectOutdialID uuid.UUID
		expectData      string
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/056eefbd-8afa-402a-8c9e-681040ec8803/data",
			reqBody:  []byte(`{"data":"test data"}`),

			responseOutdial: &omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("056eefbd-8afa-402a-8c9e-681040ec8803"),
				},
			},

			expectOutdialID: uuid.FromStringOrNil("056eefbd-8afa-402a-8c9e-681040ec8803"),
			expectData:      "test data",
			expectRes:       `{"id":"056eefbd-8afa-402a-8c9e-681040ec8803","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialUpdateData(req.Context(), &tt.agent, tt.expectOutdialID, tt.expectData.Return(tt.responseOutdial, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsIDTargetsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseOutdialtarget *omoutdialtarget.WebhookMessage

		expectOutdialID    uuid.UUID
		expectName         string
		expectDetail       string
		expectData         string
		expectDestination0 *commonaddress.Address
		expectDestination1 *commonaddress.Address
		expectDestination2 *commonaddress.Address
		expectDestination3 *commonaddress.Address
		expectDestination4 *commonaddress.Address
		expectRes          string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/726d6b88-2028-44fe-a415-a58067d98acf/targets",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","data":"test data","destination_0":{"type":"tel","target":"+821100000001"},"destination_1":{"type":"tel","target":"+821100000002"},"destination_2":{"type":"tel","target":"+821100000003"},"destination_3":{"type":"tel","target":"+821100000004"},"destination_4":{"type":"tel","target":"+821100000005"}}`),

			responseOutdialtarget: &omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("e3097653-4c68-4915-add3-78b12a4ba151"),
			},

			expectOutdialID: uuid.FromStringOrNil("726d6b88-2028-44fe-a415-a58067d98acf"),
			expectName:      "test name",
			expectDetail:    "test detail",
			expectData:      "test data",
			expectDestination0: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDestination1: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			expectDestination2: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000003",
			},
			expectDestination3: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000004",
			},
			expectDestination4: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000005",
			},
			expectRes: `{"id":"e3097653-4c68-4915-add3-78b12a4ba151","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialtargetCreate(req.Context(), &tt.agent, tt.expectOutdialID, tt.expectName, tt.expectDetail, tt.expectData, tt.expectDestination0, tt.expectDestination1, tt.expectDestination2, tt.expectDestination3, tt.expectDestination4.Return(tt.responseOutdialtarget, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsIDTargetsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseOutdialtarget *omoutdialtarget.WebhookMessage

		expectOutdialID       uuid.UUID
		expectOutdialtargetID uuid.UUID
		expectRes             string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/112950f8-e3d3-4585-b858-125a59f8f51f/targets/86a52dde-c523-11ec-a8b0-53d9628a5d7f",

			responseOutdialtarget: &omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("86a52dde-c523-11ec-a8b0-53d9628a5d7f"),
			},

			expectOutdialID:       uuid.FromStringOrNil("112950f8-e3d3-4585-b858-125a59f8f51f"),
			expectOutdialtargetID: uuid.FromStringOrNil("86a52dde-c523-11ec-a8b0-53d9628a5d7f"),
			expectRes:             `{"id":"86a52dde-c523-11ec-a8b0-53d9628a5d7f","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialtargetGet(req.Context(), &tt.agent, tt.expectOutdialID, tt.expectOutdialtargetID.Return(tt.responseOutdialtarget, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsIDTargetsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseOutdialtarget *omoutdialtarget.WebhookMessage

		expectOutdialID       uuid.UUID
		expectOutdialtargetID uuid.UUID
		expectRes             string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/112950f8-e3d3-4585-b858-125a59f8f51f/targets/0adb2487-eea7-4ec9-bb7f-b2b2aa5af49e",

			responseOutdialtarget: &omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("0adb2487-eea7-4ec9-bb7f-b2b2aa5af49e"),
			},

			expectOutdialID:       uuid.FromStringOrNil("112950f8-e3d3-4585-b858-125a59f8f51f"),
			expectOutdialtargetID: uuid.FromStringOrNil("0adb2487-eea7-4ec9-bb7f-b2b2aa5af49e"),
			expectRes:             `{"id":"0adb2487-eea7-4ec9-bb7f-b2b2aa5af49e","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialtargetDelete(req.Context(), &tt.agent, tt.expectOutdialID, tt.expectOutdialtargetID.Return(tt.responseOutdialtarget, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_outdialsIDTargetGET(t *testing.T) {
	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseOutdialtargets []*omoutdialtarget.WebhookMessage

		expectOutdialID uuid.UUID
		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/fe7a06b6-c82c-11ec-89fd-f741623099f0/targets?page_size=10&page_token=2020-09-20%2003:23:21.995000",

			responseOutdialtargets: []*omoutdialtarget.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("80fcacd4-c82c-11ec-b008-67e3b5299bec"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectOutdialID: uuid.FromStringOrNil("fe7a06b6-c82c-11ec-89fd-f741623099f0"),
			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:21.995000",
			expectRes:       `{"result":[{"id":"80fcacd4-c82c-11ec-b008-67e3b5299bec","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/outdials/33d8b93c-c82e-11ec-b630-f304b7d48448/targets?page_size=15&page_token=2020-09-20%2003:23:21.995000",

			responseOutdialtargets: []*omoutdialtarget.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("340757d8-c82e-11ec-92ef-235422080f76"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("34353180-c82e-11ec-b8f2-87eaa2dc5a1b"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("61f53c3c-c82e-11ec-ba3d-f387359c8014"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectOutdialID: uuid.FromStringOrNil("33d8b93c-c82e-11ec-b630-f304b7d48448"),
			expectPageSize:  15,
			expectPageToken: "2020-09-20 03:23:21.995000",
			expectRes:       `{"result":[{"id":"340757d8-c82e-11ec-92ef-235422080f76","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"34353180-c82e-11ec-b8f2-87eaa2dc5a1b","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"61f53c3c-c82e-11ec-ba3d-f387359c8014","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().OutdialtargetGetsByOutdialID(req.Context(), &tt.agent, tt.expectOutdialID, tt.expectPageSize, tt.expectPageToken.Return(tt.responseOutdialtargets, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body.String())
			}
		})
	}
}
