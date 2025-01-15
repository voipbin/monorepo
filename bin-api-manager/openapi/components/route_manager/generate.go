package generate

//go:generate sh -c "mkdir -p ../../../gens/models/route_manager && go run -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.generate.yaml -o ../../../gens/models/route_manager/gen.go ./main.yaml"
