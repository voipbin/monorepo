Overview
========

The installer is an eight-stage pipeline driven by the
``voipbin-install`` CLI.

The eight stages run in order when you execute ``./voipbin-install apply``:

1. **terraform_init** initializes the Terraform backend (GCS state bucket)
   and downloads providers.
2. **reconcile_imports** detects any GCP resources that already exist
   outside Terraform state and imports them, preventing 409 conflicts on
   resume.
3. **terraform_apply** provisions GCP infrastructure: a custom VPC, GKE
   cluster, Cloud SQL (MySQL 8.0), Kamailio and RTPEngine VMs, DNS zone,
   load balancers, KMS key ring for SOPS, and GCS buckets.
4. **reconcile_outputs** reads Terraform outputs (IPs, connection names)
   into ``config.yaml`` so later stages can consume them.
5. **k8s_apply** deploys VoIPBin services to the GKE cluster using
   manifests under ``k8s/``, with placeholders substituted from Terraform
   outputs and SOPS-decrypted secrets.
6. **reconcile_k8s_outputs** reads Kubernetes load balancer IPs into
   ``config.yaml``.
7. **cert_provision** issues Kamailio TLS certificates.
8. **ansible_run** configures Kamailio and RTPEngine VMs over an IAP
   tunnel and renders their Docker Compose ``.env`` files from templates.

The pipeline is resumable: if a stage fails, fix the issue and rerun
``./voipbin-install apply``; it continues from where it left off. Run a
single stage with ``./voipbin-install apply --stage <name>`` (for example
``--stage ansible_run``).

What gets deployed
------------------

A successful install produces:

- **VPC and connectivity**: custom VPC ``10.0.0.0/16``, Cloud NAT with a
  static IP for outbound, Cloud Router, eight firewall rules covering
  SIP, RTP, IAP SSH, GKE internal, and health checks.
- **GKE cluster**: zonal or regional, 2 nodes of ``n1-standard-2`` by
  default, private nodes, shielded instances, COS_CONTAINERD image,
  REGULAR release channel.
- **Kamailio VM**: 1x ``f1-micro`` by default, SIP/TLS/WSS proxy reached
  through an external network load balancer.
- **RTPEngine VM**: 1x ``f1-micro`` by default, RTP media relay with a
  static external IP for direct media paths.
- **Cloud SQL MySQL 8.0**: ``db-f1-micro`` instance, SSL required, daily
  automated backups, Cloud SQL Proxy deployed as a sidecar in GKE.
- **Kubernetes workloads**: backend microservices in the ``bin-manager``
  namespace, Asterisk in ``voip``, Redis, RabbitMQ, ClickHouse, and Cloud
  SQL Proxy in ``infrastructure``, plus three frontend apps
  (``square-admin``, ``square-talk``, ``square-meet``) in the
  ``square-manager`` namespace.
- **DNS records**: six A records are required. With ``example.com`` as
  your domain: ``api.example.com``, ``hook.example.com``,
  ``admin.example.com``, ``talk.example.com``, ``meet.example.com``, and
  ``sip.example.com``. The first five point at the GKE load balancer IP
  and the last points at the Kamailio external load balancer IP.

Cost
----

Estimated monthly costs for the minimal default sizing in
``us-central1``:

============================  ==============================
Resource                       Cost (USD)
============================  ==============================
GKE control plane              $0 zonal, ~$73 regional
GKE nodes (2x n1-standard-2)   ~$97
Kamailio VM (f1-micro)         ~$6
RTPEngine VM (f1-micro)        ~$6
Cloud SQL (db-f1-micro)        ~$13
Cloud NAT                      ~$10
External IPs                   ~$8
Network load balancers         ~$20
DNS, GCS, KMS, disks           ~$6
**Total**                      **~$170 zonal, ~$243 regional**
============================  ==============================

Costs vary by region. The figures above add up the line items in the
table and are list-price snapshots, not committed-use prices; the
installer's ``README.md`` may show a slightly different headline figure
depending on which sizing snapshot it was last regenerated against. You
are responsible for all charges incurred on your GCP project.

.. warning::

   ``./voipbin-install destroy`` is irreversible. It permanently deletes
   every resource the installer created, including the Cloud SQL instance
   and its backups. Export any data you need before destroying.
