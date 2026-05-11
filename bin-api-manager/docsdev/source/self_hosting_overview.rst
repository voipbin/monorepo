Overview
========

The installer is a three stage pipeline driven by the
``voipbin-install`` CLI.

1. **Terraform** provisions GCP infrastructure: a custom VPC, GKE cluster,
   Cloud SQL (MySQL 8.0), Kamailio and RTPEngine VMs, DNS zone, load
   balancers, KMS key ring for SOPS, and GCS buckets.
2. **Ansible** configures the Kamailio and RTPEngine VMs over an IAP
   tunnel and renders their Docker Compose ``.env`` files from templates.
3. **Kubernetes** applies the manifests under ``k8s/`` to the GKE cluster
   with placeholders substituted from Terraform outputs and SOPS-decrypted
   secrets.

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
  (``square-admin``, ``square-talk``, ``square-meet``) and an Ingress
  fronted by TLS certificates.

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

Costs vary by region. You are responsible for all charges incurred on your
GCP project.

.. warning::

   ``./voipbin-install destroy`` is irreversible. It permanently deletes
   every resource the installer created, including the Cloud SQL instance
   and its backups. Export any data you need before destroying.
