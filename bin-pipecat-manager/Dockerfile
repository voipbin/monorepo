# build
FROM public.ecr.aws/docker/library/golang:1.25-alpine AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY ./ .

RUN mkdir -p /app/bin
RUN cd bin-pipecat-manager && go build -o /app/bin/ ./cmd/...

RUN mkdir -p /app/scripts
RUN cp -r bin-pipecat-manager/scripts/pipecat /app/scripts/pipecat

# run
FROM python:3.12-slim

WORKDIR /app/bin/

COPY --from=build /app/bin /app/bin
COPY --from=build /app/scripts/pipecat /app/scripts/pipecat

COPY --from=build /app/scripts/pipecat/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt && rm /tmp/requirements.txt

RUN apt-get update && apt-get install -y ffmpeg

ENV PATH="/app/bin:${PATH}"
