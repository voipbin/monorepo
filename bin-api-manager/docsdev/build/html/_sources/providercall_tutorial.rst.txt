.. _providercall-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before placing a providercall, you need:

* A **platform super-admin** authentication token. Obtain one by logging in as a user with the ``PermissionProjectSuperAdmin`` permission. Customer-tier tokens are rejected with 403.
* A **provider ID** (UUID). Obtain one from ``GET https://api.voipbin.net/v1.0/providers``. The provider must exist and must not be soft-deleted.
* A **source address** — typically a phone number in E.164 format that the provider's carrier has pre-allowlisted as an acceptable ``From`` / ``P-Asserted-Identity``. Because this endpoint skips the normal customer-ownership check on the source, the admin is trusted to supply a value the provider will accept.
* At least one **destination address** — typically a phone number in E.164 format.
* (Optional) A **flow ID** (UUID) from ``GET https://api.voipbin.net/v1.0/flows`` if you want a specific flow to execute after the destination answers.
* (Optional) A list of **inline actions** (see the :ref:`Flow Action struct <flow-struct-action>`) to execute post-answer. A temporary flow is auto-created for the call if you supply actions without a ``flow_id``.

.. note:: **AI Implementation Hint**

   The ``POST /v1/providercalls`` endpoint creates real, billable outbound calls. One Call record is created per entry in ``destinations``. The admin's own customer is charged. Existing outbound rate-limiting applies. Do not call this endpoint in automated CI — it will generate real PSTN traffic.

Place a providercall
--------------------

The minimum required body is ``provider_id`` + at least one destination. ``source`` is strongly recommended (without it the call leaves with no caller ID, which most providers reject).

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/providercalls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
            "source": {
                "type": "tel",
                "target": "+14155551234"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+821012345678"
                }
            ],
            "anonymous": "auto"
        }'

    {
        "id": "b7d1c0f6-9a2e-4b3f-8e2a-1c7d5b8a9e0f",
        "customer_id": "6a93f71e-8b2d-4e5f-9a1c-2d3e4f5a6b7c",
        "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "flow_id": "00000000-0000-0000-0000-000000000000",
        "source": {"type": "tel", "target": "+14155551234", "target_name": ""},
        "destinations": [{"type": "tel", "target": "+821012345678", "target_name": ""}],
        "anonymous": "auto",
        "call_ids": ["9f8e7d6c-5b4a-3c2d-1e0f-abcdef012345"],
        "groupcall_ids": [],
        "tm_create": "2026-04-21 23:15:00.000000",
        "tm_update": "2026-04-21 23:15:00.000000",
        "tm_delete": null
    }

Save ``call_ids[0]`` as ``call_id`` and poll it for per-call state.

Place a providercall with inline actions
----------------------------------------

If you want the call to execute a short post-answer sequence (e.g., play a TTS prompt and hang up), supply inline ``actions`` instead of a ``flow_id``. A temporary flow is created for you.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/providercalls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
            "source": {"type": "tel", "target": "+14155551234"},
            "destinations": [{"type": "tel", "target": "+821012345678"}],
            "actions": [
                {"type": "answer"},
                {
                    "type": "talk",
                    "option": {
                        "text": "This is a provider verification call.",
                        "language": "en-US",
                        "gender": "female"
                    }
                },
                {"type": "hangup"}
            ]
        }'

Observe call outcome
--------------------

The ProviderCall record is an audit summary — it does not update as the call progresses. Use the standard call API with each ID in ``call_ids``.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/calls/9f8e7d6c-5b4a-3c2d-1e0f-abcdef012345?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "9f8e7d6c-5b4a-3c2d-1e0f-abcdef012345",
        "status": "progressing",
        "hangup_reason": "",
        "metadata": {
            "route_provider_ids": ["4dbeabd6-f397-4375-95d2-a38411e07ed1"],
            "skip_source_validation": true
        },
        ...
    }

.. note:: **AI Implementation Hint**

   If the call never reaches ``progressing`` and ends with ``hangup_reason: failed`` immediately, the provider rejected the INVITE. Common causes: hostname unreachable, authentication failure on the provider's end, source number not on the provider's allowlist, or destination number not supported by the provider.

Get list of providercalls
-------------------------

Returns records scoped to the authenticated admin's own customer. Optional ``provider_id`` filter narrows to a single provider.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/providercalls?token=<YOUR_AUTH_TOKEN>&provider_id=4dbeabd6-f397-4375-95d2-a38411e07ed1'

    {
        "result": [
            {
                "id": "b7d1c0f6-9a2e-4b3f-8e2a-1c7d5b8a9e0f",
                "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
                "call_ids": ["9f8e7d6c-5b4a-3c2d-1e0f-abcdef012345"],
                ...
            }
        ],
        "next_page_token": "2026-04-21 23:15:00.000000"
    }

Get detail of a providercall
----------------------------

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/providercalls/b7d1c0f6-9a2e-4b3f-8e2a-1c7d5b8a9e0f?token=<YOUR_AUTH_TOKEN>'

Delete (soft) a providercall
----------------------------

Removes the audit record from list results. Does **not** affect the underlying Call records.

.. code::

    $ curl -k --location --request DELETE 'https://api.voipbin.net/v1.0/providercalls/b7d1c0f6-9a2e-4b3f-8e2a-1c7d5b8a9e0f?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "b7d1c0f6-9a2e-4b3f-8e2a-1c7d5b8a9e0f",
        "tm_delete": "2026-04-22 00:00:00.000000",
        ...
    }


Troubleshooting
---------------

* **403 Forbidden:**
    * **Cause:** Token belongs to a customer-tier user. ProviderCalls require ``PermissionProjectSuperAdmin``.
    * **Fix:** Log in as a platform super-admin or use an admin access key.

* **400 Bad Request — provider not found:**
    * **Cause:** The supplied ``provider_id`` does not exist, or has been soft-deleted.
    * **Fix:** List providers with ``GET /v1/providers`` and choose a current ID.

* **Call immediately hangs up (``hangup_reason: failed``):**
    * **Cause:** Provider rejected the INVITE. Common reasons: source not on allowlist, destination unsupported, provider auth failed, hostname unreachable.
    * **Fix:** Review the provider's acceptance rules. Try a source number the provider is known to accept. Cross-check the provider's SIP server health via ``GET /v1/providers/{id}`` (``health_status`` field).

* **Call reports ``no_answer`` / ``busy``:**
    * **Cause:** Signaling reached the provider and the destination, but the far-end did not answer or rejected the call.
    * **Fix:** This usually means the provider is working correctly — the destination phone's state is the issue, not the routing.
