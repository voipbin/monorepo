.. voipbin documentation master file.
   The root toctree document for docs.voipbin.net.

#####################
VoIPBin Documentation
#####################

**Opensource CPaaS platform** for Voice, SMS, Email, Chat, and Social. Own your communications stack.

.. note:: **AI Implementation Hint**

   For API integration, start with the :ref:`Quickstart <quickstart-main>` guide. For the full machine-readable API spec, use ``GET https://api.voipbin.net/openapi.json`` (OpenAPI 3.0 JSON). For interactive API reference, see `ReDoc <https://api.voipbin.net/redoc/index.html>`_ or `Swagger <https://api.voipbin.net/swagger/index.html>`_.

Start here
==========

- :ref:`Quickstart <quickstart-main>` — Create your account, place a first call, and observe events in under 10 minutes.
- :ref:`Flow <flow-main>` — Build programmable communication workflows with visual flows and AI-driven flows.
- :ref:`Webhook <webhook-main>` — Receive real-time events for calls, messages, and conversations.
- `REST API Reference <https://api.voipbin.net/redoc/index.html>`_ — Full endpoint catalog (ReDoc).

What you can build
==================

- **Voice** — PSTN, SIP, and WebRTC calls. Recording, transcription, AMD, transfers, group calls, conferencing.
- **Messaging** — SMS, Email, and unified Conversation threads across channels.
- **AI** — Voice agents, live transcription, TTS, RAG-grounded responses, agent teams.
- **Routing & Queues** — Inbound flows, queues, agents, campaigns, outdialing.
- **Numbers & Connectivity** — Provision numbers, configure providers, manage extensions and trunks.

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Getting Started

   intro
   quickstart
   sdk
   restful_api
   restful_api_errors

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Core Concepts

   flow
   variable
   webhook
   direct_hash
   common

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Voice & Real-Time

   call
   conference
   queue
   recording
   transcribe
   speaking
   mediastream

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Messaging

   message
   email
   talk
   conversation

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: AI & Automation

   ai
   ai_voice_agent_integration
   rag
   team
   campaign
   outdial
   outplan

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Identity & Contacts

   accesskey
   agent
   contact
   customer

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Numbers & PSTN

   number
   provider
   providercall
   route
   extension
   trunk
   outbound_config

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Billing & Operations

   billing_account
   storage
   tag

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Architecture (Deep Dive)

   architecture
   websocket

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Self-Hosting

   self_hosting

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Reference

   glossary

.. toctree::
   :maxdepth: 2
   :hidden:
   :caption: Help & Support

   support
