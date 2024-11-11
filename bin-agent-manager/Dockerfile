# build
FROM golang:1.22-alpine AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-agent-manager && go build -o /app/bin/ ./cmd/...

# run
FROM alpine

RUN apk add --no-cache libc6-compat
COPY --from=build /app/bin /app/bin

WORKDIR /app/bin/

