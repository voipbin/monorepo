FROM golang:1.18.4-bullseye

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY go.mod go.sum ./
COPY . .

RUN apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
RUN go build ./cmd/...
