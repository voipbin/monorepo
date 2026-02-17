.. _provider-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with providers, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* The provider's SIP hostname (e.g., ``sip.telnyx.com``). This is provided by the telephony service provider.
* (Optional) Any custom SIP headers or number formatting rules required by the provider.

.. note:: **AI Implementation Hint**

   Providers are typically managed by platform administrators. After creating a provider, you must create at least one route (via ``POST /routes``) that references the provider's ``id`` before outbound calls can use it. The provider alone does not enable call routing.

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
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
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

