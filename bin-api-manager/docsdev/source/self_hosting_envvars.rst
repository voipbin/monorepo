Environment variables and post-install configuration
====================================================

The defaults produce a deployment that boots and passes ``verify``, but
production workloads need adjustments. This section enumerates what to
touch and where.

1. Domain-related values
------------------------

Derived from the ``domain`` you entered in the wizard. Using
``voipbin.example.com`` as an example, the rendered values are:

==========================  =================================  ===========================================
Variable                    Value                              Location
==========================  =================================  ===========================================
``DOMAIN``                  ``voipbin.example.com``            ``k8s/backend/configmap.yaml``
``BASE_DOMAIN``             ``voipbin.example.com``            Kamailio VM ``/opt/kamailio-docker/.env``
``DOMAIN_NAME_EXTENSION``   ``registrar.voipbin.example.com``  Kamailio VM ``.env``
``DOMAIN_NAME_TRUNK``       ``trunk.voipbin.example.com``      Kamailio VM ``.env``
==========================  =================================  ===========================================

To change the base domain after install, edit ``domain`` in
``config.yaml`` and rerun ``./voipbin-install apply``. Both the Ansible
env template and the Kubernetes ConfigMap regenerate consistently.

2. Generated credentials (RabbitMQ, Redis, MySQL, JWT, API signing)
-------------------------------------------------------------------

These live in the SOPS-encrypted ``secrets.yaml`` and are wired into the
ConfigMap, Secret, and VM ``.env`` automatically. Rotate them through
SOPS:

.. code-block:: bash

    sops secrets.yaml
    ./voipbin-install apply

3. SIP and PSTN settings
------------------------

These are not part of the wizard. Edit them in the appropriate place and
rerun the relevant stage.

PSTN allowlist
~~~~~~~~~~~~~~

Kamailio only accepts inbound PSTN traffic from explicitly whitelisted
source IPs. Populate the list with ``VOIPBIN_PSTN_WHITELIST_IPS`` and
rerun the Ansible stage:

.. code-block:: bash

    export VOIPBIN_PSTN_WHITELIST_IPS="203.0.113.10,198.51.100.4"
    ./voipbin-install apply --stage ansible

The value flows through ``ansible/group_vars/all.yml`` into the Kamailio
env template as ``PSTN_WHITELIST_IPS``.

SIP auth backend
~~~~~~~~~~~~~~~~

The four ``KAMAILIO_AUTH_*`` variables in the Kamailio env template
default to empty. Production deployments point them at a database that
holds SIP credentials:

- ``KAMAILIO_AUTH_DB_URL``: connection string Kamailio uses for the
  auth database.
- ``KAMAILIO_AUTH_USER_COLUMN``: defaults to ``username``.
- ``KAMAILIO_AUTH_DOMAIN_COLUMN``: defaults to ``realm``.
- ``KAMAILIO_AUTH_PASSWORD_COLUMN``: defaults to ``password``.

Set them via Ansible extra vars or by editing
``ansible/group_vars/kamailio.yml`` and rerunning
``./voipbin-install apply --stage ansible``.

RTPEngine port range
~~~~~~~~~~~~~~~~~~~~

The default media port range is ``20000-65535``. If your network only
opens ``20000-30000``, set ``rtpengine_port_max: 30000`` in
``ansible/group_vars/rtpengine.yml`` and rerun the Ansible stage. Make
sure the firewall rule matches.

4. TLS certificate strategy
---------------------------

The ``tls_strategy`` chosen in the wizard governs how certificates are
issued:

- ``letsencrypt``: cert-manager issues per-host certificates via
  HTTP-01 once DNS resolves. DNS must resolve first, otherwise the
  challenge fails and the certificate sits in ``Pending``. Verify with
  ``kubectl get certificate -A``.
- ``gcp-managed``: a GCP-managed certificate is attached to the
  Ingress and provisioned by Google. Same DNS dependency.
- ``self-signed``: a self-signed certificate is installed immediately.
  Use only for staging or local-only deployments.
- ``byoc``: bring your own. Drop ``tls.crt`` and ``tls.key`` into the
  ``voipbin-tls`` Secret in ``bin-manager`` before
  ``./voipbin-install apply --stage k8s``.

To change strategies after install, edit ``tls_strategy`` in
``config.yaml`` and rerun apply.

5. Third-party integrations
---------------------------

VoIPBin services can integrate with external providers for LLM, ASR,
TTS, email, SMS, and payments. The installer ships with these slots
intentionally **empty**, because they are operator-specific and not
free. Add them as Kubernetes Secrets in the ``bin-manager`` namespace,
then mount them into the relevant deployments.

