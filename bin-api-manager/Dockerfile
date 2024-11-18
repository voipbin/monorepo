# build
FROM golang:1.23-bookworm AS build

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN apt update && apt install -y build-essential pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
RUN cd bin-api-manager && go build -o /app/bin/ ./cmd/...

# run
FROM golang:1.23-bookworm

WORKDIR /app/bin/
RUN apt update && apt install -y build-essential pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
COPY --from=build /app/bin /app/bin
