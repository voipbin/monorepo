package listenhandler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
)

func Test_v1CampaignsFlowsReconcilePost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		expectRecentIntervalSec int64
		responseResult          campaign.ReconcileResult
	}{
		{
			"normal, clean pass",
			&sock.Request{
				URI:      "/v1/campaigns/flows/reconcile",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"recent_interval_sec":21600}`),
			},

			21600,
			campaign.ReconcileResult{
				Cleaned: 2,
				Skipped: 3,
				Failed:  0,
			},
		},
		{
			"pass with row-level failures still returns 200",
			&sock.Request{
				URI:      "/v1/campaigns/flows/reconcile",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"recent_interval_sec":3600}`),
			},

			3600,
			campaign.ReconcileResult{
				Cleaned: 0,
				Skipped: 0,
				Failed:  4,
				Partial: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().ReconcileOrphanedFlows(gomock.Any(), tt.expectRecentIntervalSec).Return(tt.responseResult, nil)

			expectData, err := json.Marshal(tt.responseResult)
			if err != nil {
				t.Fatalf("Could not marshal expected result. err: %v", err)
			}
			expectRes := &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       expectData,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expectRes.Data, res.Data)
			}
		})
	}
}

func Test_v1CampaignsFlowsReconcilePost_queryError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		campaignHandler: mockCampaign,
	}

	req := &sock.Request{
		URI:      "/v1/campaigns/flows/reconcile",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"recent_interval_sec":3600}`),
	}

	mockCampaign.EXPECT().ReconcileOrphanedFlows(gomock.Any(), int64(3600)).Return(campaign.ReconcileResult{}, fmt.Errorf("could not query the database"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. processRequest itself must not return an error (mapped to a response). got: %v", err)
	}
	if res == nil || res.StatusCode == 200 {
		t.Errorf("Wrong match. expect a non-200 response when the handler itself fails, got: %+v", res)
	}
}