Provider categories you will likely need to wire up:

- An LLM or AI provider.
- A speech-to-text provider.
- A text-to-speech provider.
- An email API or SMTP provider.
- An SMS provider.
- A payment provider for billing flows.

Pick providers based on your own region, compliance, and pricing
constraints. Until the relevant keys are wired up, the corresponding
flow actions return ``provider not configured`` errors; the platform
itself stays healthy and the rest of the channels remain usable.

Recommended pattern: a separate Secret per provider so rotation is
independent.

.. code-block:: bash

    kubectl -n bin-manager create secret generic voipbin-llm \
      --from-literal=LLM_API_KEY=...

Then patch the deployment to mount it via ``envFrom``. Track the provider
matrix per service in your own runbook; upstream manifests deliberately
do not enumerate provider keys because the supported list evolves.

6. Scaling
----------

The defaults are minimal:

- GKE: 2 nodes of ``n1-standard-2``.
- Kamailio: 1 VM of ``f1-micro``.
- RTPEngine: 1 VM of ``f1-micro``.
- Backend deployments: 1 replica each, ``50m`` CPU and ``64Mi`` memory
  request, ``200m`` and ``256Mi`` limits.

For anything beyond a demo, raise the VM types and replica counts.
Adjust ``gke_machine_type``, ``gke_node_count``, ``vm_machine_type``,
``kamailio_count``, and ``rtpengine_count`` in ``config.yaml`` and
rerun ``./voipbin-install apply``.

Backend replica counts live in the per-service manifest under
``k8s/backend/services/<name>.yaml``. For now, edit those files
directly; a future installer revision will surface scaling profiles
through ``config.yaml``.

Consolidated environment variable map
-------------------------------------

Every variable a fresh install touches comes from one of three sources:
the wizard, generated secrets, or operator overrides.

Wizard-sourced (``config.yaml``)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

- ``gcp_project_id``
- ``region``, ``zone``
- ``gke_type`` (``zonal`` or ``regional``)
- ``tls_strategy`` (``letsencrypt``, ``gcp-managed``, ``self-signed``,
  ``byoc``)
- ``image_tag_strategy`` (``latest`` or ``pinned``)
- ``domain``
- ``dns_mode`` (``auto`` or ``manual``)
- ``gke_machine_type``, ``gke_node_count``
- ``vm_machine_type``, ``kamailio_count``, ``rtpengine_count``

Any of these can be overridden at runtime with the ``VOIPBIN_`` prefix
(uppercased), for example ``VOIPBIN_REGION=europe-west1``.

Generated and SOPS-encrypted (``secrets.yaml``)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

- ``jwt_key``
- ``cloudsql_password``
- ``redis_password``
- ``rabbitmq_user``, ``rabbitmq_password``
- ``api_signing_key``

Ansible variables (Kamailio VMs)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Defined in ``ansible/group_vars/all.yml`` and
``ansible/group_vars/kamailio.yml``. The ones operators routinely change:

- ``domain`` via ``VOIPBIN_DOMAIN`` or ``config.yaml``.
- ``image_tag`` via ``VOIPBIN_IMAGE_TAG``.
- ``pstn_whitelist_ips`` via ``VOIPBIN_PSTN_WHITELIST_IPS``.
- ``kamailio_auth_db_url`` and the three ``kamailio_auth_*_column``
  variables via extra-vars or ``group_vars/kamailio.yml``.
- ``kamailio_shm_size``, ``kamailio_pkg_size``: memory tuning.
- ``pike_enabled``, ``pike_rate``, ``pike_timeout``: anti-flood.
- ``homer_enabled``, ``homer_uri``: HEP capture.

Kubernetes ConfigMap (``voipbin-config``)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In namespaces ``bin-manager`` and ``infrastructure``. Placeholders are
substituted at apply time:

- ``DOMAIN``, ``DB_HOST``, ``DB_PORT``, ``DB_NAME``,
  ``CLOUDSQL_CONNECTION_NAME``
- ``REDIS_URL``, ``RABBITMQ_URL``, ``CLICKHOUSE_URL``

Kubernetes Secret (``voipbin-secret``)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In namespace ``bin-manager``:

- ``JWT_KEY``, ``DB_USER``, ``DB_PASSWORD``, ``REDIS_PASSWORD``,
  ``RABBITMQ_PASSWORD``, ``API_SIGNING_KEY``
