package arieventhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_EventHandlerContactStatusChange(t *testing.T) {

	type test struct {
		name  string
		event *ari.ContactStatusChange

		expectCustomerID uuid.UUID
		expectextension  string
	}

	tests := []test{
		{
			"normal",
			&ari.ContactStatusChange{
				Event: ari.Event{
					Type:        ari.EventTypeContactStatusChange,
					Application: "voipbin",
					Timestamp:   "2021-02-19T06:32:14.621",
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
			uuid.FromStringOrNil("1e5dcc80-57d1-11ee-a0bc-8718bdf822a7"),
			"test11",
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

			mockReq.EXPECT().RegistrarV1ContactRefresh(ctx, tt.expectCustomerID, tt.expectextension).Return(nil)
			if err := h.EventHandlerContactStatusChange(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
