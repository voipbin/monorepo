.. _provider-struct-provider:

Provider
========

.. _provider-struct-provider-provider:

Provider
--------

.. code::

    {
        "id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "type": "<string>",
        "hostname": "<string>",
        "tech_prefix": "<string">,
        "tech_postfix": "<string>",
        "tech_headers": {
            "<string>": "<string">,
        },
        "metadata": {},
        "codecs": "<string>",
        "health_status": "<string>",
        "health_checked_at": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The provider's unique identifier. Returned when creating via ``POST /providers`` or listing via ``GET /providers``.
* ``name`` (String): A human-readable label for the provider. Free-form text for organizational use.
* ``detail`` (String): A longer description of the provider's purpose or configuration notes.
* ``type`` (enum string): The provider's protocol type. See :ref:`Type <provider-struct-provider-type>`.
* ``hostname`` (String): The SIP server address for the provider (e.g., ``sip.telnyx.com``). Used as the destination for outbound SIP INVITE messages.
* ``tech_prefix`` (String): Prefix attached to the beginning of the dialed destination number. Valid only for type ``sip``. Leave as empty string if not required.
* ``tech_postfix`` (String): Postfix attached to the end of the dialed destination number. Valid only for type ``sip``. Leave as empty string if not required.
* ``tech_headers`` (Object): Key/value pairs of custom SIP headers sent with outbound calls. Valid only for type ``sip``. Use empty object ``{}`` if not required.
* ``metadata`` (Object): Carrier-specific resource identifiers stored during automated setup. For Telnyx providers created via ``POST /providers/setup``, contains ``telnyx_profile_id``, ``telnyx_connection_id``, and ``telnyx_ip_ids``. Read-only — populated automatically by the setup endpoint. Empty object ``{}`` if the provider was created manually via ``POST /providers``.
* ``codecs`` (String): Comma-separated list of preferred audio codecs for SIP calls routed through this provider (e.g., ``"PCMU,PCMA"``). Leave as empty string ``""`` to use the system default codec negotiation. Valid only for type ``sip``.
* ``health_status`` (enum string): The result of the most recent SIP OPTIONS health check. See :ref:`Health Status <provider-struct-provider-health-status>`.
* ``health_checked_at`` (string, ISO 8601 or null): Timestamp of the last completed health check. ``null`` if no check has been performed yet. Resets to ``null`` when ``hostname`` is updated.
* ``tm_create`` (string, ISO 8601): Timestamp when the provider was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any provider property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the provider was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the provider has not been deleted.

Example
+++++++

.. code::

    {
        "id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "type": "sip",
        "hostname": "sip.telnyx.com",
        "tech_prefix": "",
        "tech_postfix": "",
        "tech_headers": {},
        "metadata": {
            "telnyx_profile_id": "2944757397136082899",
            "telnyx_connection_id": "2944757397198982899",
            "telnyx_ip_ids": ["2944757397261882899"]
        },
        "codecs": "PCMU,PCMA",
        "health_status": "healthy",
        "health_checked_at": "2026-04-20 03:15:00.000000",
        "name": "telnyx basic",
        "detail": "telnyx basic",
        "tm_create": "2022-10-22 16:16:16.874761",
        "tm_update": "2022-10-24 04:53:14.171374",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _provider-struct-provider-health-status:

Health Status
+++++++++++++

Reflects the result of the most recent SIP OPTIONS health check sent from the Kamailio VM.

============= ===========================================================================
Health Status Description
============= ===========================================================================
unknown       No health check has been performed yet. Initial state for all new providers
              and after the ``hostname`` field is updated.
healthy       The provider responded to a SIP OPTIONS request within the timeout window.
              Any SIP response code is considered healthy.
unhealthy     No SIP response was received within the timeout (default 5 seconds), or the
              hostname could not be resolved.
============= ===========================================================================

.. note:: **AI Implementation Hint**

   ``health_status`` is read-only. It is set automatically by the background health check
   goroutine and cannot be set via the API. When ``hostname`` is updated on a provider,
   ``health_status`` resets to ``unknown`` and ``health_checked_at`` resets to ``null``
   until the next health check cycle completes.

.. _provider-struct-provider-type:

Type
----
Defines types of provider.

=========== ====================================================================
Type        Description
=========== ====================================================================
sip         SIP service provider. Uses Session Initiation Protocol for voice
            call routing to the PSTN. Examples: Telnyx, Twilio, Bandwidth.
=========== ====================================================================
