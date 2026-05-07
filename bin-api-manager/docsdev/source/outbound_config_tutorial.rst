.. _outbound_config_tutorial:

OutboundConfig Tutorial
=======================

Prerequisites
+++++++++++++

* Your customer ID (UUID). Obtained from ``GET https://api.voipbin.net/v1.0/customer``.
* A valid JWT token for authentication.

Step 1 — Create your OutboundConfig
++++++++++++++++++++++++++++++++++++

.. code::

   POST https://api.voipbin.net/v1.0/outbound_configs
   Authorization: Bearer <token>
   Content-Type: application/json

   {
     "name": "production",
     "destination_whitelist": ["us", "gb", "kr"],
     "codecs": "PCMU,PCMA"
   }

Response:

.. code::

   {
     "id": "a1b2c3d4-...",           // Save this as outbound_config_id
     "customer_id": "...",
     "name": "production",
     "destination_whitelist": ["us", "gb", "kr"],
     "codecs": "PCMU,PCMA",
     "tm_create": "2026-05-07T10:00:00Z",
     "tm_update": "2026-05-07T10:00:00Z",
     "tm_delete": null
   }

Step 2 — Add a country to the whitelist
++++++++++++++++++++++++++++++++++++++++

Partial update — only ``destination_whitelist`` changes; other fields are unchanged.

.. code::

   PUT https://api.voipbin.net/v1.0/outbound_configs/{outbound_config_id}
   Authorization: Bearer <token>
   Content-Type: application/json

   {
     "destination_whitelist": ["us", "gb", "kr", "de"]
   }

.. note:: **AI Implementation Hint**

   To add a single country without losing existing entries: first ``GET /v1/outbound_configs/{id}`` to retrieve the current list, append the new country, then ``PUT`` the updated list.

Step 3 — Verify
+++++++++++++++

.. code::

   GET https://api.voipbin.net/v1.0/outbound_configs/{outbound_config_id}
   Authorization: Bearer <token>
