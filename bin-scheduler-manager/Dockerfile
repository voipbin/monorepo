# build
FROM public.ecr.aws/docker/library/golang:1.25-alpine AS build

LABEL maintainer="Sungtae Kim <pchero21@gmail.com>"

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-scheduler-manager && go mod vendor && CGO_ENABLED=0 go build -o /app/bin/ ./cmd/...

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

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates gnupg \
    && export GNUPGHOME="$(mktemp -d)" \
    && gpg --keyserver keyserver.ubuntu.com --recv-keys B7B3B788A8D3785C \
    && gpg --export B7B3B788A8D3785C > /usr/share/keyrings/mysql.gpg \
    && rm -rf "$GNUPGHOME" \
    && echo "deb [signed-by=/usr/share/keyrings/mysql.gpg] http://repo.mysql.com/apt/debian/ bookworm mysql-8.0" > /etc/apt/sources.list.d/mysql.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends mysql-community-client \
    && apt-get purge -y --auto-remove gnupg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
