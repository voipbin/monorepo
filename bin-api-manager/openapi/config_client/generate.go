package generate

//go:generate sh -c "mkdir -p ../../gens/openapi_client && go run -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.generate.yaml -o ../../gens/openapi_client/gen.go ../openapi.yaml"
