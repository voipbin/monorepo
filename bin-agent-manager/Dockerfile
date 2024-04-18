FROM golang:1.22-alpine

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
RUN go version

COPY ./ .

RUN ls -tlr
RUN ls -tlr ../
RUN go -C bin-agent-manager build ./cmd/... && mv agent-manager ../