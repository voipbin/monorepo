FROM golang:1.14.1-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download
RUN go build ./...

CMD ["./call-manager"]