# build
FROM golang:1.24-alpine AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-sentinel-manager && go build -o /app/bin/ ./cmd/...

# run
FROM alpine

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
