.. _outbound_config_tutorial:

OutboundConfig Tutorial
=======================

Prerequisites
+++++++++++++

* A valid JWT token for authentication. The API resolves your OutboundConfig automatically from your JWT — no customer ID or config ID is required for self-service endpoints.

Step 1 — Retrieve your OutboundConfig
++++++++++++++++++++++++++++++++++++++

.. note:: **AI Implementation Hint**

   An OutboundConfig is **automatically created** for every new customer account.
   Use ``GET https://api.voipbin.net/v1.0/outbound_config`` (singular, no ID) to retrieve it.
   The config is resolved from the JWT — no customer ID or config ID is needed.

.. code::

   GET https://api.voipbin.net/v1.0/outbound_config
   Authorization: Bearer <token>

Response:

.. code::

   {
     "id": "a1b2c3d4-...",
     "customer_id": "...",
     "name": "",
     "destination_whitelist": [],
     "codecs": "",
     "tm_create": "2026-05-07T10:00:00Z",
     "tm_update": null,
     "tm_delete": null
   }

Step 2 — Populate the destination whitelist and codec preferences
++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

Use ``PUT /v1.0/outbound_config`` (singular, no ID) to update your OutboundConfig. The API resolves which config to update from your JWT.

.. note:: **AI Implementation Hint**

   ``codecs`` applies to **SIP outgoing calls only**. PSTN (telephone) calls negotiate codecs directly with the carrier trunk via SDP — setting ``codecs`` has no effect on PSTN calls.

.. code::

   PUT https://api.voipbin.net/v1.0/outbound_config
   Authorization: Bearer <token>
   Content-Type: application/json

   {
     "destination_whitelist": ["us", "gb", "kr"],
     "codecs": "PCMU,PCMA"
   }

Response:

.. code::

   {
     "id": "a1b2c3d4-...",
     "customer_id": "...",
     "name": "",
     "destination_whitelist": ["us", "gb", "kr"],
     "codecs": "PCMU,PCMA",
     "tm_create": "2026-05-07T10:00:00Z",
     "tm_update": "2026-05-07T10:05:00Z",
     "tm_delete": null
   }

Step 3 — Add a country to the whitelist
++++++++++++++++++++++++++++++++++++++++

Partial update — only ``destination_whitelist`` changes; other fields are unchanged.

.. code::

   PUT https://api.voipbin.net/v1.0/outbound_config
   Authorization: Bearer <token>
   Content-Type: application/json

   {
     "destination_whitelist": ["us", "gb", "kr", "de"]
   }

.. note:: **AI Implementation Hint**

   To add a single country without losing existing entries: first ``GET https://api.voipbin.net/v1.0/outbound_config``
   to retrieve the current ``destination_whitelist``, append the new country code (ISO 3166 alpha-2, lowercase),
   then ``PUT`` the updated list.

Step 4 — Verify
+++++++++++++++

.. code::

   GET https://api.voipbin.net/v1.0/outbound_config
   Authorization: Bearer <token>

Admin-Only Operations
+++++++++++++++++++++

The following operations require ``PermissionProjectSuperAdmin`` and are not available to normal customers:

* ``GET https://api.voipbin.net/v1.0/outbound_configs`` — List all OutboundConfigs across all customers.
* ``GET https://api.voipbin.net/v1.0/outbound_configs/{id}`` — Get a specific OutboundConfig by UUID.
* ``PUT https://api.voipbin.net/v1.0/outbound_configs/{id}`` — Update a specific OutboundConfig by UUID.
* ``DELETE https://api.voipbin.net/v1.0/outbound_configs/{id}`` — Soft-delete a specific OutboundConfig by UUID. After deletion all outbound PSTN calls for that customer are blocked.
* ``POST https://api.voipbin.net/v1.0/outbound_configs`` — Create an OutboundConfig for a specific customer (for cases where auto-create did not fire).

.. note:: **AI Implementation Hint**

   Use the singular ``GET /v1.0/outbound_config`` endpoint in user-facing documentation and AI agent flows.
   Only use the plural ``/v1.0/outbound_configs`` endpoints in admin tooling.
