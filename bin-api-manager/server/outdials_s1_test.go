package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"

	"go.uber.org/mock/gomock"
)

const (
	s1OutdialIDPath = "43f71abc-0010-11f0-9d00-2748f4a50b10"
	s1TargetID      = "54082bcd-0011-11f0-8e11-3859a5b61c11"
)

func s1InvalidArg(name, method, path, body string) s1Case {
	return s1Case{
		name:       name,
		method:     method,
		path:       path,
		body:       body,
		wantStatus: cerrors.StatusInvalidArgument,
		wantReason: "INVALID_ARGUMENT",
	}
}

func Test_GetOutdials_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/outdials", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().OutdialGetsByCustomerID(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_PostOutdials_Errors(t *testing.T) {
	full := `{"campaign_id":"` + s1CampaignID + `","name":"n","detail":"d","data":"x"}`
	s1Run(t, []s1Case{
		s1Unauth("POST", "/outdials", full),
		s1BadJSON("POST", "/outdials"),
		s1InvalidArg("missing campaign_id", "POST", "/outdials", `{"name":"n","detail":"d","data":"x"}`),
		s1InvalidArg("missing name", "POST", "/outdials", `{"campaign_id":"`+s1CampaignID+`","detail":"d","data":"x"}`),
		s1InvalidArg("missing detail", "POST", "/outdials", `{"campaign_id":"`+s1CampaignID+`","name":"n","data":"x"}`),
		s1InvalidArg("missing data", "POST", "/outdials", `{"campaign_id":"`+s1CampaignID+`","name":"n","detail":"d"}`),
		s1PermDenied("POST", "/outdials", full, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialCreate(gomock.Any(), a, s1ID(s1CampaignID), "n", "d", "x").Return(nil, err)
		}),
	})
}

func Test_OutdialsId_Errors(t *testing.T) {
	path := "/outdials/" + s1OutdialIDPath
	id := s1ID(s1OutdialIDPath)
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/outdials/not-a-uuid", ""),
		s1NotFound("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialGet(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/outdials/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialDelete(gomock.Any(), a, id).Return(nil, err)
		}),
	})
}

func Test_PutOutdialsId_Errors(t *testing.T) {
	path := "/outdials/" + s1OutdialIDPath
	body := `{"name":"n2","detail":"d2"}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/outdials/not-a-uuid", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialUpdateBasicInfo(gomock.Any(), a, s1ID(s1OutdialIDPath), "n2", "d2").Return(nil, err)
		}),
	})
}

func Test_PutOutdialsIdCampaignId_Errors(t *testing.T) {
	path := "/outdials/" + s1OutdialIDPath + "/campaign_id"
	body := `{"campaign_id":"` + s1CampaignID + `"}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/outdials/not-a-uuid/campaign_id", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialUpdateCampaignID(gomock.Any(), a, s1ID(s1OutdialIDPath), s1ID(s1CampaignID)).Return(nil, err)
		}),
	})
}

func Test_PutOutdialsIdData_Errors(t *testing.T) {
	path := "/outdials/" + s1OutdialIDPath + "/data"
	body := `{"data":"new-data"}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/outdials/not-a-uuid/data", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialUpdateData(gomock.Any(), a, s1ID(s1OutdialIDPath), "new-data").Return(nil, err)
		}),
	})
}

func Test_PostOutdialsIdTargets_Errors(t *testing.T) {
	path := "/outdials/" + s1OutdialIDPath + "/targets"
	dest := `"destination_0":{"type":"tel","target":"+15555550100"},"destination_1":{},"destination_2":{},"destination_3":{},"destination_4":{}`
	full := `{"name":"n","detail":"d","data":"x",` + dest + `}`
	s1Run(t, []s1Case{
		s1Unauth("POST", path, full),
		s1BadID("POST", "/outdials/not-a-uuid/targets", full),
		s1BadJSON("POST", path),
		s1InvalidArg("missing name", "POST", path, `{"detail":"d","data":"x",`+dest+`}`),
		s1InvalidArg("missing detail", "POST", path, `{"name":"n","data":"x",`+dest+`}`),
		s1InvalidArg("missing data", "POST", path, `{"name":"n","detail":"d",`+dest+`}`),
		s1InvalidArg("no destination target", "POST", path, `{"name":"n","detail":"d","data":"x"}`),
		s1PermDenied("POST", path, full, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			d0 := commonaddress.Address{Type: commonaddress.Type("tel"), Target: "+15555550100"}
			empty := commonaddress.Address{}
			m.EXPECT().OutdialtargetCreate(gomock.Any(), a, s1ID(s1OutdialIDPath), "n", "d", "x", &d0, &empty, &empty, &empty, &empty).Return(nil, err)
		}),
	})
}

func Test_OutdialsIdTargetsTargetId_Errors(t *testing.T) {
	path := "/outdials/" + s1OutdialIDPath + "/targets/" + s1TargetID
	badTarget := "/outdials/" + s1OutdialIDPath + "/targets/not-a-uuid"
	badOutdial := "/outdials/not-a-uuid/targets/" + s1TargetID

	badTargetCase := func(method string) s1Case {
		return s1Case{
			name:       "invalid target id",
			method:     method,
			path:       badTarget,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_ID",
		}
	}

	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", badOutdial, ""),
		badTargetCase("GET"),
		s1NotFound("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialtargetGet(gomock.Any(), a, s1ID(s1OutdialIDPath), s1ID(s1TargetID)).Return(nil, err)
		}),
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", badOutdial, ""),
		badTargetCase("DELETE"),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialtargetDelete(gomock.Any(), a, s1ID(s1OutdialIDPath), s1ID(s1TargetID)).Return(nil, err)
		}),
	})
}

func Test_GetOutdialsIdTargets_Errors(t *testing.T) {
	path := "/outdials/" + s1OutdialIDPath + "/targets"
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/outdials/not-a-uuid/targets", ""),
		s1PermDenied("GET", path+"?page_size=500&page_token=tok", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().OutdialtargetGetsByOutdialID(gomock.Any(), a, s1ID(s1OutdialIDPath), uint64(100), "tok").Return(nil, err)
		}),
	})
}
