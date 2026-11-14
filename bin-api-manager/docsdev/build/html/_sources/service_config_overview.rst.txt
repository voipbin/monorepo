.. _service-config-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. These endpoints change settings only. They send no email.
   * **Async:** No. Every request returns the updated key immediately.
   * **Access:** ProjectSuperAdmin only. Customer-level tokens, access keys and direct tokens are refused.

The Service Config API manages the stored configuration of a platform service.

.. note:: **Stored platform secrets are not used now**

   The platform secrets of the services listed below (provider keys, the AWS credential, the Paddle keys, the voice AI vendor keys and the MCP OAuth app secrets) are read from environment variables of the service, and only from there. A value that is stored through this API stays accepted, listed and recorded in the history, but no service reads it. To change a platform secret, change its environment variable and restart the service (see Provider configuration in the self-hosting guide for the variable names). The ``config set`` and ``clear`` commands of the service print the variable that is read. Customer-owned credentials (the engine key and the engine base key of an AI, the secrets of an MCP server and OAuth tokens) are not part of this API and are not affected.

The ``{service}`` path parameter names the service. Seven services are supported now, and the keys below are the stored settings; they are not used by the services (see the note above). ``email`` has the keys ``mailgun.api_key``, ``sendgrid.api_key`` and ``mailgun.webhook_signing_key``. ``ai`` has the keys ``llm.openai_compat.api_key`` (the API key of the OpenAI compatible provider) and ``llm.gemini.api_key`` (the Google API key of the Gemini endpoint). ``message`` has the keys ``provider.messagebird.auth_token`` (the access key of the MessageBird account) and ``provider.telnyx.auth_token`` (the API key of the Telnyx account that sends the SMS and MMS). ``tts`` has the keys ``aws.access_key`` and ``aws.secret_key`` (the access key and the secret key of Amazon Polly) and ``elevenlabs.api_key`` (the API key of the ElevenLabs streaming service).  ``transcribe`` has the keys ``aws.access_key`` and ``aws.secret_key`` (the access key and the secret access key of the AWS credential that the AWS Transcribe provider uses). ``billing`` has the keys ``paddle.api_key`` (the API key of the Paddle account) and ``paddle.webhook_secret_key`` (the secret that signs the Paddle webhooks). ``pipecat`` has the platform keys of the voice AI vendors: ``openai.api_key``, ``deepgram.api_key``, ``cartesia.api_key``, ``xai.api_key``, ``elevenlabs.api_key``, ``google.api_key`` (a Gemini API key only), ``openrouter.api_key`` and ``local_llm.endpoints`` (a one-line JSON object that maps the name of a local endpoint to its ``base_url`` and ``api_key``). A voice AI session that uses a key of the customer does not use the platform key of the same vendor. ``ai`` also has the keys ``mcp_oauth.<vendor>.client_secret`` for the OAuth apps of the MCP vendors (``github``, ``linear``, ``asana`` and ``hubspot``); the matching ``client_id`` is not a secret, so it is set only with the ``config set`` command of ``ai-control``, and it must be set before the secret. The OAuth app of a vendor is read from the variables ``GITHUB_MCP_OAUTH_CLIENT_ID`` and ``GITHUB_MCP_OAUTH_CLIENT_SECRET`` (and the same two for ``LINEAR``, ``ASANA`` and ``HUBSPOT``). A service name that is not listed is answered with ``404 Not Found``.

With the Service Config API you can:

- List the keys of a service, where each value comes from, and what each key is for
- Store a secret (write-only)
- Delete the stored secret; the key is then not configured
- Read the history of the changes of a key

Only secrets can be stored or deleted now. A key that is not a secret is answered with 400.


Where a Value Comes From
------------------------
A key of a service can have a value stored through this API, an environment variable, or a default. All keys of ``ai`` (the two ``llm`` keys and the ``mcp_oauth`` keys), ``email``, ``message``, ``tts``, ``transcribe``, ``billing`` and ``pipecat`` are platform secrets: the service reads the environment variable (``OPENAI_API_KEY`` for ``llm.openai_compat.api_key``, ``GOOGLE_API_KEY`` for ``llm.gemini.api_key``, and so on) and does not read the stored value. The ``source`` that this API shows for such a key describes the stored value only (``db`` when a value is stored, otherwise ``none``); it does not say which value a service uses, and the response never shows the environment variable. ``env_name`` is not returned for these keys, and ``env_shadowed`` is always false.

