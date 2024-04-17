FROM golang:1.22-bookworm

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app

COPY go.mod go.sum ./
COPY . .

# set private reposiroty config
RUN git config --global url."https://GL_DEPLOY_USER:GL_DEPLOY_TOKEN@gitlab.com".insteadOf "https://gitlab.com"
RUN export GOPRIVATE="gitlab.com/voipbin"

RUN apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
RUN go mod vendor
RUN go build ./cmd/...
