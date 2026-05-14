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

Estimated monthly costs in ``us-central1`` (on-demand, list price). Costs vary by
region. You are responsible for all charges incurred on your GCP project.

**Minimum config** (``gke_node_count=1``, ``kamailio_count=1``, ``rtpengine_count=1``):

.. list-table::
   :header-rows: 1
   :widths: 55 25

   * - Resource
     - Cost/mo (USD)
   * - GKE Control Plane
     - $0 zonal, ~$74 regional
   * - GKE Node (1x n1-standard-2 + 100 GB disk)
     - ~$213
   * - Kamailio VM (1x f1-micro + 30 GB disk)
     - ~$7
   * - RTPEngine VM (1x f1-micro + 30 GB disk)
     - ~$7
   * - Cloud SQL MySQL db-f1-micro + 10 GB SSD
     - ~$13
   * - Static External IPs (8x in-use)
     - ~$23
   * - Load Balancers (Kamailio NLB 3 rules + K8s LB 5 services)
     - ~$146
   * - Cloud NAT gateway
     - ~$32
   * - Cloud DNS, GCS, KMS
     - ~$1
   * - **Total**
     - **~$442 zonal / ~$516 regional**

**Default config** (``gke_node_count=2``, ``kamailio_count=2``, ``rtpengine_count=2``):

.. list-table::
   :header-rows: 1
   :widths: 55 25

   * - Resource
     - Cost/mo (USD)
   * - GKE Control Plane
     - $0 zonal, ~$74 regional
   * - GKE Nodes (2x n1-standard-2 + 100 GB disks)
     - ~$426
   * - Kamailio VMs (2x f1-micro + 30 GB disks)
     - ~$14
   * - RTPEngine VMs (2x f1-micro + 30 GB disks)
     - ~$14
   * - Cloud SQL MySQL db-f1-micro + 10 GB SSD
     - ~$13
   * - Static External IPs (9x in-use)
     - ~$26
   * - Load Balancers (Kamailio NLB 3 rules + K8s LB 5 services)
     - ~$146
   * - Cloud NAT gateway
     - ~$32
   * - Cloud DNS, GCS, KMS
     - ~$1
   * - **Total**
     - **~$672 zonal / ~$746 regional**

These figures are on-demand list prices and do not include committed-use
discounts, data egress, or usage beyond the included free tiers.

.. warning::

   ``./voipbin-install destroy`` is irreversible. It permanently deletes
   every resource the installer created, including the Cloud SQL instance
   and its backups. Export any data you need before destroying.
