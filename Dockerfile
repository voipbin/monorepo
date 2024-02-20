FROM golang:1.20-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY . .
RUN pwd
RUN ls -l /app

RUN go build ./cmd/...
