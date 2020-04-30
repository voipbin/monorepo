FROM golang:1.14.2-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY . .
RUN pwd
RUN ls -l /app

RUN go mod download
RUN go build ./cmd/...
