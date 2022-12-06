FROM golang:1.18-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY go.mod go.sum ./
COPY . .

RUN go build ./cmd/...
