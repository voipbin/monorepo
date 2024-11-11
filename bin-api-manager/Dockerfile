FROM golang:1.22-bookworm AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY . .

RUN apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
RUN cd bin-api-manager && go build -o /app/bin/ ./cmd/...


# run
FROM alpine

RUN apk add --no-cache libc6-compat zeromq czmq
COPY --from=build /app/bin /app/bin

WORKDIR /app/bin/
