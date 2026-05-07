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
* ``codecs`` (String): Comma-separated codec preference list (e.g. ``PCMU,PCMA,G729``). Empty string = server default codec negotiation.
* ``tm_create`` (ISO 8601 / null): Creation timestamp.
* ``tm_update`` (ISO 8601 / null): Last update timestamp.
* ``tm_delete`` (ISO 8601 / null): Soft-deletion timestamp. ``null`` for active records.
