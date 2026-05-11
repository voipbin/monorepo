Operations and troubleshooting
==============================

Day-to-day commands
-------------------

.. code-block:: bash

    # Show the current pipeline state
    ./voipbin-install status

    # Re-render and apply just the K8s manifests after editing a value
    ./voipbin-install apply --stage k8s

    # Re-render the Kamailio VM .env after editing group_vars
    ./voipbin-install apply --stage ansible

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
outputs and create A records for ``api``, ``admin``, ``talk``, ``meet``,
and ``sip`` subdomains at your DNS provider.

SIP: devices cannot register
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

1. Confirm Kamailio is running on the VM:
   ``docker compose ps`` via IAP SSH.
2. Inspect the logs:
   ``docker compose logs --tail 100``.
3. Check the external LB health:
   ``gcloud compute backend-services get-health voipbin-kamailio-backend --region REGION``.

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
