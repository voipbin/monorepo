.. _providercall-struct-providercall:

ProviderCall
============

.. _providercall-struct-providercall-providercall:

ProviderCall
------------

A ``ProviderCall`` is a persisted audit record for an admin-triggered call placed through a specific SIP provider. It captures the admin's original request (customer, provider, flow, source, destinations, anonymous option) plus the identifiers of the calls and groupcalls that were created by the underlying call-creation step. The ``ProviderCall`` itself does not embed the full Call/Groupcall records — use the IDs with ``GET https://api.voipbin.net/v1.0/calls/{id}`` to observe per-call progress.

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "provider_id": "<string>",
        "flow_id": "<string>",
        "source": {
            "type": "<string>",
            "target": "<string>",
            "target_name": "<string>"
        },
        "destinations": [
            {
                "type": "<string>",
                "target": "<string>",
                "target_name": "<string>"
            }
        ],
        "anonymous": "<string>",
        "call_ids": ["<string>"],
        "groupcall_ids": ["<string>"],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The ``ProviderCall``'s unique identifier. Returned when creating via ``POST https://api.voipbin.net/v1.0/providercalls`` or listing via ``GET https://api.voipbin.net/v1.0/providercalls``.
* ``customer_id`` (UUID): The customer the record is attributed to. Set server-side from the authenticated admin's own customer (from the JWT/accesskey); not settable via the request body.
* ``provider_id`` (UUID): The provider the call was forced through. Obtained from the ``id`` field of ``GET https://api.voipbin.net/v1.0/providers``.
* ``flow_id`` (UUID): The flow executed after the destination answered. ``00000000-0000-0000-0000-000000000000`` when no flow was attached (inline ``actions`` or no post-answer logic).
* ``source`` (Object, nullable): The admin-supplied source address (caller ID). Preserved verbatim — the normal customer-ownership check on the source number is bypassed for this flow so a provider-allowlisted caller ID reaches the carrier unchanged. ``null`` if the admin did not supply one.
* ``destinations`` (Array of Object, minItems: 1): The admin-supplied dial targets. One ``Call`` or ``Groupcall`` is created per destination, depending on destination type.
* ``anonymous`` (enum string): The anonymous caller-ID option requested. See :ref:`Anonymous <providercall-struct-providercall-anonymous>`.
* ``call_ids`` (Array of UUID): IDs of the ``Call`` records that the call-creation step produced. Use each ID with ``GET https://api.voipbin.net/v1.0/calls/{id}`` to observe per-call progress.
* ``groupcall_ids`` (Array of UUID): IDs of any ``Groupcall`` records that were produced (when a destination resolved to a group-type address). Use each ID with ``GET https://api.voipbin.net/v1.0/groupcalls/{id}``.
* ``tm_create`` (string, ISO 8601): Timestamp when the ``ProviderCall`` record was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to the record.
* ``tm_delete`` (string, ISO 8601): Timestamp when the ``ProviderCall`` was soft-deleted. Remains ``null`` until a ``DELETE`` call is made.

.. note:: **AI Implementation Hint**

   The ``ProviderCall`` is a summary record — the actual per-call state (dialing, ringing, answered, hangup reason) lives on each ``Call`` referenced by ``call_ids``. Do not poll the ``ProviderCall`` itself for call progress. Instead, iterate ``call_ids`` and poll ``GET https://api.voipbin.net/v1.0/calls/{id}`` (or listen for the corresponding webhooks) to determine outcome.

Example
+++++++

.. code::

    {
        "id": "b7d1c0f6-9a2e-4b3f-8e2a-1c7d5b8a9e0f",
        "customer_id": "6a93f71e-8b2d-4e5f-9a1c-2d3e4f5a6b7c",
        "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "flow_id": "00000000-0000-0000-0000-000000000000",
        "source": {
            "type": "tel",
            "target": "+14155551234",
            "target_name": ""
        },
        "destinations": [
            {
                "type": "tel",
                "target": "+821012345678",
                "target_name": ""
            }
        ],
        "anonymous": "auto",
        "call_ids": ["9f8e7d6c-5b4a-3c2d-1e0f-abcdef012345"],
        "groupcall_ids": [],
        "tm_create": "2026-04-21 23:15:00.000000",
        "tm_update": "2026-04-21 23:15:00.000000",
        "tm_delete": null
    }

.. _providercall-struct-providercall-anonymous:

Anonymous
+++++++++

Controls whether the outbound caller ID is anonymized on the INVITE that leaves VoIPbin.

=========== ====================================================================
Value       Description
=========== ====================================================================
``yes``     Always send anonymous caller ID (RFC 3323 Privacy header). The real
            source number is carried in the P-Asserted-Identity header (RFC 3325)
            so carriers can route and bill correctly while the called party sees
            "Anonymous".
``no``      Never anonymize. The source number is sent as-is.
``auto``    Default behavior. Today this resolves the same as ``no``; in the
            future it will inherit from the incoming channel's SIP Privacy header
            when the outbound call is a relay of an inbound leg.
``""``      Empty string. Equivalent to ``auto``.
=========== ====================================================================
