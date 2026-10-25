.. _number-provider-struct-number-provider:

Number Provider
===============

.. _number-provider-struct-number-provider-provider:

Number Provider
---------------

.. code::

    {
        "provider_name": "<string>",
        "enabled": <boolean>,
        "priority": <integer>,
        "config": {
            "<string>": "<string>"
        },
        "has_secret": <boolean>,
        "key_version": <integer>,
        "credential_source": "<string>",
        "usable": <boolean>,
        "updated_by": "<string>",
        "secret_updated_by": "<string>",
        "tm_update": "<string>",
        "tm_secret_update": "<string>"
    }

* ``provider_name`` (String): The name of the carrier, such as ``telnyx``. Returned from ``GET /number_providers``.
* ``enabled`` (Boolean): Whether the carrier takes part in number purchase.
* ``priority`` (Integer): The purchase order. A lower value is tried first. Ties are ordered by ``provider_name``.
* ``config`` (Object): The non-secret settings of the carrier, such as the Telnyx ``connection_id`` and ``profile_id``. Omitted when empty. The ``config`` is used only while a secret is stored (``credential_source`` is ``db``).
* ``has_secret`` (Boolean): Whether a secret is stored through the API. The secret itself is never returned.
* ``key_version`` (Integer): The version of the encryption key that encrypted the stored secret. 0 when no secret is stored.
* ``credential_source`` (enum string): Where the credentials come from. See :ref:`Credential Source <number-provider-struct-number-provider-credential-source>`.
* ``usable`` (Boolean): Whether the carrier can currently serve: it is enabled, has a credential, and (for a stored secret) has the ``config`` keys it requires. The flag does not mean the carrier can purchase: ``twilio`` cannot purchase numbers and is never used for purchase.
* ``updated_by`` (UUID): The agent who last changed ``enabled``, ``priority`` or ``config``. Returned from ``GET /agents``. Omitted when never changed.
* ``secret_updated_by`` (UUID): The agent who last changed the secret. Returned from ``GET /agents``. Omitted when never changed.
* ``tm_update`` (String, ISO 8601): The time of the last change of the setting, including a secret change. Omitted when never changed.
* ``tm_secret_update`` (String, ISO 8601): The time of the last secret change. Omitted when never changed.

Example

.. code::

    {
        "provider_name": "telnyx",
        "enabled": true,
        "priority": 10,
        "config": {
            "connection_id": "1234567890123456789"
        },
        "has_secret": true,
        "key_version": 1,
        "credential_source": "db",
        "usable": true,
        "updated_by": "550e8400-e29b-41d4-a716-446655440000",
        "secret_updated_by": "00000000-0000-4000-8000-000000000002",
        "tm_update": "2026-01-02T03:04:05.000000Z",
        "tm_secret_update": "2026-01-03T04:05:06.000000Z"
    }

.. _number-provider-struct-number-provider-credential-source:

Credential Source
-----------------
.. list-table::
   :header-rows: 1

   * - Value
     - Description
   * - ``db``
     - The secret stored through ``PUT /number_providers/{provider_name}/secret`` is used, with the ``config`` of the provider.
   * - ``env``
     - No secret is stored. The credentials of the platform environment are used.
   * - ``none``
     - No credential exists.
   * - ``db_undecryptable``
     - A secret is stored but cannot be decrypted. The provider is unusable until the key is restored or the stored secret is deleted.

Update request
--------------

.. code::

    {
        "enabled": <boolean>,
        "priority": <integer>,
        "config": {
            "<string>": "<string>"
        }
    }

* ``enabled`` (Boolean, optional): Whether the carrier takes part in number purchase.
* ``priority`` (Integer, optional): 0 to 1000000. A lower value is tried first.
* ``config`` (Object, optional): Merged key by key. A key with an empty string value is removed. At most 8 keys, a key up to 32 bytes and a value up to 256 bytes. Telnyx accepts ``connection_id`` and ``profile_id``; Twilio accepts ``sid``.

At least one field is required. Omitted fields keep their current value.

Secret request
--------------

.. code::

    {
        "secret": "<string>"
    }

* ``secret`` (String, required): The carrier secret, such as an API token. Surrounding whitespace is trimmed. It must be 1 to 1024 bytes long after trimming. It is write-only and never returned.
