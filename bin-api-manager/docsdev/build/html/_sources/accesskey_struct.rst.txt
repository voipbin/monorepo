.. _accesskey-struct:

Struct
======

.. _accesskey-struct-accesskey:

Accesskey
---------

.. code::

   {
      "id": "5f1f8f7e-9b3d-4c60-8465-b69e9f28b6db",
      "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
      "name": "My API Key",
      "detail": "For accessing reporting APIs",
      "token_prefix": "vb_a3Bf9xKm",
      "tm_expire": "2027-04-28T01:41:40.503790Z",
      "tm_create": "2026-04-28T01:41:40.503790Z",
      "tm_update": "2026-04-28T01:41:40.503790Z",
      "tm_delete": "9999-01-01T00:00:00.000000Z"
   }

* ``id`` (UUID): The accesskey's unique identifier. Returned when creating an accesskey via ``POST /accesskeys`` or when listing accesskeys via ``GET /accesskeys``.
* ``customer_id`` (UUID): The customer that owns this accesskey. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (String, Optional): An optional human-readable name for the accesskey. Useful for identification in multi-key environments.
* ``detail`` (String, Optional): An optional description of the accesskey's intended use or purpose.
* ``token`` (String, Optional): The full API token credential. **Only returned once at creation time** via ``POST /accesskeys``. Store it securely and immediately. If lost, delete the key and create a new one.
* ``token_prefix`` (String): A short prefix of the token (e.g., ``vb_a3Bf9xKm``) for identification. Always returned in ``GET`` responses.
* ``tm_expire`` (String, ISO 8601): Timestamp when the accesskey will expire. After this time, the key will no longer be valid for authentication.
* ``tm_create`` (String, ISO 8601): Timestamp when the accesskey was created.
* ``tm_update`` (String, ISO 8601): Timestamp when the accesskey was last updated.
* ``tm_delete`` (String, ISO 8601): Timestamp when the accesskey was deleted, if applicable.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01T00:00:00.000000Z`` indicates the accesskey has not been deleted and is still active. This sentinel value is used across all VoIPBIN resources to represent "not yet occurred."

.. note:: **AI Implementation Hint**

   The ``token`` field is only present in the response to ``POST /accesskeys`` (creation). All subsequent ``GET /accesskeys`` and ``GET /accesskeys/{id}`` responses will NOT include the ``token`` field. Use ``token_prefix`` to identify which key is which. If the token is lost, delete the key via ``DELETE /accesskeys/{id}`` and create a new one.
