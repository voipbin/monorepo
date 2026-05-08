.. _outbound_config_struct_outbound_config:

OutboundConfig Struct
=====================

.. code::

   {
     "id": "<uuid>",
     "customer_id": "<uuid>",
     "name": "<string>",
     "detail": "<string>",
     "destination_whitelist": ["<iso-alpha2>", ...],
     "codecs": "<string>",
     "default_outgoing_source_number_id": "<uuid>",
     "tm_create": "<RFC3339 | null>",
     "tm_update": "<RFC3339 | null>",
     "tm_delete": "<RFC3339 | null>"
   }

Fields
------

* ``id`` (UUID): Server-generated unique identifier. Obtained from ``POST /v1/outbound_configs`` or ``GET /v1/outbound_configs``.
* ``customer_id`` (UUID): The customer this config belongs to. Obtained from ``GET https://api.voipbin.net/v1.0/customer``.
* ``name`` (String, Optional): Human-readable label.
* ``detail`` (String, Optional): Extended description.
* ``destination_whitelist`` (Array of String): ISO 3166 alpha-2 country codes (lowercase, e.g. ``us``, ``gb``, ``kr``). Empty array = deny all outbound PSTN calls. Always enforced — must be populated before placing PSTN calls.
* ``codecs`` (String): Comma-separated codec preference list applied to **SIP outgoing calls** (e.g. ``PCMU,PCMA,G729``). Empty string = server default codec negotiation. **Not applied to PSTN calls** — PSTN trunks negotiate codecs with the carrier via SDP.
* ``default_outgoing_source_number_id`` (UUID, Optional): Fallback source number used when an outgoing call is placed without a valid source the customer owns. Obtained from the ``id`` field of ``GET https://api.voipbin.net/v1.0/numbers`` (must be a normal active number owned by the same customer). Set to ``00000000-0000-0000-0000-000000000000`` to disable the fallback (calls without a valid customer-owned source will be rejected).
* ``tm_create`` (ISO 8601 / null): Creation timestamp.
* ``tm_update`` (ISO 8601 / null): Last update timestamp.
* ``tm_delete`` (ISO 8601 / null): Soft-deletion timestamp. ``null`` for active records.
