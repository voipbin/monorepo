package server

import (
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"

	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentsMe_Errors(t *testing.T) {
	const path = "/service_agents/me"
	name, detail := "n", "d"
	ring := amagent.RingMethod("ringall")

	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1PermDenied("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentMeGet(gomock.Any(), a).Return(nil, err)
		}),

		s1Unauth("PUT", path, `{"name":"n"}`),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, `{"name":"n","detail":"d","ring_method":"ringall"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentMeUpdate(gomock.Any(), a, &name, &detail, &ring).Return(nil, err)
		}),
		s1PermDenied("PUT", path, `{}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentMeUpdate(gomock.Any(), a, (*string)(nil), (*string)(nil), (*amagent.RingMethod)(nil)).Return(nil, err)
		}),

		s1Unauth("PUT", path+"/addresses", `{"addresses":[]}`),
		s1BadJSON("PUT", path+"/addresses"),
		s1PermDenied("PUT", path+"/addresses", `{"addresses":[{"type":"tel","target":"+15555550100"}]}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentMeUpdateAddresses(gomock.Any(), a, []commonaddress.Address{{Type: commonaddress.Type("tel"), Target: "+15555550100"}}).Return(nil, err)
		}),

		s1Unauth("PUT", path+"/status", `{"status":"available"}`),
		s1BadJSON("PUT", path+"/status"),
		s1PermDenied("PUT", path+"/status", `{"status":"available"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentMeUpdateStatus(gomock.Any(), a, amagent.Status("available")).Return(nil, err)
		}),

		s1Unauth("PUT", path+"/password", `{"password":"pw"}`),
		s1BadJSON("PUT", path+"/password"),
		s1PermDenied("PUT", path+"/password", `{"password":"pw"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentMeUpdatePassword(gomock.Any(), a, "pw").Return(nil, err)
		}),
	})
}
