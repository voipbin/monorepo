FROM golang:1.20-bookworm

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

RUN apt update && apt install python3

WORKDIR /app

COPY go.mod go.sum ./
COPY . .

RUN go build ./cmd/...
