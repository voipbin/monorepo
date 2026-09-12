Install
=======

The recommended path is the four-command flow, then day-to-day operation
through the ``voipbin`` CLI.

.. code-block:: bash

    git clone https://github.com/voipbin/voipbin.git
    cd voipbin/install

    ./scripts/init.sh --yes          # 1. Generate .env, certificates, docker-compose.yml
    sudo ./scripts/setup-host.sh     # 2. The single sudo command (host mutations)
    ./scripts/start.sh               # 3. Start all services
    ./scripts/check-install.sh       # 4. Self-verify the install

``init.sh`` (generate configuration)
-------------------------------------

Generates ``.env``, TLS certificates, and copies ``docker-compose.yml``
from the committed ``docker-compose.yml.dist``. After this first copy,
``docker-compose.yml`` is untracked and operator-owned: a later
``git pull`` in this repo updates ``docker-compose.yml.dist`` but never
touches your live file. The same split applies to ``versions.lock`` /
``versions.lock.dist``.

.. code-block:: bash

    ./scripts/init.sh --yes

    # external mode (real domain, bring-your-own certificate)
    ./scripts/init.sh --mode external --domain example.com --tls byo \
      --cert fullchain.pem --key privkey.pem --yes

``setup-host.sh`` (the single sudo command)
----------------------------------------------

Owns every host mutation: mkcert package and CA trust (internal mode
only), CoreDNS setup (internal mode only), the Compose default Docker
network, and the VoIP network interfaces. Idempotent; safe to rerun.

.. code-block:: bash

    sudo ./scripts/setup-host.sh

``start.sh`` (bring the stack up)
------------------------------------

Starts infrastructure, runs database migrations inside a container, and
starts all services.

.. code-block:: bash

    ./scripts/start.sh

``check-install.sh`` (self-verify)
--------------------------------------

Verifies service counts, DNS resolution, API liveness, and (external
mode) the TLS chain strictly, without skipping validation.

.. code-block:: bash

    ./scripts/check-install.sh

If anything fails at any of the four steps, run ``./scripts/doctor.sh``: a
read-only diagnostic that works at any stage and prints the exact recovery
command for every failure. See the Troubleshooting section below.

Once installed, ``sudo ./voipbin`` is the interactive CLI for day-to-day
operations.

.. code-block:: bash

    sudo ./voipbin

    voipbin> status
    voipbin> logs -f api-manager

Or run single commands directly: ``sudo ./voipbin status``.

Command categories:

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Category
     - Examples
   * - Service control
     - ``start``, ``stop``, ``restart``, ``status``/``ps``, ``logs``
   * - Debug shells
     - ``ast`` (Asterisk CLI), ``kam`` (Kamailio kamcmd), ``db`` (MySQL),
       ``api`` (authenticated REST client)
   * - Extension management
     - ``ext list``/``create``/``delete``
   * - Infrastructure
     - ``dns status``/``test``/``regenerate``, ``network status``/``setup``,
       ``certs status``/``trust``
   * - Resource management
     - ``customer``, ``agent``, ``billing``, ``number``, ``registrar``,
       ``call``, ``conference``, ``flow``, ``campaign``, ``queue``, and more
   * - Maintenance
     - ``version``, ``update``, ``backup``, ``restore``, ``rollback``,
       ``clean``, ``config``

See the Environment variables and maintenance section below for the full
maintenance command reference (backup, restore, upgrade).

External mode (real domain)
------------------------------

External mode runs the install under a real domain with a real
certificate. The installer never touches DNS or the trust store in this
mode; you own both.

Step 1: create DNS records
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. list-table::
   :header-rows: 1
   :widths: 30 15 25 30

   * - Record
     - Type
     - Target
     - Purpose
   * - ``api.<domain>``
     - A
     - Host IP
     - REST API + WebSocket (:8443)
   * - ``admin.<domain>`` / ``meet.<domain>`` / ``talk.<domain>``
     - A
     - Host IP
     - Web UIs (:3003/:3004/:3005)
   * - ``sip.<domain>``
     - A
     - Kamailio external IP
     - SIP signaling / WSS (:5060/:5066)
   * - ``sip-service.<domain>``, ``conference.<domain>``,
       ``trunk.<domain>``, ``pstn.<domain>``
     - A
     - Kamailio external IP
     - SIP surfaces
   * - ``registrar.<domain>``
     - A
     - Kamailio external IP
     - Apex registrar name, not covered by the wildcard below
   * - ``*.registrar.<domain>``
     - A
     - Kamailio external IP
     - Per-customer SIP realm resolution

Host IP and Kamailio IP are two distinct addresses on the same subnet.

Step 2: obtain a certificate
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The certificate must cover ``api.``, ``sip.``, ``sip-service.``,
``conference.``, ``trunk.``, and ``registrar.`` of your domain; a wildcard
covers all six. A wildcard requires the DNS-01 challenge:

.. code-block:: bash

    certbot certonly --preferred-challenges dns --manual \
      -d example.com -d '*.example.com' -d '*.registrar.example.com'

Step 3: initialize
~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash

    ./scripts/init.sh --mode external --domain example.com --tls byo \
      --cert /etc/letsencrypt/live/example.com/fullchain.pem \
      --key  /etc/letsencrypt/live/example.com/privkey.pem \
      --yes

The certificate is validated (key match, SAN coverage, expiry) before
``.env`` is written; a bad certificate aborts cleanly.

Step 4: host setup, start, and verify
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash

    sudo ./scripts/setup-host.sh
    ./scripts/start.sh
    ./scripts/check-install.sh

Certificate renewal
~~~~~~~~~~~~~~~~~~~~~

``install-certs.sh`` is idempotent and usable as a certbot deploy hook:

.. code-block:: bash

    certbot renew --deploy-hook \
      '/path/to/install/scripts/install-certs.sh /etc/letsencrypt/live/example.com/fullchain.pem /etc/letsencrypt/live/example.com/privkey.pem'

.. warning::

   By default, the ``admin``/``meet``/``talk`` web UIs are plain HTTP on
   ports 3003 to 3005 in both modes. On a routable domain this means
   credentials travel in the clear. Front them with a TLS-terminating
   reverse proxy, restrict those ports to trusted networks, or use the
   built-in ``--web-reverse-proxy`` flag described next.

Web reverse proxy (port-less URLs)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``init.sh --web-reverse-proxy`` (external mode with ``--tls byo`` only)
runs a Caddy container that terminates TLS with your certificate and
routes ``api``/``admin``/``meet``/``talk.<domain>`` by Host header, so
``https://admin.example.com`` works with no port suffix:

.. code-block:: bash

    ./scripts/init.sh --mode external --domain example.com --tls byo \
      --cert fullchain.pem --key privkey.pem \
      --web-reverse-proxy --yes
    sudo ./scripts/setup-host.sh
    ./scripts/start.sh
    ./scripts/check-install.sh

The certificate must additionally cover ``admin``, ``meet``, and ``talk``
(a wildcard already does). Once enabled, ``admin``/``meet``/``talk``'s
published ports become loopback-only; Caddy's 80/443 is the only
externally-reachable path to them.
