Prerequisites
=============

Local tools
-----------

Install these on the server that will run the stack. The preflight step
in ``init.sh`` verifies Docker is present.

================  ================  =================================================
Tool              Min. version      Notes
================  ================  =================================================
Docker Engine     n/a               Required.
Docker Compose    2.24.4+           v2 CLI plugin. The test override file uses
                                    ``!reset``/``!override`` merge tags that need
                                    this version or later.
mkcert            n/a               Recommended (internal mode). Falls back to
                                    self-signed certificates if absent.
================  ================  =================================================

No host Python, alembic, or MySQL client is required: database migrations
run inside a container (``scripts/migrate.sh``, ``python:3.11-slim`` on
the Compose network).

Installing prerequisites
-------------------------

Ubuntu/Debian:

.. code-block:: bash

    sudo apt update && sudo apt install -y docker.io docker-compose-v2
    sudo usermod -aG docker $USER && newgrp docker

    sudo apt install -y mkcert
    mkcert -install

macOS:

.. code-block:: bash

    brew install --cask docker
    brew install mkcert
    mkcert -install

``mkcert -install`` adds a local Certificate Authority to your system
trust store, so browsers trust the certificates the installer generates
for ``*.voipbin.test`` without a warning.

System requirements
--------------------

- **OS**: Linux (Ubuntu/Debian tested) or macOS.
- **Disk space**: ``doctor.sh`` enforces a hard minimum of 3 GiB free and
  warns under 15 GiB. Budget more over time for call recordings and
  database growth.
- **CPU/RAM**: not automatically checked. The stack is roughly 50
  containers: 32 backend Go microservices, the SIP/media stack (Kamailio,
  RTPEngine, 3x Asterisk plus their AMI/ARI proxy sidecars), supporting
  infrastructure (MySQL, Redis, RabbitMQ, PostgreSQL, ClickHouse, CoreDNS),
  and 3 frontend apps. A laptop-class multi-core machine with a few GB of
  RAM headroom works for development; a single-vCPU, 1 GB VM will not keep
  up.
- **Networking (external mode only)**: a directly-routable host with
  distinct IPs for the host, Kamailio, and RTPEngine. See the Install
  section's External mode instructions below.

Credentials generated at install time
---------------------------------------

``init.sh`` generates fresh, random credentials for this install; nothing
is shipped as a shared default.

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Service
     - Credentials
   * - MySQL
     - Randomly generated per install, in ``.env`` (``MYSQL_ROOT_PASSWORD``)
   * - RabbitMQ
     - Randomly generated per install, in ``.env``
       (``RABBITMQ_DEFAULT_USER`` / ``RABBITMQ_DEFAULT_PASS``)
   * - JWT signing key
     - Auto-generated in ``.env``
   * - Admin account and extensions (opt-in)
     - ``admin@localhost`` / ``admin@localhost``, extensions ``1000``,
       ``2000``, ``3000``. Only created when ``VOIPBIN_SANDBOX_DEV_SEED=true``
       is set. Never set this on an install reachable from the public
       internet.

Before exposing an install beyond localhost, review the TLS mode, the
firewall and network exposure of the ports this stack opens, and confirm
``VOIPBIN_SANDBOX_DEV_SEED`` is unset or ``false``.
