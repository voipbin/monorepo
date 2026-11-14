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
* ``config`` (Object): The non-secret settings of the carrier, such as the Telnyx ``connection_id`` and ``profile_id``. Omitted when empty. This is the stored setting only; a value that the stored setting does not have can be filled from an environment variable when the provider is used (see the overview), and the filled value is not shown here.
* ``has_secret`` (Boolean): Whether the secret of the carrier is present in the environment of the platform. A secret stored through the API does not count. The secret itself is never returned.
* ``key_version`` (Integer): The version of the encryption key that encrypted the stored secret. It describes the stored row only, which is not used. 0 when no secret is stored.
* ``credential_source`` (enum string): Where the credentials come from. See :ref:`Credential Source <number-provider-struct-number-provider-credential-source>`.
* ``usable`` (Boolean): Whether the carrier can currently serve: it is enabled, has its secret in the environment, and has the ``config`` keys it requires (``twilio`` needs ``sid`` and ``trunk_sid``). A carrier that is usable and enabled takes part in the purchase.
* ``updated_by`` (UUID): The agent who last changed ``enabled``, ``priority`` or ``config``. Returned from ``GET /agents``. Omitted when never changed.
* ``secret_updated_by`` (UUID): The agent who last changed the stored secret (not used). Returned from ``GET /agents``. Omitted when never changed.
* ``tm_update`` (String, ISO 8601): The time of the last change of the setting, including a secret change. Omitted when never changed.
* ``tm_secret_update`` (String, ISO 8601): The time of the last change of the stored secret (not used). Omitted when never changed.

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
        "credential_source": "env",
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
   * - ``env``
     - The secret variable of the carrier (``TELNYX_API_KEY`` or ``TWILIO_AUTH_TOKEN``) is set on the platform and is used, with the ``config`` of the provider and the variables that fill a missing ``config`` key.
   * - ``none``
     - The secret variable is not set (or is empty or an unresolved placeholder). The carrier is not usable until it is set.
   * - ``db``
     - Not returned now. A stored secret is not read.
   * - ``db_undecryptable``
     - Not returned now. A stored secret is not read, so it cannot fail to decrypt.

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
* ``config`` (Object, optional): Merged key by key. A key with an empty string value is removed. At most 8 keys, a key up to 32 bytes and a value up to 256 bytes. Telnyx accepts ``connection_id`` and ``profile_id``; Twilio accepts ``sid`` (the account SID) and ``trunk_sid`` (the Elastic SIP Trunk that receives the calls of the purchased numbers); both are required before a secret can be stored through the API.

At least one field is required. Omitted fields keep their current value.

Secret request
--------------

.. code::

    {
        "secret": "<string>"
    }

* ``secret`` (String, required): The carrier secret, such as an API token. Surrounding whitespace is trimmed. It must be 1 to 1024 bytes long after trimming. It is write-only and never returned. The stored secret is not used: the platform reads the secret from its environment variable (see the overview).
