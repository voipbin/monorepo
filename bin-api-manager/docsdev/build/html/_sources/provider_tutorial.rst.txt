.. _provider-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with providers, you need:

* An authentication token with **ProjectSuperAdmin** permission. Obtain one via ``POST /auth/login``.
* The provider's SIP hostname (e.g., ``sip.telnyx.com``), or a carrier API key if using ``POST /providers/setup``.
* (Optional) Any custom SIP headers or number formatting rules required by the provider.

.. note:: **AI Implementation Hint**

   Providers are typically managed by platform administrators. After creating a provider, you must create at least one route (via ``POST /routes``) that references the provider's ``id`` before outbound calls can use it. The provider alone does not enable call routing.


Set up a provider via carrier API key
--------------------------------------

``POST /providers/setup`` validates your carrier API key, creates the carrier-side SIP trunk, and returns a ready-to-use VoIPBIN provider record — all in one call. Requires **ProjectSuperAdmin** permission.

**Supported carriers:** ``telnyx``

Request

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/providers/setup?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "carrier": "telnyx",
            "name": "My Telnyx Trunk",
            "detail": "Primary outbound Telnyx SIP trunk",
            "credentials": {
                "api_key": "KEY_01234567890abcdef"
            }
        }'

Response (200 OK)

.. code::

    {
        "id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "type": "sip",
        "hostname": "sip.telnyx.com",
        "tech_prefix": "",
        "tech_postfix": "",
        "tech_headers": {},
        "name": "My Telnyx Trunk",
        "detail": "Primary outbound Telnyx SIP trunk",
        "tm_create": "2026-04-23T10:00:00.000000Z",
        "tm_update": "2026-04-23T10:00:00.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }

.. note:: **AI Implementation Hint**

   Save the returned ``id`` field (UUID). You will need it when creating a route via ``POST /routes`` to enable outbound calling through this provider. If the API key is invalid or lacks the required permissions, the endpoint returns ``422 Unprocessable Entity`` — verify the key in the Telnyx portal and retry.

Get list of providers
---------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/providers?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
                "type": "sip",
                "hostname": "sip.telnyx.com",
                "tech_prefix": "",
                "tech_postfix": "",
                "tech_headers": {},
                "name": "telnyx basic",
                "detail": "telnyx basic",
                "tm_create": "2022-10-22 16:16:16.874761",
                "tm_update": "2022-10-24 04:53:14.171374",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
            ...
        ],
        "next_page_token": "2022-10-22 16:16:16.874761"
    }

Get detail of provider
----------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/providers/4dbeabd6-f397-4375-95d2-a38411e07ed1?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "type": "sip",
        "hostname": "sip.telnyx.com",
        "tech_prefix": "",
        "tech_postfix": "",
        "tech_headers": {},
        "name": "telnyx basic",
        "detail": "telnyx basic",
        "tm_create": "2022-10-22 16:16:16.874761",
        "tm_update": "2022-10-24 04:53:14.171374",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Create a new provider
---------------------

The following example creates a provider without codec restrictions. VoIPBIN will use its
system-default codec list during SDP negotiation for calls through this provider.

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/providers?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "type": "sip",
            "hostname": "test.com",
            "tech_prefix": "",
            "tech_postfix": "",
            "tech_headers": {},
            "name": "test domain",
            "detail": "test domain creation"
        }'

To create a provider with a specific codec restriction from the start, include ``"codecs"`` in
the request body:

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/providers?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "type": "sip",
            "hostname": "sip.example.com",
            "tech_prefix": "",
            "tech_postfix": "",
            "tech_headers": {},
            "name": "example carrier",
            "detail": "carrier with G.711 codec restriction",
            "codecs": "PCMU,PCMA"
        }'

.. note:: **AI Implementation Hint**

   ``POST /providers/setup`` (the automated carrier setup endpoint) does **not** accept a
   ``codecs`` parameter. If you need codec restrictions on a provider created via
   ``POST /providers/setup``, use ``PUT /providers/{id}`` afterward to set the ``codecs``
   field. See the update example below.


Update provider
--------------------------

The following example updates all provider fields. Omit ``"codecs"`` or set it to ``""`` to
remove any existing codec restriction and revert to system-default negotiation.

Example

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/providers/4dbeabd6-f397-4375-95d2-a38411e07ed1?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "type": "sip",
            "hostname": "sip.telnyx.com",
            "tech_prefix": "",
            "tech_postfix": "",
            "tech_headers": {},
            "name": "telnyx basic",
            "detail": "telnyx basic"
        }'

Set codec restriction on an existing provider
+++++++++++++++++++++++++++++++++++++++++++++++

Use ``PUT /providers/{id}`` to restrict which audio codecs VoIPBIN offers during SDP
negotiation for PSTN outbound calls through this provider. This is useful when a carrier
only supports specific codecs.

The following example restricts the provider to G.711 µ-law and G.711 A-law:

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/providers/4dbeabd6-f397-4375-95d2-a38411e07ed1?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "type": "sip",
            "hostname": "sip.telnyx.com",
            "tech_prefix": "",
            "tech_postfix": "",
            "tech_headers": {},
            "name": "telnyx basic",
            "detail": "telnyx basic",
            "codecs": "PCMU,PCMA"
        }'

To remove the restriction and revert to system-default codec negotiation, set ``"codecs"``
to an empty string:

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/providers/4dbeabd6-f397-4375-95d2-a38411e07ed1?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "type": "sip",
            "hostname": "sip.telnyx.com",
            "tech_prefix": "",
            "tech_postfix": "",
            "tech_headers": {},
            "name": "telnyx basic",
            "detail": "telnyx basic",
            "codecs": ""
        }'

.. note:: **AI Implementation Hint**

   ``codecs`` only affects PSTN outbound calls routed through this provider. SIP-to-SIP calls
   within VoIPBIN are not affected. The field is valid only for providers with ``"type": "sip"``.
   Leave ``codecs`` as ``""`` unless you have a specific carrier interoperability requirement.


Delete provider
---------------

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/providers/7efc9379-2d3e-4e54-9d36-23cff676a83e?token=<YOUR_AUTH_TOKEN>'

