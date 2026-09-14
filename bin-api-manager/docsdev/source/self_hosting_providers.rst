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
     - ``OPENAI_API_KEY``, ``GOOGLE_API_KEY``, ``XAI_API_KEY``
       (restart ``pipecat-manager``)
   * - Speech-to-text
     - Deepgram, AWS Transcribe, Google Cloud Speech-to-Text
     - ``DEEPGRAM_API_KEY``; ``AWS_ACCESS_KEY``/``AWS_SECRET_KEY``;
       ``GOOGLE_APPLICATION_CREDENTIALS`` (restart ``transcribe-manager``)
   * - Text-to-speech
     - ElevenLabs, Cartesia, AWS Polly, Google Cloud TTS
     - ``ELEVENLABS_API_KEY``, ``CARTESIA_API_KEY``;
       ``AWS_ACCESS_KEY``/``AWS_SECRET_KEY``;
       ``GOOGLE_APPLICATION_CREDENTIALS`` (restart ``tts-manager``)

``GOOGLE_API_KEY`` and ``GOOGLE_APPLICATION_CREDENTIALS`` are two
unrelated variables that happen to share the "Google" prefix. Do not
confuse them:

- ``GOOGLE_API_KEY`` is a Gemini API key. It is consumed by two
  independent features: ``ai-manager``'s batch audit evaluation
  (mapped internally to a ChatGPT-compatible engine key,
  ``ENGINE_KEY_CHATGPT=${OPENAI_API_KEY}``, does **not** actually use
  ``GOOGLE_API_KEY`` for this) and ``pipecat-manager``'s real-time voice
  pipeline (the sidecar that actually reads ``GOOGLE_API_KEY`` directly
  for live Gemini conversations). Restart ``pipecat-manager`` after
  changing it for the voice-pipeline use.
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

``XAI_API_KEY`` has no dedicated feature write-up beyond its
``.env.template`` entry; it is consumed by ``pipecat-manager``'s
real-time voice pipeline as an LLM alternative, not by ``ai-manager``.

After setting an AI provider key, restart the service(s) that actually
consume that specific variable, per the tables above:

.. code-block:: bash

    sudo ./voipbin restart ai-manager           # OPENAI_API_KEY (batch)
    sudo ./voipbin restart pipecat-manager       # OPENAI_API_KEY, GOOGLE_API_KEY, XAI_API_KEY (real-time)
    sudo ./voipbin restart transcribe-manager    # DEEPGRAM_API_KEY, AWS_*, GOOGLE_APPLICATION_CREDENTIALS
    sudo ./voipbin restart tts-manager           # ELEVENLABS_API_KEY, CARTESIA_API_KEY, AWS_*, GOOGLE_APPLICATION_CREDENTIALS
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
     - ``TWILIO_SID``, ``TWILIO_API_KEY``
     - ``number-manager``
   * - Telnyx (PSTN numbers and messaging)
     - ``TELNYX_API_KEY``, ``TELNYX_CONNECTION_ID``,
       ``TELNYX_PROFILE_ID`` (all three required)
     - ``number-manager``
   * - MessageBird (SMS)
     - ``MESSAGEBIRD_API_KEY``
     - ``message-manager``

The ``.env.template`` variable names above are what you set;
``number-manager`` itself binds them internally as ``TWILIO_TOKEN`` and
``TELNYX_TOKEN`` (``docker-compose.yml.dist`` maps
``TWILIO_TOKEN=${TWILIO_API_KEY}`` and
``TELNYX_TOKEN=${TELNYX_API_KEY}``), and ``message-manager`` binds
``MESSAGEBIRD_API_KEY`` internally as ``AUTHTOKEN_MESSAGEBIRD``. This
only matters if you go looking for the raw variable name inside
``number-manager``'s or ``message-manager``'s own config; the ``.env``
names above are what you actually set. MessageBird has no dedicated
feature write-up beyond its ``.env.template`` entry.

.. code-block:: bash

    sudo ./voipbin restart number-manager
    sudo ./voipbin restart message-manager

Email providers
-------------------

.. list-table::
   :header-rows: 1
   :widths: 25 40 35

   * - Provider
     - Env var
     - Consuming service
   * - SendGrid
     - ``SENDGRID_API_KEY``
     - ``email-manager``
   * - Mailgun
     - ``MAILGUN_API_KEY``
     - ``email-manager``

.. code-block:: bash

    sudo ./voipbin restart email-manager

Configuring at least one of these two is required for the email-based
signup flow (Path B in First login and API access) to actually deliver
its verification email.
