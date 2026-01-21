# build
FROM public.ecr.aws/docker/library/golang:1.25-alpine AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

# Install CGO dependencies for libsoxr (audio resampling)
RUN apk add --no-cache gcc musl-dev pkgconfig libsoxr-dev

WORKDIR /app
COPY ./ .

RUN mkdir -p /app/bin
RUN cd bin-pipecat-manager && CGO_ENABLED=1 go build -o /app/bin/ ./cmd/...

RUN mkdir -p /app/scripts
RUN cp -r bin-pipecat-manager/scripts/pipecat /app/scripts/pipecat

# run
FROM python:3.12-slim

WORKDIR /app/bin/

COPY --from=build /app/bin /app/bin
COPY --from=build /app/scripts/pipecat /app/scripts/pipecat

COPY --from=build /app/scripts/pipecat/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt && rm /tmp/requirements.txt

# Install runtime dependencies: ffmpeg for media processing, libsoxr for audio resampling
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg libsoxr0 && rm -rf /var/lib/apt/lists/*

ENV PATH="/app/bin:${PATH}"
