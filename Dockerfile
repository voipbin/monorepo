FROM golang:1.18-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY . .
RUN pwd
RUN ls -l /app

RUN go build ./cmd/...
