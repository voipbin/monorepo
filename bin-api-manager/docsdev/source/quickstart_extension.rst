.. _quickstart-extension:

Extension & Softphone Setup
----------------------------
Create a SIP extension and register a softphone (Linphone) to receive calls from VoIPBIN. This is required for the :ref:`Real-Time Voice Interaction <quickstart-realtime>` scenario.

Prerequisites
~~~~~~~~~~~~~

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart-authentication>`.
* Your customer ID (UUID). Obtained from ``GET https://api.voipbin.net/v1.0/customer`` or from your admin console profile.
* Linphone softphone installed on your computer or mobile device. Download from `linphone.org <https://www.linphone.org/>`_.

.. note:: **AI Implementation Hint**

   This section requires a human with a softphone to complete the registration and answer calls. AI agents can execute the API call to create the extension and instruct the human for the Linphone registration.

Create an extension
^^^^^^^^^^^^^^^^^^^^
Create a SIP extension that your softphone will register to. The ``name`` (String, Required) identifies the extension for dialing. The ``detail`` (String, Required) is a description. The ``extension`` (String, Required) and ``password`` (String, Required) are used for SIP authentication.

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/extensions?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "quickstart-phone",
            "detail": "Quickstart softphone extension",
            "extension": "quickstart1",
            "password": "your-secure-password-here"
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "quickstart-phone",
        "detail": "Quickstart softphone extension",
        "extension": "quickstart1",
        "username": "quickstart1",
        "password": "your-secure-password-here",
        "domain_name": "ab12.reg.voipbin.net",
        "direct_hash": "a8f3b2c1d4e5",
        "tm_create": "2026-02-21T10:00:00.000000Z",
        "tm_update": "",
        "tm_delete": ""
    }

The ``id`` (UUID) is the extension's unique identifier — use it for ``GET /extensions/{id}``, ``PUT /extensions/{id}``, or ``DELETE /extensions/{id}`` operations. For dialing, use the ``name`` field instead. Save the ``name`` (String) — you will use it as the call destination in the :ref:`Real-Time Voice Interaction <quickstart-realtime>` scenario.

.. note:: **AI Implementation Hint**

   The ``extension`` and ``password`` are SIP credentials, not VoIPBIN login credentials. The ``name`` field is the extension identifier used when dialing (e.g., ``"target_name": "quickstart-phone"`` in the call request). The response includes both ``extension`` and ``username`` fields — they contain the same value (``username`` is a Kamailio-internal mirror of ``extension``). Choose a memorable ``extension`` value and a strong ``password``.

Register Linphone
^^^^^^^^^^^^^^^^^^
Configure your Linphone softphone to register with VoIPBIN using the extension credentials created above.

**Linphone configuration:**

+-------------------+------------------------------------------------------------+
| Field             | Value                                                      |
+===================+============================================================+
| Username          | ``quickstart1`` (from the ``extension`` field above)       |
+-------------------+------------------------------------------------------------+
| Password          | The password you set when creating the extension           |
+-------------------+------------------------------------------------------------+
| Domain            | The ``domain_name`` from the extension response            |
|                   | (e.g. ``ab12.reg.voipbin.net``)                            |
+-------------------+------------------------------------------------------------+
| Transport         | UDP (use TCP or TLS if SIP messages are too large for UDP) |
+-------------------+------------------------------------------------------------+

Use the ``domain_name`` value returned when you created the extension (also visible via ``GET https://api.voipbin.net/v1.0/extensions``). It is your customer-specific SIP domain in the format ``<label>.reg.voipbin.net`` (a short 4-character label, e.g. ``ab12.reg.voipbin.net``). Do not construct the domain yourself. Always copy it from the API response. Accounts created before the domain migration may still receive the legacy ``<customer-id>.registrar.voipbin.net`` format; such domains are still accepted for SIP registration during the migration window, but inbound calls are only routed to the current ``domain_name`` served by the API, so always configure your softphone with that value.

**Setup steps (Linphone desktop):**

1. Open Linphone and go to **Preferences** > **Account** (or **SIP Account** on mobile).
2. Select **I already have a SIP account** (or **Use SIP account**).
3. Enter the username, password, and domain from the table above.
4. Save. Linphone should show **Registered** status within a few seconds.

If registration succeeds, the status indicator turns green. If it fails, see Troubleshooting below.

Troubleshooting
~~~~~~~~~~~~~~~

* **Extension creation returns 400 Bad Request:**
    * **Cause:** Missing required fields (``name``, ``detail``, ``extension``, ``password``).
    * **Fix:** Ensure all four fields are present in the request body.

* **Linphone shows "Registration failed" or "408 Timeout":**
    * **Cause:** Incorrect domain, extension/username, or password. The domain must be the extension's ``domain_name``.
    * **Fix:** Verify the domain matches the ``domain_name`` field from the extension response exactly (e.g. ``ab12.reg.voipbin.net``). Double-check the ``extension`` and ``password`` match exactly what was set when creating the extension. Ensure UDP port 5060 is not blocked by your firewall. If SIP messages are too large for a single UDP datagram, switch the transport to TCP or TLS.
