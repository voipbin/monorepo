FROM golang:1.22-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY . .

RUN cd voip-asterisk-proxy && go build ./cmd/...
WORKDIR /app/voip-asterisk-proxy
