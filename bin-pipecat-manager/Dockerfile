# build
FROM public.ecr.aws/docker/library/golang:1.25-bookworm AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

# Install CGO dependencies for libsoxr (audio resampling)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    pkg-config \
    libsoxr-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY ./ .

RUN mkdir -p /app/bin
RUN cd bin-pipecat-manager && go mod vendor && CGO_ENABLED=1 go build -o /app/bin/ ./cmd/...

RUN mkdir -p /app/scripts
RUN cp -r bin-pipecat-manager/scripts/pipecat /app/scripts/pipecat

# run
FROM python:3.12-slim

WORKDIR /app/bin/

COPY --from=build /app/bin /app/bin
COPY --from=build /app/scripts/pipecat /app/scripts/pipecat

COPY --from=build /app/scripts/pipecat/requirements.txt /tmp/requirements.txt
# pipecat-ai 1.4.0's silero/local-smart-turn extras pull in torch. The smart-turn
# model runs on CPU (smart-turn-v3.x-cpu.onnx) and this runner does no GPU work,
# so install the CPU-only torch/torchaudio wheels FIRST (pinned for a reproducible
# build). This satisfies the torch requirement before the requirements.txt install
# (pip then skips the default CUDA wheel), cutting the image's site-packages from
# ~5.5GB to ~1.7GB. Bump these in lockstep with the pipecat-ai version.
RUN pip install --no-cache-dir torch==2.12.1 torchaudio==2.11.0 --index-url https://download.pytorch.org/whl/cpu
RUN pip install --no-cache-dir -r /tmp/requirements.txt && rm /tmp/requirements.txt

# Install runtime dependencies: ffmpeg for media processing, libsoxr for audio resampling
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg libsoxr0 && rm -rf /var/lib/apt/lists/*

ENV PATH="/app/bin:${PATH}"
