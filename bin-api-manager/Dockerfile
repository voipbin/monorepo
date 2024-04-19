FROM golang:1.22-bookworm

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY . .

RUN apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
RUN cd bin-api-manager && go build ./cmd/...

WORKDIR /app/bin-api-manager
