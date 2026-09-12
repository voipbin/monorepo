Configuration files
===================

The installer writes and manages two operator-owned files in the
``install/`` working directory.

``docker-compose.yml`` (operator-owned copy of ``.dist``)
------------------------------------------------------------

``docker-compose.yml`` is copied once from the committed
``docker-compose.yml.dist`` the first time ``init.sh`` runs, then left
alone: a later ``git pull`` in this repo updates ``docker-compose.yml.dist``
but never touches your live ``docker-compose.yml``. This means a repo
update never silently changes a running install's Compose configuration.

To adopt upstream changes (new services, image digest bumps), diff the two
files and merge deliberately, or run ``scripts/sync-compose-images.sh``
with ``COMPOSE_FILE=docker-compose.yml`` to pull in just the image digest
updates from ``versions.lock.dist``.

``versions.lock`` gets the identical treatment for the identical reason:
copied once from ``versions.lock.dist``, then untracked. Deploying a new
image and updating ``versions.lock.dist`` are deliberately decoupled; the
live ``versions.lock`` is yours to manage on your own schedule.

``.env`` (generated secrets and configuration)
--------------------------------------------------

``init.sh`` auto-generates ``.env`` with detected network settings and
freshly generated credentials (MySQL, RabbitMQ, JWT signing key). See the
Environment variables section below for the full variable reference.

CLI configuration (``~/.voipbin-cli.conf``)
-----------------------------------------------

The ``voipbin`` CLI stores its own settings separately from ``.env``:

.. code-block:: bash

    voipbin> config                    # Show all settings
    voipbin> config log_lines 100      # Set log lines to 100
    voipbin> config reset              # Reset to defaults

.. list-table::
   :header-rows: 1
   :widths: 30 20 50

   * - Setting
     - Default
     - Description
   * - ``api_host``
     - localhost
     - API hostname
   * - ``api_port``
     - 8443
     - API port
   * - ``log_lines``
     - 50
     - Number of log lines to display
   * - ``colors``
     - True
     - Enable colored output
   * - ``asterisk_container``
     - voipbin-ast-call
     - Default Asterisk container for the ``ast`` debug shell
