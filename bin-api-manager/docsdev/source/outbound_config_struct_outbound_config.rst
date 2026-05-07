.. _outbound-config-struct:

Struct
======

.. _outbound-config-struct-outbound-config:

OutboundConfig
--------------

.. code::

   {
      "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
      "name": "Default Outbound Config",
      "detail": "Restrict calls to US and Canada only",
      "destination_whitelist": ["us", "ca"],
      "codecs": "ulaw,alaw",
      "tm_create": "2026-05-07T00:00:00.000000Z",
      "tm_update": "2026-05-07T00:00:00.000000Z",
      "tm_delete": "9999-01-01T00:00:00.000000Z"
   }

* ``id`` (UUID): The outbound config's unique identifier.
* ``customer_id`` (UUID): The customer that owns this outbound config.
* ``name`` (String): A human-readable name for the config.
* ``detail`` (String): An optional description of the config's purpose.
* ``destination_whitelist`` (Array of String): List of allowed destination country codes in ISO 3166 alpha-2 lowercase format (e.g., ``"us"``, ``"ca"``). An empty list means all destinations are allowed.
* ``codecs`` (String): Comma-separated list of allowed codecs (e.g., ``"ulaw,alaw"``). An empty string means the server default codec list is used.
* ``tm_create`` (String, ISO 8601): Timestamp when the config was created.
* ``tm_update`` (String, ISO 8601): Timestamp when the config was last updated.
* ``tm_delete`` (String, ISO 8601): Timestamp when the config was deleted, if applicable.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01T00:00:00.000000Z`` indicates the config has not been deleted and is still active. This sentinel value is used across all VoIPBIN resources to represent "not yet occurred."
