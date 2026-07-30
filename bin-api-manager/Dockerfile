# base
FROM public.ecr.aws/docker/library/golang:1.25-bookworm AS base

# build
FROM base AS build

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-api-manager && go mod vendor && go build -o /app/bin/ ./cmd/...

# run
FROM base

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
COPY --from=build /app/bin-api-manager/docsdev /app/bin/docsdev
COPY --from=build /app/bin-api-manager/gens/openapi_redoc /app/bin/gens/openapi_redoc
