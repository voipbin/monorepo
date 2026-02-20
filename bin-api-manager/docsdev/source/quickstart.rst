.. _quickstart-main:

*******************
Quickstart
*******************

There are two ways to get started with VoIPBIN:

- **Try the Demo** — Click the **Try Demo Account** button at `admin.voipbin.net <https://admin.voipbin.net>`_ to explore VoIPBIN instantly. No setup or sign-up needed.
- **Run the Sandbox** — Run the full VoIPBIN platform on your local machine using Docker. See the :ref:`Sandbox <quickstart_sandbox>` section below.

For production use, you can :ref:`sign up <quickstart_signup>` for your own account.

This quickstart walks you through three progressive scenarios:

1. **Your First Call** — Sign up, authenticate, and make an outbound voice call with text-to-speech.
2. **Receiving Events** — Set up webhooks and WebSocket subscriptions to receive real-time notifications.
3. **Real-Time Voice Interaction** — Create a SIP extension, register a softphone, make a call with live transcription, and speak into the call using the TTS API.

.. note:: **AI Implementation Hint**

   Each scenario builds on the previous one. Retain your authentication token (String) or accesskey (String) from Scenario 1 for use in Scenarios 2 and 3. Tokens expire after 7 days — if you receive a ``401 Unauthorized`` response, re-authenticate via ``POST /auth/login``.

.. include:: quickstart_sandbox.rst
.. include:: quickstart_signup.rst
.. include:: quickstart_authentication.rst
.. include:: quickstart_call.rst
.. include:: quickstart_events.rst
.. include:: quickstart_realtime.rst

.. _quickstart_next:

What's Next
===========
Now that you have completed all three scenarios, explore the full capabilities of VoIPBIN:

- :ref:`Flow <flow-main>` — Build programmable voice workflows with the visual flow builder.
- :ref:`AI <ai-main>` — Integrate AI-powered voice agents with real-time speech processing.
- :ref:`Conference <conference-main>` — Set up multi-party conferencing.
- :ref:`Conversation <conversation-main>` — Manage messaging conversations.
- :ref:`Queue <queue-main>` — Route incoming calls to available agents with queue management.

For the complete API reference, visit the `API documentation <https://api.voipbin.net/redoc/index.html>`_.

.. note:: **AI Implementation Hint**

   The full OpenAPI 3.0 specification is available as machine-readable JSON at ``https://api.voipbin.net/openapi.json``. Use this for automated client generation, API discovery, and building integrations programmatically.
