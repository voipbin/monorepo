Prerequisites
=============

Local tools
-----------

Install these on the workstation that will run the installer. The
preflight step in ``voipbin-install init`` verifies each version.

================  ================  =================================================
Tool              Min. version      Notes
================  ================  =================================================
gcloud CLI        400.0.0           Both ``gcloud auth login`` and
                                    ``gcloud auth application-default login`` required.
terraform         1.5.0             Bundled under ``terraform/`` in the installer repo.
ansible           2.15.0            Install via ``pip install ansible``.
kubectl           1.28.0            Used by the installer and for verification.
python3           3.10.0            Required by the installer CLI.
sops              3.7.0             Encrypts ``secrets.yaml`` with GCP KMS.
================  ================  =================================================

Python dependencies are installed with ``pip install -r requirements.txt``
after cloning the installer repo.

GCP account
-----------

The installer refuses to proceed unless the following are true:

- A GCP project exists with **billing enabled**.
- The authenticated principal has Owner or Editor on the project, or the
  least-privilege set of twelve roles listed in
  ``config/gcp_iam_roles.yaml`` (Compute Admin, Container Admin, Cloud
  SQL Admin, DNS Admin, Cloud KMS Admin, Secret Manager Admin, Service
  Account Admin, Service Account User, Project IAM Admin, Storage Admin,
  Service Usage Admin, IAP-Secured Tunnel User).
- You own a domain name. The installer uses
  ``api.<domain>``, ``admin.<domain>``, ``talk.<domain>``,
  ``meet.<domain>``, and ``sip.<domain>``.

GCP quotas
----------

The ``init`` command checks regional quotas. New GCP projects ship with
8 vCPUs and 8 in-use external IPs, which is below what VoIPBin needs.
Request increases for the following before installing if you do not want
the apply to fail mid stage:

==========================  ====================  ===============================
Quota                       Minimum required      Notes
==========================  ====================  ===============================
CPUs (region)               12                    2x GKE nodes + 2x VoIP VMs
In-use external IPs         10                    NAT, LBs, RTPEngine static IPs
Static external IPs         4                     usually sufficient by default
SSD total (GB)              100                   usually sufficient by default
==========================  ====================  ===============================

Quota increases are requested at
https://console.cloud.google.com/iam-admin/quotas.

GCP APIs
--------

The ``init`` command automatically enables sixteen APIs on the project:
``compute``, ``container``, ``sqladmin``, ``dns``, ``cloudkms``,
``secretmanager``, ``cloudresourcemanager``, ``iam``,
``servicenetworking``, ``storage``, ``storage-api``, ``logging``,
``monitoring``, ``oslogin``, ``serviceusage``, and ``iap``.

Two authentications, not one
----------------------------

VoIPBin's installer depends on **both** of the following being live, and
this catches operators by surprise often enough to be worth calling out:

1. ``gcloud auth login`` for the human/CLI principal.
2. ``gcloud auth application-default login`` for Terraform and SOPS,
   which read Application Default Credentials.

The preflight in ``init`` checks both and offers to refresh ADC if it is
missing or expired.
