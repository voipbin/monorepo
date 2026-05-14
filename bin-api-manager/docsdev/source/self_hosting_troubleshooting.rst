Operations and troubleshooting
==============================

Day-to-day commands
-------------------

.. code-block:: bash

    # Show the current pipeline state
    ./voipbin-install status

    # Show status as JSON (for scripting)
    ./voipbin-install status --json

    # Verify deployment health (API, SIP, pods, TLS)
    ./voipbin-install verify

    # Run only a specific health check
    ./voipbin-install verify --check http_health

    # Re-render and apply just the K8s manifests after editing a value
    ./voipbin-install apply --stage k8s_apply

    # Re-render the Kamailio VM .env after editing group_vars
    ./voipbin-install apply --stage ansible_run

    # Show Kamailio TLS certificate status
    ./voipbin-install cert status

    # Renew Kamailio TLS certificates
    ./voipbin-install cert renew

    # Show help for any command
    ./voipbin-install --help
    ./voipbin-install apply --help

    # Inspect what secrets are stored
    sops --decrypt secrets.yaml

    # SSH into Kamailio for debugging
    gcloud compute ssh kamailio-0 --tunnel-through-iap \
      --project "$(yq .gcp_project_id config.yaml)"

    # Get logs from a backend service
    kubectl -n bin-manager logs deploy/call-manager --tail 200

    # Tear everything down (irreversible, including the database)
    ./voipbin-install destroy

Common issues
-------------

Terraform: quota exceeded
~~~~~~~~~~~~~~~~~~~~~~~~~

``Error: googleapi: Error 403: Quota exceeded``. Request quota increases
at https://console.cloud.google.com/iam-admin/quotas, then rerun
``./voipbin-install apply``. Common quotas: ``CPUS_ALL_REGIONS``,
``IN_USE_ADDRESSES``, ``SSD_TOTAL_GB``.

Terraform: state lock
~~~~~~~~~~~~~~~~~~~~~

``Error: Error acquiring the state lock``. Only when you are certain no
other process is running Terraform:

.. code-block:: bash

    cd terraform
    terraform force-unlock LOCK_ID

Terraform: API not enabled
~~~~~~~~~~~~~~~~~~~~~~~~~~

``Error: googleapi: Error 403: ... API not enabled``. Re-run
``./voipbin-install init`` to enable APIs, or enable manually:

.. code-block:: bash

    gcloud services enable container.googleapis.com --project PROJECT_ID

Terraform: state bucket missing
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``Error: Failed to get existing workspaces: storage: bucket doesn't exist``.
The GCS state bucket must exist before ``terraform init``. The ``apply``
command creates it automatically; if you are running Terraform manually,
create it first:

.. code-block:: bash

    gsutil mb -p PROJECT_ID gs://PROJECT_ID-voipbin-tf-state

Terraform: permission denied
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``Error: googleapi: Error 403: Required permission``. Re-check the
authenticated principal:

.. code-block:: bash

    gcloud auth list
    gcloud projects get-iam-policy PROJECT_ID

The minimum role set is in ``config/gcp_iam_roles.yaml`` (12 roles). The principal also needs ``roles/compute.osLogin`` and ``roles/compute.osAdminLogin`` for Ansible VM access.

Ansible: IAP tunnel connection failed
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``ERROR! Timeout waiting for connection`` or
``Permission denied (publickey)``.

1. Verify IAP SSH works directly:
   ``gcloud compute ssh VM_NAME --tunnel-through-iap --project PROJECT_ID --zone ZONE``.
2. Ensure the IAP API is enabled.
3. Check the IAP firewall rule exists.
4. Verify the principal has ``roles/iap.tunnelResourceAccessor``.

Ansible: docker install failed on a fresh VM
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``Could not get lock /var/lib/dpkg/lock`` typically means cloud-init is
still running. Wait for it and rerun the stage:

.. code-block:: bash

    gcloud compute ssh kamailio-0 --tunnel-through-iap \
      -- 'cloud-init status --wait'
    ./voipbin-install apply --stage ansible

Kubernetes: pod in CrashLoopBackOff
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash

    kubectl get pods -n bin-manager
    kubectl logs POD_NAME -n bin-manager
    kubectl describe pod POD_NAME -n bin-manager

Most common causes:

- Cloud SQL Proxy not ready (check ``infrastructure`` namespace first).
- Redis or RabbitMQ not ready.
- Missing ConfigMap or Secret values.

Kubernetes: ImagePullBackOff
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

1. Verify the image exists at the tag declared in
   ``config/versions.yaml``.
2. GKE nodes need internet access through Cloud NAT.
3. If images live in a private GAR or GCR repository, confirm the GKE
   node service account has ``roles/artifactregistry.reader`` (GAR) or
   ``roles/storage.objectViewer`` on the registry bucket (legacy GCR).

Cloud SQL: connection refused from pods
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash

    kubectl get pods -n infrastructure -l app=cloudsql-proxy
    kubectl logs -n infrastructure -l app=cloudsql-proxy

Verify the Cloud SQL instance is up, the proxy service account has
permissions, and the connection name matches the Terraform output.

DNS: domain not resolving
~~~~~~~~~~~~~~~~~~~~~~~~~

When ``dns_mode: auto``:

.. code-block:: bash

    gcloud dns managed-zones describe voipbin-zone --project PROJECT_ID \
      --format="value(nameServers)"

Update your registrar's NS records to point at those nameservers. NS
delegation can take up to 48 hours.

When ``dns_mode: manual``: pull the external LB IP from Terraform
outputs and create A records for ``api``, ``hook``, ``admin``, ``talk``,
``meet``, and ``sip`` subdomains at your DNS provider.

SIP: devices cannot register
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

1. Confirm Kamailio is running on the VM:
   ``docker compose ps`` via IAP SSH.
2. Inspect the logs:
   ``docker compose logs --tail 100``.
3. List the Kamailio load balancer resources and check health, since the
   exact resource names vary by deployment environment:

   .. code-block:: bash

       gcloud compute target-pools list --filter="name~kamailio"
       gcloud compute forwarding-rules list --filter="name~kamailio"

Audio: calls connect but no audio
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

1. Confirm RTPEngine is running on the VM.
2. Check the firewall allows the RTP UDP port range.
3. Verify RTPEngine sees the correct external IP, especially after
   moving regions.

Rollback
--------

Kubernetes rollout:

.. code-block:: bash

    kubectl rollout undo deployment/DEPLOYMENT_NAME -n NAMESPACE

Terraform state from a previous version:

.. code-block:: bash

    gsutil ls -la gs://PROJECT_ID-voipbin-tf-state/default.tfstate
    gsutil cp gs://PROJECT_ID-voipbin-tf-state/default.tfstate#VERSION terraform.tfstate
    cd terraform && terraform apply

Full teardown and rebuild:

.. code-block:: bash

    ./voipbin-install destroy
    ./voipbin-install apply

Where to go next
----------------

- The installer repo's own ``docs/troubleshooting.md`` has a wider list
  of error symptoms by stage.
- The installer's ``README.md`` lists exact IAM roles and Terraform
  outputs.
- For platform support, contact ``support@voipbin.net``.
