# build
FROM public.ecr.aws/docker/library/golang:1.25-alpine AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-flow-manager && go build -o /app/bin/ ./cmd/...

# run
FROM alpine

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
