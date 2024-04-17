FROM golang:1.22-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
RUN go version

COPY go.mod go.sum ./
COPY . .

RUN go build ./cmd/...
