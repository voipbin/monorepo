# --- Stage 1: Migration and SQL Export ---
FROM python:3.11-slim AS builder

ENV MYSQL_ROOT_PASSWORD=root_password
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    mariadb-server \
    default-libmysqlclient-dev \
    build-essential \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY bin-dbscheme-manager/* .

RUN pip install --no-cache-dir "sqlalchemy<2.0" alembic mysqlclient

RUN set -e; \
    mysql_install_db --user=mysql --datadir=/var/lib/mysql; \
    mysqld_safe --user=mysql --skip-networking & \
    until mysqladmin ping -h localhost --silent; do sleep 1; done; \
    \
    mysql -u root -e "CREATE DATABASE IF NOT EXISTS bin_manager;"; \
    mysql -u root -e "CREATE DATABASE IF NOT EXISTS asterisk;"; \
    \
    cp bin-manager/alembic.ini.sample bin-manager/alembic.ini; \
    sed -i 's|^sqlalchemy.url.*|sqlalchemy.url = mysql://root@localhost/bin_manager|' bin-manager/alembic.ini; \
    cp asterisk_config/alembic.ini.sample asterisk_config/alembic.ini; \
    sed -i 's|^sqlalchemy.url.*|sqlalchemy.url = mysql://root@localhost/asterisk|' asterisk_config/alembic.ini; \
    \
    echo "Running migrations..."; \
    cd /app/bin-manager && PYTHONPATH=/app/bin-manager alembic upgrade head; \
    cd /app/asterisk_config && PYTHONPATH=/app/asterisk_config alembic upgrade head; \
    \
    echo "Exporting database dumps with collation patch..."; \
    # Patching MariaDB specific collation to universal utf8mb4_general_ci for MySQL compatibility
    mysqldump -u root --databases bin_manager | sed 's/utf8mb4_uca1400_ai_ci/utf8mb4_general_ci/g' > /app/bin_manager.sql; \
    mysqldump -u root --databases asterisk | sed 's/utf8mb4_uca1400_ai_ci/utf8mb4_general_ci/g' > /app/asterisk.sql; \
    \
    mysqladmin -u root shutdown

# --- Stage 2: Final MySQL 8.0 Image ---
FROM mysql:8.0

ENV MYSQL_ROOT_PASSWORD=root_password

COPY --from=builder /app/bin_manager.sql /docker-entrypoint-initdb.d/01_bin_manager.sql
COPY --from=builder /app/asterisk.sql /docker-entrypoint-initdb.d/02_asterisk.sql

EXPOSE 3306