.. list-table::
   :header-rows: 1

   * - ``source``
     - Meaning
   * - ``db``
     - The value stored through this API is used.
   * - ``env``
     - No value is stored and the environment of the service is used. Only a key that has an environment variable can use this source, and ``env_name`` is returned for such a key. No key of the supported services has one in this API now, so a response does not show this source. The other keys never have this source, and ``env_name`` is not returned for them. The process that answers sees its own environment, so ``env`` can show ``none`` for a key that a different process of the platform reads from its environment.
   * - ``default``
     - No value is stored. The default of the key is used.
   * - ``none``
     - The key has no value anywhere.
   * - ``db_undecryptable``
     - A secret is stored but cannot be decrypted (for example the encryption key was removed). The service does **not** use an empty value instead. Restore the key, store a new secret, or delete the stored secret.
   * - ``db_invalid``
     - The stored value does not fit the declaration of the key. The service does not use an empty value instead. Store a new value or delete the stored value.

``env_shadowed`` is true only when a service reads an environment variable for the key and a stored value takes precedence over it. It is always false for the keys that have no environment variable. The response never contains the value of an environment variable.

The secret is write-only. It is never returned, logged or sent in an event by a service version that supports this API (see Secret Rules). A response tells only ``has_secret``, the ``key_version`` of the key that encrypted it, the ``revision`` and who changed it. The history of a secret holds the key versions, never a value.

Storing a secret needs the encryption key to be configured on the platform. ``secret_writable`` in ``GET /service_configs/{service}`` tells whether it is. Without it the request is answered with ``503 Service Unavailable``.


Revision
--------
Every change must carry the ``revision`` that the caller read from ``GET /service_configs/{service}``. When the stored revision is different (someone else changed the key in the meantime) the request is rejected with ``409 Conflict`` and the reason ``REVISION_MISMATCH``, and nothing is changed. Read the key again and decide again.

``0`` means that the key has no stored value. Deleting a key that has no stored value with ``revision`` 0 succeeds and changes nothing.

The revision starts at 1 again when a key is deleted and stored again, so a caller that still holds revision 1 from before the delete cannot tell the difference.


Secret Rules
------------
A secret is 1 to 2020 bytes long (multi-byte characters count per byte). A key can declare a shorter limit: the keys of the email, ai, message, tts, transcribe, billing and pipecat services allow 512 bytes, except ``local_llm.endpoints`` of ``pipecat``, which allows 2020 bytes. A longer secret is rejected with ``400 Bad Request``. The ``local_llm.endpoints`` value is one line of JSON without white space around it, for example ``{"local1":{"base_url":"https://llm.example.com/v1","api_key":"<key>"}}``, and the request body that carries it must stay within the 4 KB limit after JSON escaping; a body over the limit is rejected with ``400 Bad Request``. A secret that starts or ends with white space, or that contains a line break, is rejected with ``400 Bad Request``; it is not trimmed. The request body is limited to 4 KB, and a larger body is rejected with ``400 Bad Request``.

After a timeout, or an answer of ``500`` or ``503``, check the key with ``GET /service_configs/{service}`` before you send the secret again. The answer ``500`` with the reason ``CONFIG_WRITE_VERIFY_FAILED`` means that the change may have been stored, and any other ``500`` can leave the state unknown in the same way.

The request is also refused with ``503 Service Unavailable`` while the service has not been updated to a version that supports this API. This check lowers the chance that a secret reaches a service that cannot handle it during a rollout, but it does not rule it out: store a secret only after every instance of the service has been updated.


Errors
------
.. list-table::
   :header-rows: 1

   * - Status
     - Meaning
   * - 400
     - The request is invalid: a body that is not JSON, a missing ``secret`` or ``revision``, a ``revision`` outside 0 to 4294967295, a secret that is empty, too long, has surrounding white space or a line break, a key that is not a secret, a body larger than 4 KB, a ``page_size`` below 0, a ``page_token`` that was not returned for the key, or a key name that does not match ``^[a-z0-9][a-z0-9._]{0,127}$``.
   * - 401
     - The request is not authenticated.
   * - 403
     - The caller is not a ProjectSuperAdmin, or uses a direct token.
   * - 404
     - The service or the key is unknown.
   * - 409
     - The ``revision`` is not the stored one (``REVISION_MISMATCH``).
   * - 503
     - The platform has no secret encryption key (``ENCRYPTION_KEY_NOT_CONFIGURED``), the service does not support the API yet, or the service did not answer in time.

The default ``page_size`` of the history is 20 and the largest is 100. A larger value is lowered to 100.

The response of an error does not say which rule failed for a 400, so check the list above.
