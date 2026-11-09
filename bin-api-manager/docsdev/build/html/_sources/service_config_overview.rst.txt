.. _service-config-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. These endpoints change settings only. They send no email.
   * **Async:** No. Every request returns the updated key immediately.
   * **Access:** ProjectSuperAdmin only. Customer-level tokens, access keys and direct tokens are refused.

The Service Config API manages the stored configuration of a platform service. The ``{service}`` path parameter names the service. ``email`` is the only service that is supported now. Its keys are ``mailgun.api_key``, ``sendgrid.api_key`` and ``mailgun.webhook_signing_key``.

With the Service Config API you can:

- List the keys of a service, where each value comes from, and what each key is for
- Store a secret (write-only)
- Delete the stored secret to return to the environment of the service
- Read the history of the changes of a key

Only secrets can be stored or deleted now. A key that is not a secret is answered with 400.


Where a Value Comes From
------------------------
A service uses one source for a key at a time, in this order: the value stored through the API, the environment of the service, the default of the key.

.. list-table::
   :header-rows: 1

   * - ``source``
     - Meaning
   * - ``db``
     - The value stored through this API is used.
   * - ``env``
     - No value is stored. The environment of the service is used. ``env_name`` is the name of the variable.
   * - ``default``
     - No value is stored and the environment has none. The default of the key is used.
   * - ``none``
     - The key has no value anywhere.
   * - ``db_undecryptable``
     - A secret is stored but cannot be decrypted (for example the encryption key was removed). The service does **not** fall back to the environment. Restore the key, store a new secret, or delete the stored secret.
   * - ``db_invalid``
     - The stored value does not fit the declaration of the key. The service does not fall back to the environment. Store a new value or delete the stored value.

``env_shadowed`` is true when the environment variable is set but a stored value takes precedence over it. The response never contains the value of the environment variable.

The secret is write-only. It is never returned, logged or sent in an event. A response tells only ``has_secret``, the ``key_version`` of the key that encrypted it, the ``revision`` and who changed it. The history of a secret holds the key versions, never a value.

Storing a secret needs the encryption key to be configured on the platform. ``secret_writable`` in ``GET /service_configs/{service}`` tells whether it is. Without it the request is answered with ``503 Service Unavailable``.


Revision
--------
Every change must carry the ``revision`` that the caller read from ``GET /service_configs/{service}``. When the stored revision is different (someone else changed the key in the meantime) the request is rejected with ``409 Conflict`` and the reason ``REVISION_MISMATCH``, and nothing is changed. Read the key again and decide again.

``0`` means that the key has no stored value. Deleting a key that has no stored value with ``revision`` 0 succeeds and changes nothing.

The revision starts at 1 again when a key is deleted and stored again, so a caller that still holds revision 1 from before the delete cannot tell the difference.


Secret Rules
------------
A secret is 1 to 2020 bytes long (multi-byte characters count per byte). A secret that starts or ends with white space, or that contains a line break, is rejected with ``400 Bad Request``; it is not trimmed. The request body is limited to 4 KB.

After a timeout, or an answer of ``500`` or ``503``, check the key with ``GET /service_configs/{service}`` before you send the secret again. The answer ``500`` with the reason ``CONFIG_WRITE_VERIFY_FAILED`` means that the change may have been stored.

The request is also refused with ``503 Service Unavailable`` while the email service has not been updated to a version that supports this API, so that a secret is never sent to a service that cannot handle it.


Errors
------
.. list-table::
   :header-rows: 1

   * - Status
     - Meaning
   * - 400
     - The request is invalid: a body that is not JSON, a missing ``secret`` or ``revision``, a ``revision`` outside 0 to 4294967295, a secret that is empty, too long, has surrounding white space or a line break, a key that is not a secret, or a key name that does not match ``^[a-z0-9][a-z0-9._]{0,127}$``.
   * - 401
     - The request is not authenticated.
   * - 403
     - The caller is not a ProjectSuperAdmin, or uses a direct token.
   * - 404
     - The service or the key is unknown.
   * - 409
     - The ``revision`` is not the stored one (``REVISION_MISMATCH``).
   * - 503
     - The platform has no secret encryption key (``ENCRYPTION_KEY_NOT_CONFIGURED``), or the service does not support the API yet.

The response of an error does not say which rule failed for a 400, so check the list above.
