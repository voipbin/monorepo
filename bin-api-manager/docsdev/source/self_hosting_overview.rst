Overview
========

The installer is a Docker Compose stack driven by four shell scripts and,
after install, the ``voipbin`` interactive CLI. It runs the complete
VoIPBin platform, roughly 50 containers, on a single server.

What gets deployed
-------------------

- **32 of VoIPBin's 33 backend Go microservices** (``bin-sentinel-manager``
  needs a Kubernetes API and stays out of this Compose-based install).
- **SIP/media stack**: Kamailio (SIP proxy), RTPEngine (RTP media relay),
  three Asterisk instances (call, registrar, conference) each with an
  AMI/ARI proxy sidecar.
- **Supporting infrastructure**: MySQL, Redis, RabbitMQ, PostgreSQL
  (pgvector, for ``rag-manager``), ClickHouse (``timeline-manager``),
  CoreDNS (internal mode only).
- **Three frontend web applications**: Admin Console, Talk (agent
  messenger), Meet (voice conferencing).

Install modes
-------------

The installer runs in one of two modes, chosen at ``init`` time and
recorded in ``.env`` as ``DOMAIN_MODE``. Mode is an init-time decision:
extension SIP realms embed the base domain in the database, so ``init.sh``
refuses to switch mode or domain on an existing install.

.. list-table::
   :header-rows: 1
   :widths: 20 40 40

   * - 
     - Internal mode (default)
     - External mode
   * - Base domain
     - ``voipbin.test`` (IANA reserved TLD, RFC 2606)
     - Your real domain (for example ``example.com``)
   * - DNS
     - Automatic. CoreDNS container plus ``/etc/resolv.conf`` forwarding
     - Operator-managed A records at your DNS provider
   * - TLS
     - Automatic. mkcert (browser-trusted) or self-signed
     - Bring your own certificate
   * - Reachable from
     - This machine and your LAN
     - Any host that can route to your IPs
   * - Best for
     - Local development, demos, evaluation
     - A production or shared install under a real domain

Use internal mode unless you specifically need the install reachable under
a real domain from machines you do not control. Internal mode requires
nothing from you (no domain, no certificate, no DNS provider). External
mode targets directly-routable hosts (on-prem, corporate LAN with internal
DNS, cloud environments with multiple routable IPs); single-public-IP NAT
environments are a documented limitation.

The four-command install
-------------------------

The recommended path isolates the one step that needs root:

.. code-block:: bash

    git clone https://github.com/voipbin/voipbin.git
    cd voipbin/install

    ./scripts/init.sh --yes          # 1. Generate .env, certificates, docker-compose.yml
    sudo ./scripts/setup-host.sh     # 2. The single sudo command (host mutations)
    ./scripts/start.sh               # 3. Start all services
    ./scripts/check-install.sh       # 4. Self-verify the install

``setup-host.sh`` owns every host mutation: mkcert package and CA trust
(internal mode only), CoreDNS setup (internal mode only), the Compose
default Docker network, and the VoIP network interfaces (internal veth
pairs, plus external macvlan for pinned hosting-provider IPs). It is
idempotent; each step probes current state and skips what is already done.

Once installed, ``sudo ./voipbin`` is the interactive CLI for day-to-day
operations: status, logs, restart, debug shells, backup/restore, and
version upgrades. See the Install section for the full command reference.
