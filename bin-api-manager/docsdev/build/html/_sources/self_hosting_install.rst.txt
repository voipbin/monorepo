Install
=======

The happy path is three commands. They are idempotent and resumable.

.. code-block:: bash

    git clone https://github.com/voipbin/install.git
    cd install
    pip install -r requirements.txt

    ./voipbin-install init      # interactive wizard + GCP bootstrap
    ./voipbin-install apply     # provision and deploy
    ./voipbin-install verify    # health check

``init`` (interactive wizard)
-----------------------------

The ``init`` command runs a multi-step bootstrap, in order:

1. Preflight checks for the six local tools.
2. ``gcloud`` user auth and Application Default Credentials.
3. **Eight wizard questions**: GCP project ID, region, GKE cluster type
   (zonal or regional), TLS strategy (``self-signed`` or ``byoc``),
   Docker image tag strategy (latest or pinned via
   ``config/versions.yaml``), base domain name, Kamailio TLS cert mode
   (``self_signed`` or ``manual``), and Cloud DNS mode (auto or manual).
4. Project existence and billing.
5. Quota check against ``config/gcp_quotas.yaml``.
6. Enable sixteen GCP APIs.
7. Create the ``voipbin-installer`` service account with the IAM role
   bindings defined in ``config/gcp_iam_roles.yaml``.
8. Create a KMS key ring and crypto key.
9. Generate six secrets (``jwt_key``, ``cloudsql_password``,
   ``redis_password``, ``rabbitmq_user``, ``rabbitmq_password``,
   ``api_signing_key``) and encrypt them into ``secrets.yaml`` with SOPS
   bound to the KMS key.
10. Write ``.sops.yaml``.
11. Save ``config.yaml``.
12. Print a cost summary.

Wizard questions
~~~~~~~~~~~~~~~~

The eight questions and their accepted values:

==================================  ==========================================
Question                            Options / Example
==================================  ==========================================
GCP project ID                      ``my-voipbin-project``
Region                              ``us-central1`` (or any GCP region)
GKE cluster type                    ``zonal`` (cheaper) or ``regional`` (HA)
TLS strategy                        ``self-signed`` or ``byoc``
Docker image tag strategy           ``latest`` or ``pinned``
Domain name                         ``voipbin.example.com``
Kamailio cert mode                  ``self_signed`` (auto) or ``manual``
Cloud DNS mode                      ``auto`` (GCP manages DNS) or ``manual``
==================================  ==========================================

Useful flags:

.. code-block:: bash

    ./voipbin-install init --reconfigure          # re-run the wizard
    ./voipbin-install init --config path/to.yaml  # import existing config
    ./voipbin-install init --skip-api-enable      # APIs already enabled
    ./voipbin-install init --skip-quota-check     # bypass quota gate
    ./voipbin-install init --dry-run              # show what would happen

The two files written are everything you need to reproduce the install:

- ``config.yaml``: non-sensitive. Safe to commit if you want a record
  per environment.
- ``secrets.yaml``: SOPS-encrypted with GCP KMS. Decrypt locally with
  ``sops --decrypt secrets.yaml``.

.. warning::

   Back up ``secrets.yaml`` and the KMS key ring. If the key ring is
   destroyed or the GCP project is deleted, ``secrets.yaml`` becomes
   unrecoverable.

``apply`` (eight-stage deploy)
------------------------------

``./voipbin-install apply`` executes eight pipeline stages in order. The
pipeline checkpoints between stages, so if a run fails you can fix the
problem and rerun the same command; it picks up from the failed stage.

.. list-table::
   :header-rows: 1
   :widths: 25 50 25

   * - Stage
     - What it does
     - Typical duration
   * - ``terraform_init``
     - Initialize Terraform backend (GCS state bucket)
     - 1 to 2 minutes
   * - ``reconcile_imports``
     - Import drifted GCP resources into Terraform state
     - 1 to 3 minutes
   * - ``terraform_apply``
     - Provision VPC, GKE, Cloud SQL, VMs
     - 12 to 18 minutes
   * - ``reconcile_outputs``
     - Read Terraform outputs into ``config.yaml``
     - under 1 minute
   * - ``k8s_apply``
     - Deploy Kubernetes workloads to GKE
     - 3 to 5 minutes
   * - ``reconcile_k8s_outputs``
     - Read Kubernetes LB IPs into ``config.yaml``
     - under 1 minute
   * - ``cert_provision``
     - Issue Kamailio TLS certificates
     - 1 to 2 minutes
   * - ``ansible_run``
     - Configure Kamailio and RTPEngine VMs
     - 5 to 8 minutes

Useful flags:

.. code-block:: bash

    ./voipbin-install apply --dry-run               # plan, do not change
    ./voipbin-install apply --auto-approve          # skip prompts
    ./voipbin-install apply --stage terraform_init  # only run this stage
    ./voipbin-install apply --stage k8s_apply
    ./voipbin-install apply --stage ansible_run

``verify`` (health check)
-------------------------

The ``verify`` command runs a series of checks against the live
deployment and prints a pass, warn, or fail per check. The current set
of checks covers, in order: GKE cluster status, pod readiness in three
namespaces, service endpoint availability in three namespaces, Kamailio
and RTPEngine VM run state, Cloud SQL instance state, DNS resolution
for ``api.<domain>``, HTTPS reachability of ``https://api.<domain>/health``,
and a TCP socket probe of SIP port ``5060``. Each check prints the
underlying command on failure so you can rerun individual probes
manually.

.. note::

   The SIP probe uses a TCP socket connect. If you need to verify UDP
   reachability for media or signalling, use a separate tool such as
   ``nc -zuv`` and the GCP firewall log stream.

``status`` (deployment overview)
---------------------------------

The ``status`` command shows a summary of the current deployment state:

.. code-block:: bash

    ./voipbin-install status            # human-readable output
    ./voipbin-install status --json     # machine-readable JSON

``cert`` (certificate management)
----------------------------------

The ``cert`` subcommand manages Kamailio TLS certificates:

.. code-block:: bash

    ./voipbin-install cert status       # show per-SAN cert expiry and mode
    ./voipbin-install cert renew        # re-run cert_provision stage
    ./voipbin-install cert renew --force  # force re-issuance even if not expired
    ./voipbin-install cert export-ca --out ca.pem  # export self-signed CA

DNS step
--------

After apply, the installer prints the DNS records you must publish if
you chose ``dns_mode: manual``, or the four nameservers to delegate to
if you chose ``dns_mode: auto``. Required records, with
``example.com`` standing in for your domain:

============================  ======  ========================
Subdomain                     Type    Value
============================  ======  ========================
``api.example.com``           A       External LB IP
``hook.example.com``          A       External LB IP
``admin.example.com``         A       External LB IP
``talk.example.com``          A       External LB IP
``meet.example.com``          A       External LB IP
``sip.example.com``           A       Kamailio external LB IP
============================  ======  ========================

.. note::

   The ``hook`` subdomain is commonly missed. It is required for
   webhook ingress (HTTP and HTTPS callbacks from external providers).

DNS propagation can take up to 48 hours for NS delegation changes. Once
records resolve, rerun ``./voipbin-install verify``.
