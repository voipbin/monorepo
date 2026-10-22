package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmcampaign "monorepo/bin-campaign-manager/models/campaign"
	cerrors "monorepo/bin-common-handler/models/errors"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	s1CampaignID = "f0a2c5c7-000b-11f0-8ebb-d2f3afb0c10b"
	s1OutplanID  = "01b3d6d8-000c-11f0-9fcc-e304b0c1d20c"
	s1OutdialID  = "12c4e7e9-000d-11f0-8add-f415c1d2e30d"
	s1QueueID    = "23d5f8fa-000e-11f0-9bee-0526d2e3f40e"
	s1NextCampID = "34e609ab-000f-11f0-8cff-1637e3f40a0f"
)

func Test_GetCampaigns_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/campaigns", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().CampaignGetsByCustomerID(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_PostCampaigns_Errors(t *testing.T) {
	body := `{"name":"n","detail":"d","type":"call","service_level":100,"end_handle":"stop","actions":[],"outplan_id":"` + s1OutplanID +
		`","outdial_id":"` + s1OutdialID + `","queue_id":"` + s1QueueID + `","next_campaign_id":"` + s1NextCampID + `"}`
	s1Run(t, []s1Case{
		s1Unauth("POST", "/campaigns", body),
		s1BadJSON("POST", "/campaigns"),
		s1PermDenied("POST", "/campaigns", body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignCreate(gomock.Any(), a, "n", "d", cmcampaign.Type("call"), 100, cmcampaign.EndHandle("stop"), []fmaction.Action{},
				s1ID(s1OutplanID), s1ID(s1OutdialID), s1ID(s1QueueID), s1ID(s1NextCampID)).Return(nil, err)
		}),
	})
}

func Test_CampaignsId_Errors(t *testing.T) {
	path := "/campaigns/" + s1CampaignID
	id := s1ID(s1CampaignID)
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/campaigns/not-a-uuid", ""),
		s1NotFound("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignGet(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/campaigns/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignDelete(gomock.Any(), a, id).Return(nil, err)
		}),
	})
}

func Test_PutCampaignsId_Errors(t *testing.T) {
	path := "/campaigns/" + s1CampaignID
	body := `{"name":"n2","detail":"d2","type":"call","service_level":50,"end_handle":"continue"}`
	name, detail, level := "n2", "d2", 50
	typ := cmcampaign.Type("call")
	endHandle := cmcampaign.EndHandle("continue")
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/campaigns/not-a-uuid", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignUpdateBasicInfo(gomock.Any(), a, s1ID(s1CampaignID), &name, &detail, &typ, &level, &endHandle).Return(nil, err)
		}),
		s1PermDenied("PUT", path, `{}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignUpdateBasicInfo(gomock.Any(), a, s1ID(s1CampaignID), nil, nil, nil, nil, nil).Return(nil, err)
		}),
	})
}

func Test_PutCampaignsIdStatus_Errors(t *testing.T) {
	path := "/campaigns/" + s1CampaignID + "/status"
	body := `{"status":"run"}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/campaigns/not-a-uuid/status", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignUpdateStatus(gomock.Any(), a, s1ID(s1CampaignID), cmcampaign.Status("run")).Return(nil, err)
		}),
	})
}

func Test_PutCampaignsIdServiceLevel_Errors(t *testing.T) {
	path := "/campaigns/" + s1CampaignID + "/service_level"
	body := `{"service_level":75}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/campaigns/not-a-uuid/service_level", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignUpdateServiceLevel(gomock.Any(), a, s1ID(s1CampaignID), 75).Return(nil, err)
		}),
	})
}

func Test_PutCampaignsIdActions_Errors(t *testing.T) {
	path := "/campaigns/" + s1CampaignID + "/actions"
	body := `{"actions":[]}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/campaigns/not-a-uuid/actions", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignUpdateActions(gomock.Any(), a, s1ID(s1CampaignID), []fmaction.Action{}).Return(nil, err)
		}),
	})
}

func Test_PutCampaignsIdResourceInfo_Errors(t *testing.T) {
	path := "/campaigns/" + s1CampaignID + "/resource_info"
	body := `{"outplan_id":"` + s1OutplanID + `","outdial_id":"` + s1OutdialID + `","queue_id":"` + s1QueueID + `","next_campaign_id":"` + s1NextCampID + `"}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/campaigns/not-a-uuid/resource_info", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignUpdateResourceInfo(gomock.Any(), a, s1ID(s1CampaignID), s1ID(s1OutplanID), s1ID(s1OutdialID), s1ID(s1QueueID), s1ID(s1NextCampID)).Return(nil, err)
		}),
	})
}

func Test_PutCampaignsIdNextCampaignId_Errors(t *testing.T) {
	path := "/campaigns/" + s1CampaignID + "/next_campaign_id"
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, `{"next_campaign_id":"`+s1NextCampID+`"}`),
		s1BadID("PUT", "/campaigns/not-a-uuid/next_campaign_id", `{"next_campaign_id":"`+s1NextCampID+`"}`),
		s1BadJSON("PUT", path),
		{
			name:       "non-empty next_campaign_id that is not a uuid",
			method:     "PUT",
			path:       path,
			body:       `{"next_campaign_id":"not-a-uuid"}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_ARGUMENT",
		},
		s1PermDenied("PUT", path, `{"next_campaign_id":"`+s1NextCampID+`"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignUpdateNextCampaignID(gomock.Any(), a, s1ID(s1CampaignID), s1ID(s1NextCampID)).Return(nil, err)
		}),
		s1PermDenied("PUT", path, `{"next_campaign_id":""}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaignUpdateNextCampaignID(gomock.Any(), a, s1ID(s1CampaignID), uuid.Nil).Return(nil, err)
		}),
	})
}

func Test_GetCampaignsIdCampaigncalls_Errors(t *testing.T) {
	path := "/campaigns/" + s1CampaignID + "/campaigncalls"
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/campaigns/not-a-uuid/campaigncalls", ""),
		s1PermDenied("GET", path+"?page_size=500&page_token=tok", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CampaigncallGetsByCampaignID(gomock.Any(), a, s1ID(s1CampaignID), uint64(100), "tok").Return(nil, err)
		}),
	})
}
