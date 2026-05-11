.. _direct-hash-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low — Direct hash is a routing shortcut; no complex setup required.
   * **Cost:** Free. Direct hash creation and regeneration incur no charges.
   * **Async:** No. Regenerate returns immediately with the updated resource.

Direct hash provides simplified public SIP URIs for VoIPBIN resources. Instead of requiring callers to know a customer-specific domain (e.g., ``sip:office1@abc123.registrar.voipbin.net``), direct hash exposes a short, unique address on a shared domain: ``sip:direct.<hash>@sip.voipbin.net``. This allows external SIP devices, trunks, and partners to reach your resources without any customer-specific configuration.

Seven resource types support direct hash: **extensions**, **agents**, **conferences**, **queues**, **flows**, **AIs**, and **teams**.

.. note:: **AI Implementation Hint**

   To place a call to a direct hash resource, use the SIP URI ``sip:direct.<hash>@sip.voipbin.net`` as the destination. The ``<hash>`` value is the ``direct_hash`` field from the resource's GET response. If ``direct_hash`` is empty, call the resource's ``direct-hash-regenerate`` endpoint first to create one.


How It Works
------------

**Hash Format**

Each direct hash consists of a ``direct.`` prefix followed by 12 hexadecimal characters generated from 6 cryptographically random bytes.

::

    Format:    direct.<12 hex chars>
    Example:   direct.a1b2c3d4e5f6
    SIP URI:   sip:direct.a1b2c3d4e5f6@sip.voipbin.net

**Routing Flow**

When an external caller dials a direct hash SIP URI, VoIPBIN resolves the hash to the underlying resource, automatically creates an activeflow with the appropriate action, and executes it.

::

    External Caller                              VoIPBIN
         |                                          |
         | INVITE                                    |
         | sip:direct.<hash>@sip.voipbin.net        |
         +----------------------------------------->|
         |                                          |
         |                                          | 1. Lookup hash in database
         |                                          | 2. Resolve resource type and ID
         |                                          | 3. Create activeflow automatically
         |                                          | 4. Execute activeflow
         |                                          |
         |           Call connected                  |
         |<-----------------------------------------+
         |                                          |
         |           Media flow                     |
         |<---------------------------------------->|

The activeflow action depends on the resource type:

- **Extension / Agent**: The activeflow dials the registered SIP device and bridges audio when answered.
- **Conference**: The activeflow joins the caller into the conference bridge.
- **Queue**: The activeflow joins the caller into the queue, waiting for an available agent.
- **Flow**: The activeflow executes the flow's defined action sequence as-is.
- **AI**: The activeflow starts an AI voice agent conversation with the caller.
- **Team**: The activeflow starts a multi-agent AI team conversation with the caller.

**Comparison with Standard Access**

In both cases, VoIPBIN internally creates an activeflow and executes it. The difference is how the caller reaches the resource — not what happens after.

::

    Standard access:
    Caller -> inbound number -> flow -> resource

    Direct hash access:
    Caller -> sip:direct.<hash>@sip.voipbin.net -> resource

Standard access requires purchasing an inbound number, creating a flow with the appropriate action, and routing the number to that flow. Direct hash eliminates all of this — VoIPBIN automatically creates the appropriate activeflow internally.


Supported Resources
-------------------

+---------------+----------------+------------------------------------------------------------------------------------+-------------------------------------------+
| Resource      | Auto-Created   | Regenerate Endpoint                                                                | Documentation                             |
+===============+================+====================================================================================+===========================================+
| Extension     | Yes            | ``POST https://api.voipbin.net/v1.0/extensions/{id}/direct-hash-regenerate``       | :ref:`extension-overview-direct`          |
+---------------+----------------+------------------------------------------------------------------------------------+-------------------------------------------+
| Conference    | Yes            | ``POST https://api.voipbin.net/v1.0/conferences/{id}/direct-hash-regenerate``      | :ref:`conference-overview`                |
+---------------+----------------+------------------------------------------------------------------------------------+-------------------------------------------+
| Team          | Yes            | ``POST https://api.voipbin.net/v1.0/teams/{id}/direct-hash-regenerate``            | :ref:`team-overview`                      |
+---------------+----------------+------------------------------------------------------------------------------------+-------------------------------------------+
| Queue         | Yes            | ``POST https://api.voipbin.net/v1.0/queues/{id}/direct-hash-regenerate``           | :ref:`queue-overview`                     |
+---------------+----------------+------------------------------------------------------------------------------------+-------------------------------------------+
| Flow          | Yes            | ``POST https://api.voipbin.net/v1.0/flows/{id}/direct-hash-regenerate``            | :ref:`flow-overview`                      |
+---------------+----------------+------------------------------------------------------------------------------------+-------------------------------------------+
| Agent         | No             | ``POST https://api.voipbin.net/v1.0/agents/{id}/direct-hash-regenerate``           | :ref:`agent-overview`                     |
+---------------+----------------+------------------------------------------------------------------------------------+-------------------------------------------+
| AI            | No             | ``POST https://api.voipbin.net/v1.0/ais/{id}/direct-hash-regenerate``              | :ref:`ai-overview`                        |
+---------------+----------------+------------------------------------------------------------------------------------+-------------------------------------------+

