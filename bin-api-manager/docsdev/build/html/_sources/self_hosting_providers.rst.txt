Provider configuration
=========================

All third-party providers are optional. Until a provider's keys are set,
the corresponding flow actions return a "provider not configured" error;
the rest of the platform stays healthy. This page lists every provider
env var in ``.env``, what it enables, and which service to restart after
setting it. All facts below are verified directly against this
installer's own ``docker-compose.yml.dist`` (the Docker Compose
self-hosting stack); a Kubernetes/hosted-service deployment wires some
of these variables differently and is out of scope for this page.

Platform provider keys are environment variables
------------------------------------------------------

The platform keys of the providers below are read from environment
variables of the service container, and only from there. A change needs a
restart of the service (a restart of ``pipecat-script-runner`` for the
``pipecat`` keys). An unset variable, an empty variable, or a value that is
still an unresolved placeholder (a text that starts with ``[[``) means that
the key is not configured: the service does not stop, and a call that needs
the key fails with the provider error. Each service logs the state of its
variables once at start (the variable name only, never a value).

.. list-table::
   :header-rows: 1
   :widths: 30 45 25

   * - Variable
     - Used for
     - Services
   * - ``OPENAI_API_KEY``
     - OpenAI key (the platform default key of the LLM calls and the real-time voice pipeline)
     - ``ai-manager``, ``pipecat-script-runner``
   * - ``GOOGLE_API_KEY``
     - Gemini API key
     - ``ai-manager``, ``pipecat-script-runner``
   * - ``XAI_API_KEY``, ``OPENROUTER_API_KEY``, ``DEEPGRAM_API_KEY``, ``CARTESIA_API_KEY``
     - xAI, OpenRouter, Deepgram and Cartesia keys
     - ``pipecat-script-runner``
   * - ``LOCAL_LLM_ENDPOINTS``
     - One line of JSON that maps a local endpoint name to its ``base_url`` and ``api_key``
     - ``pipecat-script-runner``
   * - ``ELEVENLABS_API_KEY``
     - ElevenLabs key
     - ``tts-manager``, ``pipecat-script-runner``
   * - ``AWS_ACCESS_KEY``, ``AWS_SECRET_KEY``
     - AWS credential (Polly and Transcribe)
     - ``tts-manager``, ``transcribe-manager``
   * - ``SENDGRID_API_KEY``, ``MAILGUN_API_KEY``, ``MAILGUN_WEBHOOK_SIGNING_KEY``
     - Email providers
     - ``email-manager``
   * - ``MESSAGEBIRD_API_KEY``, ``TELNYX_API_KEY``
     - SMS and MMS providers
     - ``message-manager``
   * - ``TELNYX_API_KEY``, ``TELNYX_CONNECTION_ID``, ``TELNYX_PROFILE_ID``, ``TWILIO_ACCOUNT_SID``, ``TWILIO_AUTH_TOKEN``
     - Number providers (see the telephony table below)
     - ``number-manager``
   * - ``PADDLE_API_KEY``, ``PADDLE_WEBHOOK_SECRET``
     - Paddle account
     - ``billing-manager``

A variable that two services use has one name and must hold the same value
for both (for example ``OPENAI_API_KEY`` of ``ai-manager`` and of
``pipecat-script-runner``, or ``TELNYX_API_KEY`` of ``message-manager`` and
``number-manager``). A rotation of the vendor account is one change of the
variable in the deployment, followed by a restart of the services that use it.

The AWS pair is used only together: when exactly one of the two variables is
set, the pair is refused (``tts-manager`` fails the request, and the start log
of ``tts-manager`` and ``transcribe-manager`` warns). Without
``PADDLE_WEBHOOK_SECRET`` every Paddle webhook is rejected, and without
``PADDLE_API_KEY`` the customer portal session of Paddle can not be created.
The ``pipecat-manager`` Go service reads no provider key; the runner reads
its variables from its own container and uses the key of a request first,
when the request carries one. A session that needs a key that is not set
fails (the runner answers ``<vendor> is not configured``).

The services read these keys from the environment only. The services have
no ``config`` command and no API that stores a provider key.

AI voice agent providers
----------------------------

.. list-table::
   :header-rows: 1
   :widths: 20 30 50

   * - Capability
     - Providers
     - Env var(s)
   * - LLM / conversation (batch, e.g. audit evaluation)
     - OpenAI
     - ``OPENAI_API_KEY`` (restart ``ai-manager``)
   * - LLM / conversation (real-time voice pipeline)
     - OpenAI, Google Gemini, xAI
     - ``OPENAI_API_KEY``, ``GOOGLE_API_KEY`` and ``XAI_API_KEY``
       (restart ``pipecat-script-runner``)
   * - LLM / conversation (real-time voice pipeline, platform-managed models)
     - OpenRouter (serves the Anthropic Claude, Meta Llama, DeepSeek, Qwen,
       and Mistral models that ``GET /ai_models`` lists with
       ``platform_managed`` set to ``true``)
     - ``OPENROUTER_API_KEY`` (restart ``pipecat-script-runner``)
   * - LLM / conversation (real-time voice pipeline, local models)
     - Self-hosted OpenAI-compatible servers (vLLM, llama.cpp, Ollama, and similar)
     - ``LOCAL_LLM_MODELS`` (restart ``ai-manager`` and ``pipecat-manager``),
       ``LOCAL_LLM_ENDPOINTS`` (restart ``pipecat-script-runner``)
   * - Speech-to-text
     - Deepgram, AWS Transcribe, Google Cloud Speech-to-Text
     - ``DEEPGRAM_API_KEY`` (restart ``pipecat-script-runner``);
       ``AWS_ACCESS_KEY`` and ``AWS_SECRET_KEY`` (restart
       ``transcribe-manager``); ``GOOGLE_APPLICATION_CREDENTIALS``
       (restart ``transcribe-manager``)
   * - Text-to-speech
     - ElevenLabs, Cartesia, AWS Polly, Google Cloud TTS
     - ``CARTESIA_API_KEY`` (restart ``pipecat-script-runner``);
       ``ELEVENLABS_API_KEY``, ``AWS_ACCESS_KEY`` and ``AWS_SECRET_KEY``
       (restart ``tts-manager``); ``GOOGLE_APPLICATION_CREDENTIALS``
       (restart ``tts-manager``)

``GOOGLE_API_KEY`` and ``GOOGLE_APPLICATION_CREDENTIALS`` are two
unrelated settings that happen to share the "Google" prefix. Do not
confuse them:

- ``GOOGLE_API_KEY`` is a Gemini API key. The real-time voice pipeline
  (``pipecat-script-runner``) uses it for live Gemini conversations, and
  ``ai-manager`` uses it for batch audit evaluation, prompt proposal and the
  Gemini based summary, analysis and builder calls. Both read the same
  variable name, so both must hold the same key.
- ``GOOGLE_APPLICATION_CREDENTIALS`` is a GCP service-account JSON key
  file path. It is mounted into **five** services in this installer's
  Docker Compose stack: ``api-manager``, ``rag-manager``,
  ``storage-manager``, ``transcribe-manager``, and ``tts-manager``.
  ``storage-manager`` treats a missing value as fatal for signed
  download/upload URLs; ``tts-manager`` falls back to AWS Polly if
  unset. ``pipecat-manager`` does **not** consume this variable in the
  Docker Compose stack (it only uses ``GOOGLE_API_KEY``, above) --
  restarting ``pipecat-manager`` for a ``GOOGLE_APPLICATION_CREDENTIALS``
  change has no effect.

``XAI_API_KEY`` has no dedicated feature write-up; it is
consumed by ``pipecat-manager``'s real-time voice pipeline as an LLM
alternative, not by ``ai-manager``.

OpenRouter key for platform-managed models
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Models that ``GET /ai_models`` reports with ``platform_managed`` set to
``true`` (Anthropic Claude, Meta Llama, DeepSeek, Qwen, and Mistral) do
not use the per-AI ``engine_key`` that a user enters. In the real-time
voice pipeline they are served through OpenRouter with a key that you,
the operator, supply once as the variable ``OPENROUTER_API_KEY`` of
``pipecat-script-runner``. Notes on how it behaves:

- ``pipecat-script-runner`` reads the variable when a session starts, so
  a changed value needs a restart of the runner container and is used from
  the next session. ``pipecat-manager`` (the Go service) reads no key.
- Without the variable these models stay selectable in the catalog (there
  is no global on/off switch), but a session that uses one fails when the
  runner initializes the LLM service, with the error
  ``OpenRouter is not configured``. Other models (OpenAI, Gemini, xAI)
  are not affected.
- A non-empty but invalid key is not detected at startup. The first
  request is rejected by OpenRouter with HTTP 401, which the runner
  classifies as an authentication error, the same as an invalid key for
  any other LLM provider.
- Requests are sent with OpenRouter's zero data retention routing and
  ``data_collection`` set to ``deny``, and only to endpoints that
  support tool calling. This is fixed and not configurable.
- ``pipecat-script-runner`` is pinned by image digest in
  ``docker-compose.yml.dist``. The variable has an effect only after a
  ``pipecat-script-runner`` image that reads the environment keys is adopted
  (for example with ``scripts/sync-compose-images.sh``).

Local models (self-hosted OpenAI-compatible servers)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

You can register models that run on your own server, such as vLLM,
llama.cpp or Ollama, as long as the server implements the OpenAI chat
completions API. Registered models appear in ``GET /ai_models`` with vendor
``Local`` and an ``id`` of ``local.<endpoint>.<model>``. Users only pick the
model; the server address and key are never shown. Private network
addresses and plain ``http`` are allowed, because the server is inside your
trust boundary. Nothing is listed until you set the settings below. There is
no separate on/off switch. This is not the same as the OpenRouter models above:
``platform_managed`` only says that you, the operator, supply the connection.

Two settings. Write each value as compact JSON (no space after a colon or
comma); write a ``$`` character as ``$$`` in the Docker Compose variable.

- ``LOCAL_LLM_ENDPOINTS`` (secret, a variable of
  ``pipecat-script-runner``): a JSON
  object that maps an endpoint name to its connection, for example
  ``{"vllm-a":{"base_url":"http://llm.internal:8000/v1","api_key":""}}``.
  The name is lower case letters, digits and ``-`` (up to 32 characters).
  ``base_url`` is ``http`` or ``https`` without user information, query or
  fragment. An empty ``api_key`` is allowed.
- ``LOCAL_LLM_MODELS`` (no secret, ``ai-manager`` and ``pipecat-manager``): a
  JSON list, for example
  ``[{"endpoint":"vllm-a","id":"qwen3-8b","label":"Qwen3 8B","capabilities":["tool-calling"],"default":false}]``.
  ``id`` may contain letters, digits and ``. _ : / -``. ``label`` is ASCII
  only (up to 64 characters). ``capabilities`` may contain ``tool-calling``
  and ``low-cost``; they are informational. At most one entry should set
  ``default`` to ``true``; it becomes the only ``recommended`` model. At most
  200 models are read. Invalid entries are skipped and logged as an error
  (counts only); the service still starts, so check the log after a change.
  The endpoint name, ``id`` and ``label`` are shown to users, so do not put
  internal host names in them.

To enable: set ``LOCAL_LLM_ENDPOINTS`` on the runner and ``LOCAL_LLM_MODELS``, redeploy
``pipecat-manager`` and ``pipecat-script-runner`` together (same image),
then redeploy ``ai-manager`` last. To remove a model: move the AIs that use it to another
model first, then redeploy ``ai-manager`` first and the pipecat services
afterwards. A session that uses a removed or unreachable model fails. Do not set
``OPENAI_ORG_ID`` or ``OPENAI_PROJECT_ID`` on the runner: the OpenAI client
would send them to your server. If the server cannot be reached, a voice session does
not tell the caller; the failure is recorded on the AI call as a
``pipeline_error`` message with category ``connection``.

Customer-specified endpoints (``customendpoint.`` models)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Users can also point an AI at their own OpenAI-compatible server by setting
``engine_model`` to ``customendpoint.<model-id>`` and ``engine_base_url``
through the API. The optional key for that server (``engine_base_key``) is
stored encrypted by ``ai-manager``. This needs one variable, which only
``ai-manager`` reads:

- ``ENGINE_BASE_KEY_ENCRYPTION_KEYS`` (secret): a comma separated list of
  ``<version>:<base64 32 bytes>`` entries, for example
  ``1:<base64 of 32 random bytes>`` or ``1:<key1>,2:<key2>``. The version is
  a number from 1 to 65535. The highest version encrypts new keys, and every
  listed version can decrypt. This is a separate key set from
  ``MCP_SECRET_ENCRYPTION_KEYS``.

Generate a key with ``openssl rand -base64 32`` and write it with its
version, for example ``ENGINE_BASE_KEY_ENCRYPTION_KEYS=1:<output>``. Keep the
value in your secret store. Then restart ``ai-manager``:

.. code-block:: bash

    sudo ./voipbin restart ai-manager           # ENGINE_BASE_KEY_ENCRYPTION_KEYS

Behavior to know about:

- A malformed value stops ``ai-manager`` from starting. An empty value lets
  it start with a warning, and a request that stores a key then fails with
  a server error. Endpoints without a key keep working.
- If you lose the key set, the stored endpoint keys can no longer be
  decrypted. This fails closed: those AIs are not sent any key and calls
  that need one fail, until each customer enters the key again. There is no
  other recovery.
- To rotate, add a new entry with a higher version to the list on every
  ``ai-manager`` replica first, restart all of them, and keep the old
  versions listed until no stored key uses them. A key is re-encrypted with
  the highest version when the customer sends it again. During a rotation a
  replica that does not yet have the new version cannot decrypt keys written
  with it, and calls that start in that period fail closed, so rotate in a
  quiet window.
- If you deploy from a separate install repository, add the same variable to
  its ``.env.template`` and pass it to the ``ai-manager`` service there too.
  A missing variable in that repository means the customers' endpoint keys
  cannot be stored.
- The platform connects to a customer's address only over ``https`` and only
  when it resolves to a public address, so a private or loopback address
  cannot be used for this feature. Use the ``local.`` models above for a
  server inside your own network. If the server cannot be reached, the AI
  records a ``pipeline_error`` message with category ``connection``.

After setting an AI provider key, restart the service(s) that actually
consume that specific variable, per the tables above:

.. code-block:: bash

    sudo ./voipbin restart pipecat-manager       # LOCAL_LLM_MODELS (restart ``pipecat-script-runner`` for the pipecat keys)
    sudo ./voipbin restart transcribe-manager    # GOOGLE_APPLICATION_CREDENTIALS
    sudo ./voipbin restart tts-manager           # GOOGLE_APPLICATION_CREDENTIALS
    sudo ./voipbin restart rag-manager           # GOOGLE_APPLICATION_CREDENTIALS
    sudo ./voipbin restart storage-manager       # GOOGLE_APPLICATION_CREDENTIALS
    sudo ./voipbin restart api-manager           # GOOGLE_APPLICATION_CREDENTIALS

Shared GCP storage and project variables
--------------------------------------------

Separately from ``GOOGLE_APPLICATION_CREDENTIALS`` above, four more
``.env.template`` variables use the ``GCP_`` prefix. Unlike every other
provider on this page, one service (``rag-manager``) treats two of them
as required at boot, not optional:

.. list-table::
   :header-rows: 1
   :widths: 20 35 45

   * - Var
     - Consuming service(s)
     - Behavior if unset
   * - ``GCP_PROJECT_ID``
     - ``api-manager``, ``rag-manager``, ``storage-manager``
     - ``rag-manager`` fails to start (hard validation error);
       ``api-manager`` and ``storage-manager`` degrade gracefully
   * - ``GCP_REGION``
     - ``rag-manager``
     - Fails to start (hard validation error)
   * - ``GCP_BUCKET_NAME_MEDIA``
     - ``asterisk-call-proxy``, ``asterisk-conference-proxy``,
       ``call-manager``, ``rag-manager``, ``storage-manager``
     - Optional for every consumer, including ``rag-manager`` itself;
       the three call-recording consumers have a working built-in
       default bucket name. ``rag-manager``'s boot failure below is
       driven only by ``GCP_PROJECT_ID``/``GCP_REGION``, not this var
   * - ``GCP_BUCKET_NAME_TMP``
     - ``api-manager``, ``storage-manager``
     - Both degrade gracefully

``rag-manager``'s boot failure is specific to its own startup
validation, not a property of these variables in general: the same
variables consumed by the other listed services do not crash those
services if left empty. ``init.sh`` writes non-functional placeholder
values (``sandbox-placeholder``, ``us-central1``,
``sandbox-placeholder-media``) that satisfy this startup check without
granting real GCP access, so ``rag-manager`` stays running, but
RAG ingestion and query calls fail against Vertex AI and Cloud Storage
until real values are set in ``.env``.

.. code-block:: bash

    sudo ./voipbin restart rag-manager           # GCP_PROJECT_ID, GCP_REGION, GCP_BUCKET_NAME_MEDIA
    sudo ./voipbin restart api-manager           # GCP_PROJECT_ID, GCP_BUCKET_NAME_TMP
    sudo ./voipbin restart storage-manager       # GCP_PROJECT_ID, GCP_BUCKET_NAME_MEDIA, GCP_BUCKET_NAME_TMP
    sudo ./voipbin restart asterisk-call         # GCP_BUCKET_NAME_MEDIA (also restarts its -proxy sidecar)
    sudo ./voipbin restart asterisk-conference   # GCP_BUCKET_NAME_MEDIA (also restarts its -proxy sidecar)
    sudo ./voipbin restart call-manager          # GCP_BUCKET_NAME_MEDIA

Telephony and messaging providers
--------------------------------------

.. list-table::
   :header-rows: 1
   :widths: 25 40 35

   * - Provider
     - Env var(s)
     - Consuming service
   * - Twilio (PSTN numbers)
     - ``TWILIO_ACCOUNT_SID`` and ``TWILIO_AUTH_TOKEN``
     - ``number-manager``
   * - Telnyx (PSTN numbers)
     - ``TELNYX_API_KEY``, ``TELNYX_CONNECTION_ID`` and ``TELNYX_PROFILE_ID`` (optional)
     - ``number-manager``
   * - Telnyx (SMS and MMS)
     - ``TELNYX_API_KEY``
     - ``message-manager``
   * - MessageBird (SMS)
     - ``MESSAGEBIRD_API_KEY``
     - ``message-manager``

``number-manager`` reads the carrier secret (``TELNYX_API_KEY`` or
``TWILIO_AUTH_TOKEN``) from the environment only. The ``enabled``, ``priority``
and ``config`` of the Number Provider settings are still read from the
database. The three variables ``TELNYX_CONNECTION_ID``,
``TELNYX_PROFILE_ID`` and ``TWILIO_ACCOUNT_SID`` fill a ``config`` key that
the stored setting does not have, so a deployment without any stored
setting works with the variables alone; a value that is stored in the
``config`` wins over the variable (a ``config`` value left from before
hides a changed variable, so check ``config`` after a change). ``trunk_sid``
of Twilio has no variable and is set in the ``config``. Without the secret
variable no number can be purchased or released and the available number
search fails. The Number Provider API does not accept a secret.
``message-manager`` reads its MessageBird and Telnyx keys the same way
(``MESSAGEBIRD_API_KEY`` and ``TELNYX_API_KEY``). Without ``TELNYX_API_KEY``
the SMS and MMS sends of ``message-manager`` fail at the provider.
Restart the service after a change of a variable.

.. code-block:: bash

    sudo ./voipbin restart number-manager

Email providers
-------------------

.. list-table::
   :header-rows: 1
   :widths: 25 40 35

   * - Provider
     - Env var(s)
     - Consuming service
   * - SendGrid
     - ``SENDGRID_API_KEY`` (restart ``email-manager``)
     - ``email-manager``
   * - Mailgun
     - ``MAILGUN_API_KEY`` and ``MAILGUN_WEBHOOK_SIGNING_KEY`` (restart
       ``email-manager``)
     - ``email-manager``

Configuring at least one of these two is required for the email-based
signup flow (Path B in First login and API access) to actually deliver
its verification email.
