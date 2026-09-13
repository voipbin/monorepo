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
  Docker Compose stack (it only uses ``GOOGLE_API_KEY``, above) —
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
