FROM golang:1.22-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY . .

RUN cd bin-call-manager && go build ./cmd/...
WORKDIR /app/bin-call-manager
