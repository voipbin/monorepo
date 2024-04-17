FROM golang:1.22-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
RUN go version

COPY ../ .

RUN ls -tlr && ls -tlr ../ && cd bin-agent-manager && go build ./cmd/...
