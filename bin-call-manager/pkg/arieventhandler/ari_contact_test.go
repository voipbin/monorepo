package arieventhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	rmcustomerdomain "monorepo/bin-registrar-manager/models/customerdomain"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_EventHandlerContactStatusChange(t *testing.T) {

	type test struct {
		name  string
		event *ari.ContactStatusChange

		responseCustomerDomain *rmcustomerdomain.CustomerDomain

		expectDomain  string
		expectFilters map[string]any
	}

	tests := []test{
		{
			name: "normal short realm",
			event: &ari.ContactStatusChange{
				Event: ari.Event{
					Type:        ari.EventTypeContactStatusChange,
					Application: "voipbin",
					Timestamp:   "2021-02-19T06:32:14.621Z",
					AsteriskID:  "8e:86:e2:2c:a7:51",
				},
				Endpoint: ari.Endpoint{
					Resource:   "test11@ab12.reg.voipbin.net",
					State:      ari.EndpointStateOnline,
					Technology: "PJSIP",
					ChannelIDs: []string{},
				},
				ContactInfo: ari.ContactInfo{
					AOR:           "test11@ab12.reg.voipbin.net",
					URI:           "sip:jgo101ml@r5e5vuutihlr.invalid;transport=ws",
					RoundtripUsec: "0",
					ContactStatus: ari.ContactStatusTypeNonQualified,
				},
			},

			responseCustomerDomain: &rmcustomerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("1e5dcc80-57d1-11ee-a0bc-8718bdf822a7"),
				DomainLabel: "ab12",
				Realm:       "ab12.reg.voipbin.net",
			},

			expectDomain: "ab12.reg.voipbin.net",
			expectFilters: map[string]any{
				"customer_id": uuid.FromStringOrNil("1e5dcc80-57d1-11ee-a0bc-8718bdf822a7"),
				"extension":   "test11",
			},
		},
		{
			name: "legacy uuid realm",
			event: &ari.ContactStatusChange{
				Event: ari.Event{
					Type:        ari.EventTypeContactStatusChange,
					Application: "voipbin",
					Timestamp:   "2021-02-19T06:32:14.621Z",
					AsteriskID:  "8e:86:e2:2c:a7:51",
				},
				Endpoint: ari.Endpoint{
					Resource:   "test11@1e5dcc80-57d1-11ee-a0bc-8718bdf822a7.registrar.voipbin.net",
					State:      ari.EndpointStateOnline,
					Technology: "PJSIP",
					ChannelIDs: []string{},
				},
				ContactInfo: ari.ContactInfo{
					AOR:           "test11@1e5dcc80-57d1-11ee-a0bc-8718bdf822a7.registrar.voipbin.net",
					URI:           "sip:jgo101ml@r5e5vuutihlr.invalid;transport=ws",
					RoundtripUsec: "0",
					ContactStatus: ari.ContactStatusTypeNonQualified,
				},
			},

			responseCustomerDomain: &rmcustomerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("1e5dcc80-57d1-11ee-a0bc-8718bdf822a7"),
				DomainLabel: "1e5dcc80-57d1-11ee-a0bc-8718bdf822a7",
				Realm:       "1e5dcc80-57d1-11ee-a0bc-8718bdf822a7.registrar.voipbin.net",
			},

			expectDomain: "1e5dcc80-57d1-11ee-a0bc-8718bdf822a7.registrar.voipbin.net",
			expectFilters: map[string]any{
				"customer_id": uuid.FromStringOrNil("1e5dcc80-57d1-11ee-a0bc-8718bdf822a7"),
				"extension":   "test11",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockSvc := callhandler.NewMockCallHandler(mc)

			h := eventHandler{
				db:          mockDB,
				sockHandler: mockSock,
				reqHandler:  mockReq,
				callHandler: mockSvc,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1CustomerDomainGetByRealm(ctx, tt.expectDomain).Return(tt.responseCustomerDomain, nil)
			mockReq.EXPECT().RegistrarV1ContactRefresh(ctx, tt.expectFilters).Return(nil)
			if err := h.EventHandlerContactStatusChange(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerContactStatusChange_unknownRealm(t *testing.T) {

	type test struct {
		name  string
		event *ari.ContactStatusChange

		expectDomain string
	}

	tests := []test{
		{
			name: "unknown realm skips the refresh",
			event: &ari.ContactStatusChange{
				Event: ari.Event{
					Type:        ari.EventTypeContactStatusChange,
					Application: "voipbin",
					Timestamp:   "2021-02-19T06:32:14.621Z",
					AsteriskID:  "8e:86:e2:2c:a7:51",
				},
				Endpoint: ari.Endpoint{
					Resource:   "test11@zzzz.reg.voipbin.net",
					State:      ari.EndpointStateOnline,
					Technology: "PJSIP",
					ChannelIDs: []string{},
				},
			},

			expectDomain: "zzzz.reg.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockSvc := callhandler.NewMockCallHandler(mc)

			h := eventHandler{
				db:          mockDB,
				sockHandler: mockSock,
				reqHandler:  mockReq,
				callHandler: mockSvc,
			}
			ctx := context.Background()

			// unknown realm: no RegistrarV1ContactRefresh call expected.
			mockReq.EXPECT().RegistrarV1CustomerDomainGetByRealm(ctx, tt.expectDomain).Return(nil, fmt.Errorf("not found"))
			if err := h.EventHandlerContactStatusChange(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerContactStatusChange_invalidEndpoint(t *testing.T) {

	type test struct {
		name  string
		event *ari.ContactStatusChange
	}

	tests := []test{
		{
			name: "endpoint resource without domain",
			event: &ari.ContactStatusChange{
				Event: ari.Event{
					Type:        ari.EventTypeContactStatusChange,
					Application: "voipbin",
					Timestamp:   "2021-02-19T06:32:14.621Z",
					AsteriskID:  "8e:86:e2:2c:a7:51",
				},
				Endpoint: ari.Endpoint{
					Resource:   "test11",
					State:      ari.EndpointStateOnline,
					Technology: "PJSIP",
					ChannelIDs: []string{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockSvc := callhandler.NewMockCallHandler(mc)

			h := eventHandler{
				db:          mockDB,
				sockHandler: mockSock,
				reqHandler:  mockReq,
				callHandler: mockSvc,
			}
			ctx := context.Background()

			if err := h.EventHandlerContactStatusChange(ctx, tt.event); err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
		})
	}
}
