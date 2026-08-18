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
# genuine mysqldump 8.0 as a subprocess against the MySQL 8.0 platform
# database, so the runtime image must carry the MySQL client binaries.
# mysql-community-client comes from MySQL's official APT repository —
# alpine/debian "mysql-client"/"default-mysql-client" are MariaDB aliases,
# an unsupported pairing against MySQL 8.0 (caching_sha2_password auth,
# incompatible dump output).
FROM public.ecr.aws/docker/library/debian:bookworm-slim

# Key integrity: fetch MySQL's signing key over HTTPS from repo.mysql.com
# itself (not an HKP keyserver lookup by short key id, which is spoofable),
# verify its fingerprint before trusting it, and use the HTTPS repo URL —
# closing the MITM window on both the key and the packages.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl gnupg \
    && export GNUPGHOME="$(mktemp -d)" \
    && curl -fsSL https://repo.mysql.com/RPM-GPG-KEY-mysql-2025 -o /tmp/mysql-key.asc \
    && gpg --import /tmp/mysql-key.asc \
    && gpg --fingerprint --with-colons | grep -q "^fpr:::::::::BCA43417C3B485DD128EC6D4B7B3B788A8D3785C:$" \
    && gpg --export B7B3B788A8D3785C > /usr/share/keyrings/mysql.gpg \
    && rm -rf "$GNUPGHOME" /tmp/mysql-key.asc \
    && echo "deb [signed-by=/usr/share/keyrings/mysql.gpg] https://repo.mysql.com/apt/debian/ bookworm mysql-8.0" > /etc/apt/sources.list.d/mysql.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends mysql-community-client \
    && apt-get purge -y --auto-remove gnupg \
    && rm -rf /var/lib/apt/lists/*

# curl is kept (not purged, unlike gnupg above): debian:bookworm-slim ships
# with neither wget nor curl, and this image needs an HTTP client for its
# own Docker/Komodo healthcheck (GET /metrics) — see komodo/docker-compose.yml
# and install/docker-compose.yml.dist. It was already being fetched for the
# MySQL GPG key step above, so reusing it here avoids adding a new package.

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
