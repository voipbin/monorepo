package server

import (
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	chmedia "monorepo/bin-chat-manager/models/media"
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type Server interface{}

type server struct {
	serviceHandler servicehandler.ServiceHandler
}

func NewServer(serviceHandler servicehandler.ServiceHandler) openapi_server.ServerInterface {
	return &server{
		serviceHandler: serviceHandler,
	}
}

func ConvertCommonAddress(ca openapi_server.CommonAddress) commonaddress.Address {
	safeString := func(s *string) string {
		if s != nil {
			return *s
		}
		return ""
	}

	return commonaddress.Address{
		Type:       commonaddress.Type(safeString((*string)(ca.Type))),
		Target:     safeString(ca.Target),
		TargetName: safeString(ca.TargetName),
		Name:       safeString(ca.Name),
		Detail:     safeString(ca.Detail),
	}
}

func ConvertFlowManagerAction(fma openapi_server.FlowManagerAction) fmaction.Action {
	id := uuid.FromStringOrNil(fma.Id)

	nextID := uuid.Nil
	if fma.NextId != nil && *fma.NextId != "" {
		nextID = uuid.FromStringOrNil(*fma.NextId)
	}

	var option json.RawMessage
	if fma.Option != nil {
		optionBytes, err := json.Marshal(fma.Option)
		if err == nil {
			option = optionBytes
		}
	}

	res := fmaction.Action{
		ID:        id,
		NextID:    nextID,
		Type:      fmaction.Type(fma.Type),
		Option:    option,
		TMExecute: "",
	}

	if fma.TmExecute != nil {
		res.TMExecute = *fma.TmExecute
	}

	return res
}

func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func convertAgent(agentManager openapi_server.AgentManagerAgent) amagent.Agent {
	id := uuid.Nil
	if agentManager.Id != nil {
		id = uuid.FromStringOrNil(*agentManager.Id)
	}

	customerID := uuid.Nil
	if agentManager.CustomerId != nil {
		customerID = uuid.FromStringOrNil(*agentManager.CustomerId)
	}

	tagIDs := []uuid.UUID{}
	if agentManager.TagIds != nil {
		for _, v := range *agentManager.TagIds {
			tagIDs = append(tagIDs, uuid.FromStringOrNil(v))
		}
	}

	var addresses []commonaddress.Address
	if agentManager.Addresses != nil {
		for _, v := range *agentManager.Addresses {
			addresses = append(addresses, ConvertCommonAddress(v))
		}
	}

	res := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		Username:     derefString(agentManager.Username),
		PasswordHash: "",

		Name:   derefString(agentManager.Name),
		Detail: derefString(agentManager.Detail),

		TagIDs:    tagIDs,
		Addresses: addresses,
	}

	if agentManager.RingMethod != nil {
		res.RingMethod = amagent.RingMethod(*agentManager.RingMethod)
	}

	if agentManager.Status != nil {
		res.Status = amagent.Status(*agentManager.Status)
	}

	if agentManager.Permission != nil {
		res.Permission = amagent.Permission(*agentManager.Permission)
	}

	return res

}

func ConvertChatManagerMedia(input openapi_server.ChatManagerMedia) chmedia.Media {
	mediaType := ""
	if input.Type != nil {
		mediaType = string(*input.Type)
	}

	fileID := uuid.Nil
	if input.FileId != nil {
		fileID = uuid.FromStringOrNil(*input.FileId) // Handle invalid UUID parsing as needed
	}

	address := commonaddress.Address{}
	if input.Address != nil {
		address = ConvertCommonAddress(*input.Address)
	}

	agent := amagent.Agent{}
	if input.Agent != nil {
		agent = convertAgent(*input.Agent)
	}

	return chmedia.Media{
		Type:    chmedia.Type(mediaType),
		Address: address,
		Agent:   agent,
		FileID:  fileID,
		LinkURL: derefString(input.LinkUrl),
	}
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(s int) *int {
	return &s
}

func GenerateListResponse[T any](tmps []*T, nextTokenValue string) struct {
	Result []*T `json:"result"`
	openapi_server.CommonPagination
} {
	nextToken := ""
	if len(tmps) > 0 {
		nextToken = nextTokenValue
	}

	return struct {
		Result []*T `json:"result"`
		openapi_server.CommonPagination
	}{
		Result: tmps,
		CommonPagination: openapi_server.CommonPagination{
			NextPageToken: &nextToken,
		},
	}
}
