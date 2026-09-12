First login and API access
=============================

A completed install has a running stack, but no way to sign in yet
unless you opted in to the dev-seed account. This section covers how to
create a real admin account and start calling the API.

.. warning::

   The ``admin@localhost`` account and extensions ``1000``, ``2000``,
   and ``3000`` (three specific extensions, not a range) only exist when
   the stack was started with ``VOIPBIN_SANDBOX_DEV_SEED=true``. Leave
   this flag unset or ``false`` on any install reachable beyond
   localhost; see the credentials table in Prerequisites.

If your first login attempt fails outright (curl connection errors,
browser certificate warnings), confirm TLS trust first: see
Troubleshooting's DNS/certificate checks and, for internal mode, the
mkcert CA trust step in Prerequisites. A first-login failure is very
often a certificate problem, not an auth problem.

Two ways to create your first account
----------------------------------------

**Path A: CLI bootstrap (recommended for a fresh self-hosted install).**
Works immediately, no email provider required.

**Path B: email-based signup**, the same flow the hosted service at
``voipbin.net`` uses (``POST /auth/signup``). This only works if you have
already configured an email provider (``SENDGRID_API_KEY`` or
``MAILGUN_API_KEY``, see Provider configuration) so the verification
email can actually be delivered. Without one, the signup call succeeds
but the verification email never arrives. Use this path only if you have
already set up email delivery; otherwise use Path A.

Path A: CLI bootstrap
--------------------------

Create a customer and an admin agent directly via the manager CLIs, then
set the agent's password explicitly (a fresh admin agent is created with
a random, unusable password):

.. code-block:: bash

    # 1. Create the customer
    docker exec voipbin-customer-mgr /app/bin/customer-control customer create \
      --name "My Company" \
      --email "admin@example.com"

    # 2. Get the customer ID
    CUSTOMER_ID=$(docker exec voipbin-customer-mgr /app/bin/customer-control customer list 2>/dev/null \
      | jq -r '.[] | select(.email == "admin@example.com") | .id')

    # 3. Wait a few seconds for agent-manager to process the event via RabbitMQ
    sleep 5

    # 4. Get the auto-created admin agent's ID
    AGENT_ID=$(docker exec voipbin-agent-mgr /app/bin/agent-control agent list \
      --customer-id "$CUSTOMER_ID" 2>/dev/null | jq -r '.[0].id')

    # 5. Set a real password
    docker exec voipbin-agent-mgr /app/bin/agent-control agent update-password \
      --id "$AGENT_ID" \
      --password "your-secure-password"

Get a JWT token
~~~~~~~~~~~~~~~~~~

.. code-block:: bash

    curl -sk -X POST https://api.voipbin.test:8443/auth/login \
      -H "Content-Type: application/json" \
      -d '{"username": "admin@example.com", "password": "your-secure-password"}'

    # Response: {"username": "...", "token": "JWT_TOKEN_HERE"}

Create an API key (accesskey)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

An accesskey is a long-lived API token, separate from the short-lived
JWT above, meant for scripts and integrations:

.. code-block:: bash

    docker exec voipbin-customer-mgr /app/bin/customer-control accesskey create \
      --customer-id "$CUSTOMER_ID" \
      --name "API Key" \
      --detail "For automation" \
      --expire 87600h

``--expire 87600h`` is roughly 10 years, a practically non-expiring key
for automation use. Use a shorter value for keys you intend to rotate.

Create your first extension
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash

    TOKEN="<jwt_token_from_above>"

    curl -sk -X POST https://api.voipbin.test:8443/v1.0/extensions \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d '{"extension": "2000", "password": "your-secure-password-here", "name": "Extension 2000"}'

Path B: email-based signup
-------------------------------

Once an email provider is configured (see Provider configuration), the
hosted-service signup flow works identically for a self-hosted install:
``POST /auth/signup`` against your own ``api.<domain>`` endpoint instead
of ``api.voipbin.net``. See the Quickstart guide's Signup page for the
full request/response shape and troubleshooting; the mechanics are
identical, only the base URL differs.