**Auto-Created** means the ``direct_hash`` field is populated automatically when the resource is created. For resources marked **No**, call the regenerate endpoint to create the initial hash.


Managing Direct Hashes
----------------------

**Creating a Direct Hash**

For extensions, conferences, teams, queues, and flows, a direct hash is generated automatically when the resource is created. For agents and AIs, call the regenerate endpoint to create one:

.. code::

    $ curl -k --location --request POST \
        'https://api.voipbin.net/v1.0/agents/<agent-id>/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

The response contains the full resource with the ``direct_hash`` field populated.

**Regenerating a Direct Hash**

To invalidate the current hash and generate a new one, call the same regenerate endpoint. The old hash is permanently invalidated — any SIP URIs using the old hash will stop working immediately.

.. code::

    $ curl -k --location --request POST \
        'https://api.voipbin.net/v1.0/extensions/<extension-id>/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

No request body is required. The response contains the updated resource with the new ``direct_hash``.

.. note:: **AI Implementation Hint**

   After regenerating a direct hash, update any stored SIP URIs that reference the old hash. The old hash is permanently invalidated — there is no way to restore it. If you manage multiple integrations pointing to the same resource, update all of them before relying on the new hash.


Use Cases
---------

- **External partner access**: Share a simple SIP address (``sip:direct.<hash>@sip.voipbin.net``) with partners or customers who need to reach your resources without configuring customer-specific domains.
- **SIP trunk compatibility**: Allow inbound calls from SIP trunks that cannot be configured with customer-specific domains. The shared ``sip.voipbin.net`` domain works universally.
- **AI agent dial-in**: Provide a public SIP address for an AI voice agent so external callers can reach it directly.
- **Conference bridge access**: Share a direct hash SIP URI as the conference dial-in number for participants.
- **Queue dial-in**: Provide a public SIP address for a queue so external callers can join and wait for an available agent.
- **Flow testing**: Share a direct hash SIP URI for a flow to allow testing the flow's action sequence without configuring an inbound number.
- **Security rotation**: If a hash is compromised or shared unintentionally, regenerate it immediately. The old hash stops working and a new one is issued.


Security Considerations
-----------------------

- **Treat the hash as a credential.** Anyone with the direct hash SIP URI can initiate calls to the resource. Share it only with intended recipients.
- **Cryptographically random.** Hashes are generated using ``crypto/rand`` (6 bytes / 12 hex characters). They are not guessable or sequential.
- **Regenerate if compromised.** Call the regenerate endpoint to instantly invalidate the old hash and issue a new one. This is atomic — the old hash stops working immediately.
- **No expiration.** Direct hashes remain valid until explicitly regenerated or the resource is deleted.


Troubleshooting
---------------

* **404 when calling a direct hash SIP URI:**
    * **Cause:** The hash does not exist, was regenerated, or the resource was deleted.
    * **Fix:** Retrieve the current resource via ``GET https://api.voipbin.net/v1.0/<resources>/{id}`` and verify the ``direct_hash`` field matches the URI you are dialing.

* **Empty ``direct_hash`` field in resource response:**
    * **Cause:** The resource type does not auto-create direct hashes (agent, AI), and the regenerate endpoint has not been called.
    * **Fix:** Call ``POST https://api.voipbin.net/v1.0/<resources>/{id}/direct-hash-regenerate`` to create the initial hash.

* **Calls not reaching resource after regeneration:**
    * **Cause:** SIP devices or trunks are still configured with the old hash.
    * **Fix:** Update the SIP URI in all devices and trunk configurations to use the new ``direct_hash`` value.

* **404 on the regenerate endpoint itself:**
    * **Cause:** The resource UUID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET https://api.voipbin.net/v1.0/<resources>`` list call with your authentication token.


Related Documentation
---------------------

- :ref:`Extension Overview — Direct Extension <extension-overview-direct>` — Detailed direct extension architecture and SIP flow diagrams
- :doc:`Extension Tutorial <extension_tutorial>` — Extension CRUD and direct hash regeneration example
- :ref:`Agent Tutorial <agent-tutorial>` — Agent direct hash regeneration example
- :doc:`Conference Tutorial <conference_tutorial>` — Conference direct hash regeneration example
- :ref:`Queue Tutorial <queue-tutorial>` — Queue direct hash regeneration example
- :ref:`AI Tutorial <ai-tutorial>` — AI direct hash regeneration example
- :ref:`Team Tutorial <team-tutorial>` — Team direct hash regeneration example
- :ref:`Flow Tutorial <flow-tutorial-basic>` — Flow CRUD and direct hash regeneration example
