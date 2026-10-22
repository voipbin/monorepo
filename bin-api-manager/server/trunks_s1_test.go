package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"

	"go.uber.org/mock/gomock"
)

const s1TrunkID = "6d1f0c3e-0002-11f0-9b2a-4f6a1c2d3e02"

func Test_PostTrunks_Errors(t *testing.T) {
	body := `{"name":"n","detail":"d","domain_name":"dom","auth_types":["basic"],"username":"u","password":"p","allowed_ips":["192.0.2.1"]}`
	s1Run(t, []s1Case{
		s1Unauth("POST", "/trunks", body),
		s1BadJSON("POST", "/trunks"),
		s1PermDenied("POST", "/trunks", body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().TrunkCreate(gomock.Any(), a, "n", "d", "dom", []rmsipauth.AuthType{rmsipauth.AuthTypeBasic}, "u", "p", []string{"192.0.2.1"}).Return(nil, err)
		}),
	})
}

func Test_GetTrunks_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/trunks", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().TrunkList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_GetTrunksId_Errors(t *testing.T) {
	path := "/trunks/" + s1TrunkID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().TrunkGet(gomock.Any(), a, s1ID(s1TrunkID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/trunks/not-a-uuid", ""),
		s1PermDenied("GET", path, "", expect),
		s1NotFound("GET", path, "", expect),
	})
}

func Test_PutTrunksId_Errors(t *testing.T) {
	path := "/trunks/" + s1TrunkID
	body := `{"name":"n2","auth_types":["basic","ip"],"allowed_ips":["192.0.2.2"]}`
	name := "n2"
	authTypes := []rmsipauth.AuthType{rmsipauth.AuthTypeBasic, rmsipauth.AuthTypeIP}
	allowed := []string{"192.0.2.2"}
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/trunks/not-a-uuid", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().TrunkUpdateBasicInfo(gomock.Any(), a, s1ID(s1TrunkID), &name, nil, &authTypes, nil, nil, &allowed).Return(nil, err)
		}),
	})
}

func Test_DeleteTrunksId_Errors(t *testing.T) {
	path := "/trunks/" + s1TrunkID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().TrunkDelete(gomock.Any(), a, s1ID(s1TrunkID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/trunks/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", expect),
	})
}
