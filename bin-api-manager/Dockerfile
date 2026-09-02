# build
FROM public.ecr.aws/docker/library/golang:1.27.1-alpine AS build

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-api-manager && go mod vendor && CGO_ENABLED=0 go build -o /app/bin/ ./cmd/...

# run
FROM gcr.io/distroless/static-debian12

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
COPY --from=build /app/bin-api-manager/docsdev/build/html /app/bin/docsdev/build/html
COPY --from=build /app/bin-api-manager/gens/openapi_redoc /app/bin/gens/openapi_redoc
