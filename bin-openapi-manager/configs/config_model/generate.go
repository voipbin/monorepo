package generate

//go:generate sh -c "mkdir -p ../../gens/models && go run -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.generate.yaml -o ../../gens/models/gen.go ../../openapi/openapi.yaml"
