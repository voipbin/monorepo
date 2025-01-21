package server

import (
	"encoding/json"
	"fmt"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
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

func MarshalUnmarshal[T any](input interface{}) (T, error) {
	marshal, err := json.Marshal(input)
	if err != nil {
		return *new(T), fmt.Errorf("could not marshal the data: %w", err)
	}

	var result T
	err = json.Unmarshal(marshal, &result)
	if err != nil {
		return *new(T), fmt.Errorf("could not unmarshal the data: %w", err)
	}

	return result, nil
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

func ConvertCommonAddressToAddress(ca openapi_server.CommonAddress) commonaddress.Address {
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
