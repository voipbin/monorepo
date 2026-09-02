# build
FROM public.ecr.aws/docker/library/golang:1.27.1-alpine AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-conference-manager && go mod vendor && CGO_ENABLED=0 go build -o /app/bin/ ./cmd/...

# run
FROM gcr.io/distroless/static-debian12

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
