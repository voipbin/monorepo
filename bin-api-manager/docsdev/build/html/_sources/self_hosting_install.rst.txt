Install
=======

The happy path is three commands. They are idempotent and resumable.

.. code-block:: bash

    git clone git@github.com:voipbin/install.git
    cd install
    pip install -r requirements.txt

    ./voipbin-install init      # interactive wizard + GCP bootstrap
    ./voipbin-install apply     # provision and deploy
    ./voipbin-install verify    # health check

``init`` — interactive wizard
-----------------------------

The ``init`` command runs a twelve-step bootstrap, in order:

1. Preflight checks for the six local tools.
2. ``gcloud`` user auth and Application Default Credentials.
3. **Seven wizard questions**: GCP project ID, region, GKE cluster type
   (zonal or regional), TLS strategy (Let's Encrypt, GCP-managed,
   self-signed, or BYO certificate), Docker image tag strategy (latest
   or pinned via ``config/versions.yaml``), base domain name, and Cloud
   DNS mode (auto or manual).
4. Project existence and billing.
5. Quota check against ``config/gcp_quotas.yaml``.
6. Enable sixteen GCP APIs.
7. Create the ``voipbin-installer`` service account with twelve IAM
   bindings.
8. Create a KMS key ring and crypto key.
9. Generate six secrets (``jwt_key``, ``cloudsql_password``,
   ``redis_password``, ``rabbitmq_user``, ``rabbitmq_password``,
   ``api_signing_key``) and encrypt them into ``secrets.yaml`` with SOPS
   bound to the KMS key.
10. Write ``.sops.yaml``.
11. Save ``config.yaml``.
12. Print a cost summary.

Useful flags:

.. code-block:: bash

    ./voipbin-install init --reconfigure          # re-run the wizard
    ./voipbin-install init --config path/to.yaml  # import existing config
    ./voipbin-install init --skip-api-enable      # APIs already enabled
    ./voipbin-install init --skip-quota-check     # bypass quota gate
    ./voipbin-install init --dry-run              # show what would happen

The two files written are everything you need to reproduce the install:

- ``config.yaml`` — non-sensitive. Safe to commit if you want a record
  per environment.
- ``secrets.yaml`` — SOPS-encrypted with GCP KMS. Decrypt locally with
  ``sops --decrypt secrets.yaml``.

.. warning::

   Back up ``secrets.yaml`` and the KMS key ring. If the key ring is
   destroyed or the GCP project is deleted, ``secrets.yaml`` becomes
   unrecoverable.

``apply`` — three-stage deploy
------------------------------

``./voipbin-install apply`` executes the pipeline in order: Terraform,
then Ansible, then Kubernetes. The pipeline checkpoints between stages,
so if a run fails you can fix the problem and rerun the same command;
it picks up from the failed stage.

Useful flags:

.. code-block:: bash

    ./voipbin-install apply --dry-run         # plan, do not change
    ./voipbin-install apply --auto-approve    # skip prompts
    ./voipbin-install apply --stage terraform # only run this stage
    ./voipbin-install apply --stage ansible
    ./voipbin-install apply --stage k8s

Expected duration on a fresh project:

============================  ==============================================
Stage                          Duration
============================  ==============================================
Terraform                      12 to 18 minutes (GKE cluster creation dominates)
Ansible                        5 to 8 minutes (waits for cloud-init on both VMs)
Kubernetes                     3 to 5 minutes
============================  ==============================================

``verify`` — health check
-------------------------

The ``verify`` command runs ten checks: GKE cluster status, node count,
pod readiness per namespace, presence of the ConfigMap and Secret, DNS
A record resolution, HTTPS reachability for ``api``, ``admin``,
``talk``, and ``meet``, and SIP UDP reachability on port 5060. Each
check prints a clear pass or fail with the underlying command, so you
can rerun individual checks manually.

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
``admin.example.com``         A       External LB IP
``talk.example.com``          A       External LB IP
``meet.example.com``          A       External LB IP
``sip.example.com``           A       External LB IP
============================  ======  ========================

DNS propagation can take up to 48 hours for NS delegation changes. Once
records resolve, rerun ``./voipbin-install verify``.
