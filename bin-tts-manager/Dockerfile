FROM golang:1.22-bookworm

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

RUN apt update && apt install python3

WORKDIR /app
COPY . .

RUN cd bin-tts-manager && go build ./cmd/...
WORKDIR /app/bin-tts-manager
