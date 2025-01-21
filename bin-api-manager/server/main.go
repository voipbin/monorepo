package server

import (
	"encoding/json"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"

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
