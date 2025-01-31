package generate

//go:generate sh -c "mkdir -p ../../gens/openapi_redoc &&  command -v npx >/dev/null 2>&1 && npx @redocly/cli build-docs ../../../bin-openapi-manager/openapi/openapi.yaml --output ../../gens/openapi_redoc/api.html || echo 'npx is not installed, skipping generation'"
