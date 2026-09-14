Operations and troubleshooting
==============================

Install doctor
------------------

``./scripts/doctor.sh`` (or ``sudo ./voipbin doctor``) is the read-only
diagnostic superset of ``check-install.sh``. It runs at any stage
(pre-install, mid-install, running stack), never mutates anything, and
auto-skips whatever cannot be probed at the current stage.

Output is one line per check, ``DOCTOR <name>: pass|fail|warn|skip
<detail>``; every failure (and every fixable warning) is immediately
followed by a ``FIX <name>: <exact command>`` line:

.. code-block:: bash

    ./scripts/doctor.sh
    ./scripts/doctor.sh | grep '^FIX '   # extract every recovery command

Exit codes: ``0`` only when nothing failed (warnings and skips do not
fail the run); ``1`` when any check failed; ``2`` when the doctor cannot
even start.

Day-to-day commands
-------------------

.. code-block:: bash

    # Status and logs
    sudo ./voipbin status
    sudo ./voipbin logs -f api-manager

    # Restart a service (Asterisk call/registrar/conference restarts
    # its paired proxy sidecar automatically)
    sudo ./voipbin restart api-manager

    # Debug shells
    sudo ./voipbin ast     # Asterisk CLI
    sudo ./voipbin kam     # Kamailio kamcmd
    sudo ./voipbin db      # MySQL

    # Backup and restore
    sudo ./voipbin backup
    sudo ./voipbin restore <timestamp> --force

    # Tear everything down (irreversible, including the database)
    sudo ./voipbin clean --all

Common issues
-------------

Extensions register but calls fail (AMI/ARI mismatch)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``AMI_USERNAME``/``AMI_PASSWORD`` must stay ``asterisk``/``asterisk`` to
match the static account baked into the Asterisk images' AMI and ARI
configuration. If these were ever changed, SIP registration keeps working
(that path is pjsip plus realtime MySQL) while call control silently
breaks. Fix by restoring the fixed values in ``.env`` and recreating the
three proxy sidecars:

.. code-block:: bash

    docker compose up -d --force-recreate \
      asterisk-call-proxy asterisk-conference-proxy asterisk-registrar-proxy

Extensions created with the wrong domain
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

If extensions register under ``.voipbin.test`` instead of
``.registrar.voipbin.test``, a shell-exported ``DOMAIN_NAME_EXTENSION``
is overriding ``.env``:

.. code-block:: bash

    env | grep DOMAIN_NAME
    unset DOMAIN_NAME_EXTENSION DOMAIN_NAME_TRUNK
    docker compose config | grep DOMAIN
    docker compose rm -fsv registrar-manager && docker compose up -d registrar-manager

Then delete and recreate the affected extensions via the API.

DNS not resolving ``*.voipbin.test`` (internal mode)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash

    docker ps | grep voipbin-dns          # CoreDNS container running?
    dig @127.0.0.1 voipbin.test           # CoreDNS answering directly?
    dig voipbin.test                      # system resolver picking it up?
    cat /etc/resolv.conf                  # 127.0.0.1 listed first?
    sudo ./scripts/setup-dns.sh           # re-run DNS setup

A downed CoreDNS container does not take down all host DNS: a captured
upstream nameserver (from the pre-install resolver, or a hardcoded public
fallback) is written into ``/etc/resolv.conf`` alongside ``127.0.0.1``.

Host IP changed after reboot
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The installer detects a changed host IP on ``start.sh`` and regenerates
``.env``, the CoreDNS Corefile, and TLS certificates automatically. Force
it manually if needed:

.. code-block:: bash

    sudo ./scripts/setup-dns.sh --regenerate

If a browser then shows ``ERR_CERT_AUTHORITY_INVALID``, the certificate
was regenerated correctly; do a hard refresh or use a non-incognito
window (incognito does not trust user-installed CAs).

Reset everything
-----------------

.. code-block:: bash

    sudo ./scripts/clean.sh --volumes --purge

Always use the combined ``--volumes --purge`` form. ``--purge`` alone
keeps the database volume with the old domain's data, which is the worst
partial state to be in.

Known limitations
------------------

Single-public-IP NAT is not supported
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

External mode targets directly-routable hosts (on-prem, corporate LAN
with internal DNS, cloud environments with multiple routable IPs).
Kamailio runs with host networking and binds its dedicated address
directly, so there is no port mapping to remap it behind a single public
IP shared with other services. See :doc:`self_hosting_overview` for the
internal-mode versus external-mode comparison; internal mode has no
domain, certificate, or routability requirements and works behind any
NAT.

Restart the Asterisk service and its proxy sidecar together
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``asterisk-call``, ``asterisk-conference``, and ``asterisk-registrar``
each pair with a dedicated ``-proxy`` sidecar that bridges AMI/ARI to
the rest of the stack. Recreating one side without the other leaves the
AMI/ARI bridge orphaned: SIP registration can keep working (it goes
through pjsip and realtime MySQL, not the proxy) while call control
silently breaks, reproducing the same symptom as the AMI/ARI credential
mismatch above. ``sudo ./voipbin restart <service>`` already restarts a
service and its paired proxy together; avoid running
``docker compose restart`` directly on just one side of the pair.

Getting help
------------

If ``doctor.sh`` does not resolve the issue, open an issue at
https://github.com/voipbin/voipbin or contact support@voipbin.net with
the full output of ``./scripts/doctor.sh``.
