# build
FROM public.ecr.aws/docker/library/golang:1.25-alpine AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-schedule-manager && go mod vendor && CGO_ENABLED=0 go build -o /app/bin/ ./cmd/...

# run
#
# Deviation from the fleet-standard distroless static runtime (design §7,
# documented in this service's CLAUDE.md): the scheduled DB backup runs
# mariadb-dump as a subprocess against the platform database, so the
# runtime image must carry the client binaries. The platform DB moved to
# MariaDB 12.3 in VOIP-1386; this switches the client to match (VOIP-1397,
# following VOIP-1387's interim mysqldump-8.0-with-compatibility-flags
# workaround). default-mysql-client comes straight from Debian bookworm's
# own repository (resolves to mariadb-client) — no external repo or GPG
# key needed, unlike the mysql-community-client this replaces.
FROM public.ecr.aws/docker/library/debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl default-mysql-client \
    && rm -rf /var/lib/apt/lists/*

# curl: debian:bookworm-slim ships with neither wget nor curl, and this
# image needs an HTTP client for its own Docker/Komodo healthcheck
# (GET /metrics) — see komodo/docker-compose.yml and
# install/docker-compose.yml.dist.

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
