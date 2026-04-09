.. _architecture-main:

************
Architecture
************

Deep dive into VoIPBIN's internal architecture, microservices, communication patterns, and deployment.

.. note:: **AI Context**

   This section provides comprehensive documentation of VoIPBIN's system internals. Relevant when an AI agent needs to understand how the platform is built, how services communicate, or how infrastructure is deployed. For API usage, see the individual resource documentation pages.

.. note:: **AI Implementation Hint**

   VoIPBIN uses RabbitMQ RPC for all inter-service communication, not HTTP. When making API calls, use ``https://api.voipbin.net/v1.0/`` as the base URL. The API gateway (bin-api-manager) handles authentication and routes requests internally via RabbitMQ to the appropriate backend service.

.. include:: architecture_overview.rst
.. include:: architecture_backend.rst
.. include:: architecture_communication.rst
.. include:: architecture_data.rst
.. include:: architecture_dataflow.rst
.. include:: architecture_rtc.rst
.. include:: architecture_flow.rst
.. include:: architecture_sequences.rst
.. include:: architecture_deployment.rst
.. include:: architecture_security.rst
