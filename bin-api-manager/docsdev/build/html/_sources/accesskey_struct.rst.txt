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
      "token": "DuRWq5T4DAK32dw4",
      "tm_expire": "2025-04-28 01:41:40.503790",
      "tm_create": "2022-04-28 01:41:40.503790",
      "tm_update": "2022-04-28 01:41:40.503790",
      "tm_delete": "9999-01-01 00:00:00.000000"
   }

* ``id`` (UUID): The accesskey's unique identifier. Returned when creating an accesskey via ``POST /accesskeys`` or when listing accesskeys via ``GET /accesskeys``.
* ``customer_id`` (UUID): The customer that owns this accesskey. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (String): An optional human-readable name for the accesskey. Useful for identification in multi-key environments.
* ``detail`` (String): An optional description of the accesskey's intended use or purpose.
* ``token`` (String): The API token credential used to authenticate requests. Must be stored securely and never exposed in client-side code.
* ``tm_expire`` (string, ISO 8601): Timestamp when the accesskey will expire. After this time, the key will no longer be valid for authentication.
* ``tm_create`` (string, ISO 8601): Timestamp when the accesskey was created.
* ``tm_update`` (string, ISO 8601): Timestamp when the accesskey was last updated.
* ``tm_delete`` (string, ISO 8601): Timestamp when the accesskey was deleted, if applicable.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` indicates the accesskey has not been deleted and is still active. This sentinel value is used across all VoIPBIN resources to represent "not yet occurred."
