package server

import (
	"encoding/json"
	"fmt"
	"monorepo/bin-api-manager/gens/models/common"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
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
	common.Pagination
} {
	nextToken := ""
	if len(tmps) > 0 {
		nextToken = nextTokenValue
	}

	return struct {
		Result []*T `json:"result"`
		common.Pagination
	}{
		Result: tmps,
		Pagination: common.Pagination{
			NextPageToken: &nextToken,
		},
	}
}

// func CreateResponse[T any](tmps []*T) struct {
// 	Result []*T `json:"result"`
// 	common.Pagination
// } {
// 	nextToken := ""
// 	if len(tmps) > 0 {
// 		// Assuming T has a field TMCreate of type string
// 		nextToken = tmps[len(tmps)-1].TMCreate
// 	}

// 	return struct {
// 		Result []*T `json:"result"`
// 		common.Pagination
// 	}{
// 		Result: tmps,
// 		Pagination: common.Pagination{
// 			NextPageToken: &nextToken,
// 		},
// 	}
// }
