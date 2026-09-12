Environment variables and maintenance
========================================

Key ``.env`` variables
-------------------------

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Variable
     - Purpose
   * - ``DOMAIN_MODE``
     - ``internal`` (default) or ``external``. A missing key means internal.
   * - ``BASE_DOMAIN``
     - The base domain all derived domain values are composed from.
       Edit and re-run ``init`` rather than editing derived vars directly.
   * - ``TLS_MODE``
     - ``mkcert``, ``selfsigned``, or ``byo``.
   * - ``COMPOSE_PROFILES``
     - ``internal-dns`` in internal mode (enables CoreDNS), empty in
       external mode, ``web-proxy`` when the reverse proxy is enabled.
   * - ``WEB_REVERSE_PROXY``
     - ``true`` enables the built-in Caddy reverse proxy (external mode,
       ``--tls byo`` only).
   * - ``HOST_EXTERNAL_IP``
     - Host's LAN or public IP (auto-detected).
   * - ``KAMAILIO_EXTERNAL_IP`` / ``RTPENGINE_EXTERNAL_IP``
     - Dedicated external IPs for SIP signaling and RTP media
       (auto-generated; must differ from the host IP).
   * - ``DOMAIN_NAME_EXTENSION``
     - SIP domain suffix for extensions. Full realm is
       ``{customer_id}.{DOMAIN_NAME_EXTENSION}``.
   * - ``VOIPBIN_SANDBOX_DEV_SEED``
     - ``true`` to opt in to dev/test account seeding on ``start``. Off
       by default; never enable on a public install.

Third-party integrations
---------------------------

Voice AI, transcription, TTS, phone number provisioning, and email/SMS
providers are configured through their own keys in ``.env``
(for example ``OPENAI_API_KEY``, ``TWILIO_SID``, ``SENDGRID_API_KEY``,
``AWS_ACCESS_KEY``). Until the relevant keys are set, the corresponding
flow actions return a "provider not configured" error; the rest of the
platform stays healthy.

Maintenance commands (the ``voipbin`` CLI)
---------------------------------------------

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Command
     - Description
   * - ``version [--json]``
     - Show pinned image versions
   * - ``update [images/scripts/all]``
     - Update Docker images or scripts. ``update all`` on a pinned repo
       runs the full safe upgrade: backup, git pull, migrate, recreate,
       verify.
   * - ``update --check``
     - Dry-run to preview updates
   * - ``backup``
     - Full data backup (MySQL, call recordings, ``.env``, certificates,
       ``versions.lock``, a ``manifest.json``) into ``backups/<timestamp>/``
   * - ``restore <timestamp> --force``
     - Restore data from a backup. Destructive; services must be stopped
       except ``db``/``redis``.
   * - ``rollback [timestamp]``
     - Roll back image versions from override history (unpinned repos
       only). For data recovery use ``restore``.
   * - ``clean [options]``
     - Cleanup sandbox resources

Scheduled backups
~~~~~~~~~~~~~~~~~~~~

``schedule-manager`` runs a nightly ``database-backup`` job (MySQL dump
plus gzip of the two databases, written to ``backups/scheduled-db/``,
retaining the newest 7). ``start.sh`` enables it on every run. This is
deliberately separate from the manual ``voipbin backup`` command above:
different retention, different directory, so neither one's pruning
touches the other.

.. code-block:: bash

    docker exec voipbin-schedule-mgr /app/bin/schedule-control schedule list
    docker exec voipbin-schedule-mgr /app/bin/schedule-control schedule disable database-backup

Host-side gaps (operator responsibility)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Two tasks stay outside the installer's automation:

- **Offsite copy of backups.** ``backups/`` is local disk; ship it to
  remote or object storage on your own recovery-objective schedule
  (for example an ``rsync`` cron job).
- **Host-level maintenance.** OS package updates, Docker Engine upgrades,
  disk space and log rotation, and kernel/security patching are the
  operator's responsibility.

Scaling
---------

The installer starts every backend service at one replica. Scaling a
single-server Compose install means raising the host's own CPU/RAM and,
for the SIP/media layer, the RTPEngine port range and Asterisk channel
limits. There is no automated horizontal-scale profile in this installer
today; track resource usage and scale the host vertically first.
