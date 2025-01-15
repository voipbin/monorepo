package generate

//go:generate sh -c "mkdir -p ../../../gens/models/chat_manager && go run -mod=mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.generate.yaml -o ../../../gens/models/chat_manager/gen.go ./main.yaml"
