.. _service-config-struct-service-config:

Service Config
==============

.. _service-config-struct-service-config-service:

Service Config
--------------

.. code::

    {
        "service": "<string>",
        "secret_writable": <boolean>,
        "keys": [
            <Key>
        ]
    }

* ``service`` (String): The name of the service: ``email``, ``ai``, ``message``, ``tts``, ``transcribe``, ``billing`` or ``pipecat``.
* ``secret_writable`` (Boolean): Whether the platform has an encryption key, which a secret needs.
* ``keys`` (Array of Key): The keys of the service, in the order the service declares them.

.. _service-config-struct-service-config-key:

Key
---

.. code::

    {
        "key": "<string>",
        "kind": "<string>",
        "type": "<string>",
        "description": "<string>",
        "source": "<string>",
        "has_secret": <boolean>,
        "env_name": "<string>",
        "env_shadowed": <boolean>,
        "key_version": <integer>,
        "revision": <integer>,
        "updated_by": "<string>",
        "tm_update": "<string>"
    }

* ``key`` (String): The name of the key, such as ``mailgun.api_key``. Returned from ``GET /service_configs/{service}``.
* ``kind`` (enum string): ``secret``, ``param`` or ``select``. A ``secret`` is encrypted and write-only.
* ``type`` (enum string): The value type: ``string``, ``int``, ``bool``, ``duration`` or ``enum``.
* ``description`` (String): One line that tells what the key is for.
* ``source`` (enum string): Where the value comes from. See :ref:`Where a Value Comes From <service-config-overview>`.
* ``has_secret`` (Boolean): Whether a secret value exists. The secret itself is never returned.
* ``env_name`` (String): The name of the environment variable that is read when no value is stored. The name only, never the value. Omitted when the key has none (for example the keys of the ai, email, message, tts, transcribe, billing and pipecat services).
* ``env_shadowed`` (Boolean): Whether the environment variable is set but not used, because a stored value takes precedence. Always false for a key that has no environment variable.
* ``key_version`` (Integer): The version of the encryption key that encrypted the stored secret. 0 when none is stored.
* ``revision`` (Integer): The revision of the stored value, to be sent back when the key is changed. 0 when no value is stored.
* ``updated_by`` (UUID): The agent who last changed the stored value. Returned from ``GET /agents``. Omitted when no value is stored.
* ``tm_update`` (String, ISO 8601): The time of the last change of the stored value. Omitted when no value is stored.

Example

.. code::

    {
        "key": "mailgun.api_key",
        "kind": "secret",
        "type": "string",
        "description": "API key of the Mailgun account that sends email.",
        "source": "db",
        "has_secret": true,
        "env_shadowed": false,
        "key_version": 1,
        "revision": 3,
        "updated_by": "550e8400-e29b-41d4-a716-446655440000",
        "tm_update": "2026-01-02T03:04:05.000000Z"
    }

.. _service-config-struct-service-config-history:

History
-------

.. code::

    {
        "id": "<string>",
        "key": "<string>",
        "action": "<string>",
        "is_secret": <boolean>,
        "old_key_version": <integer>,
        "new_key_version": <integer>,
        "actor_id": "<string>",
        "source": "<string>",
        "tm_create": "<string>"
    }

* ``id`` (UUID): The ID of the history entry. Returned from ``GET /service_configs/{service}/keys/{key}/histories``.
* ``key`` (String): The name of the key.
* ``action`` (enum string): ``set`` stores a value, ``clear`` deletes the stored value.
* ``is_secret`` (Boolean): Whether the key is a secret. The history of a secret never holds a value.
* ``old_key_version`` (Integer): The encryption key version before the change. 0 when there was none.
* ``new_key_version`` (Integer): The encryption key version after the change. 0 when the value was deleted.
* ``actor_id`` (UUID): The agent who made the change. Returned from ``GET /agents``.
* ``source`` (enum string): ``api`` for this API, ``cli`` for the command line tool of the service.
* ``tm_create`` (String, ISO 8601): The time of the change.

The list response is paginated: ``result`` holds the entries, the newest first, and ``next_page_token`` is the cursor for the next page (empty on the last page). Entries made at the same moment can appear in either order.
