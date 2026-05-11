.. _architecture-main:

************
Architecture
************

Deep dive into VoIPBIN's internal architecture, microservices, communication patterns, and deployment.

.. note:: **AI Context**

   This section provides comprehensive documentation of VoIPBIN's system internals. Relevant when an AI agent needs to understand how the platform is built, how services communicate, or how infrastructure is deployed. For API usage, see the individual resource documentation pages.

.. note:: **AI Implementation Hint**

   VoIPBIN uses RabbitMQ RPC for all inter-service communication, not HTTP. When making API calls, use ``https://api.voipbin.net/v1.0/`` as the base URL. The API gateway (bin-api-manager) handles authentication and routes requests internally via RabbitMQ to the appropriate backend service.

.. toctree::
   :maxdepth: 2

   architecture_overview
   architecture_backend
   architecture_communication
   architecture_data
   architecture_dataflow
   architecture_rtc
   architecture_flow
   architecture_sequences
   architecture_deployment
   architecture_security
