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


Update provider
--------------------------

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


Delete provider
---------------

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/providers/7efc9379-2d3e-4e54-9d36-23cff676a83e?token=<YOUR_AUTH_TOKEN>'

