.. _quickstart-main:

*******************
Quickstart
*******************

Getting Started
===============
Create your VoIPBIN account and get API credentials.

.. include:: quickstart_signup.rst
.. include:: quickstart_authentication.rst

Set Up Your Test Environment
============================
Before running the tutorials, set up a SIP extension and softphone for voice testing, and configure event delivery to observe real-time notifications.

.. include:: quickstart_extension.rst
.. include:: quickstart_events.rst

Make your first Hello World
===========================
Try VoIPBIN's core communication APIs:

1. **Your First Call** — Place an outbound voice call with text-to-speech.
2. **Your First Real-Time Voice Interaction** — Make a call with live transcription and speak into the call using the TTS API.

.. note:: **AI Implementation Hint**

   Tutorial 2 (Real-Time Voice) builds on the extension and event setup from the previous section. Retain your authentication token (String) or accesskey (String) from Getting Started for all tutorials. Tokens expire after 7 days — if you receive a ``401 Unauthorized`` response, re-authenticate via ``POST /auth/login``.

.. include:: quickstart_call.rst
.. include:: quickstart_realtime.rst

.. include:: quickstart_sandbox.rst

.. _quickstart_next:

What's Next
===========
Now that you have completed the quickstart, explore the full capabilities of VoIPBIN:

- :ref:`Flow <flow-main>` — Build programmable voice workflows with the visual flow builder.
- :ref:`AI <ai-main>` — Integrate AI-powered voice agents with real-time speech processing.
- :ref:`Conference <conference-main>` — Set up multi-party conferencing.
- :ref:`Conversation <conversation-main>` — Manage messaging conversations.
- :ref:`Queue <queue-main>` — Route incoming calls to available agents with queue management.

For the complete API reference, visit the `API documentation <https://api.voipbin.net/redoc/index.html>`_.

.. note:: **AI Implementation Hint**

   The full OpenAPI 3.0 specification is available as machine-readable JSON at ``https://api.voipbin.net/openapi.json``. Use this for automated client generation, API discovery, and building integrations programmatically.
