# base
FROM golang:1.23-bookworm AS base

RUN apt-get update -qq && apt-get install -y --no-install-recommends \
    apt-utils \
    build-essential \
    pkg-config \
    libzmq5 \
    libzmq3-dev \
    libczmq4 \
    libczmq-dev \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# build
FROM base AS build

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-api-manager && go build -o /app/bin/ ./cmd/...

# run
FROM base

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
COPY --from=build /app/bin-api-manager/docsdev /app/bin/docsdev
