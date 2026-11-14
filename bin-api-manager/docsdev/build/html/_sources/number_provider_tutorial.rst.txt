.. _number-provider-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

* An authentication token with **ProjectSuperAdmin** permission. Obtain one via ``POST /auth/login``.
* The carrier secret (``TELNYX_API_KEY`` or ``TWILIO_AUTH_TOKEN``) must be set as an environment variable of ``number-manager`` by the platform operator. For Telnyx the ``connection_id`` is set in the ``config`` or as ``TELNYX_CONNECTION_ID``.

.. note:: **AI Implementation Hint**

   The examples use obviously fake values. Never place a real secret in a document, a log or a chat message.


List the number providers
-------------------------

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/number_providers?token=<YOUR_AUTH_TOKEN>'

Response (200 OK)

.. code::

    {
        "result": [
            {
                "provider_name": "telnyx",
                "enabled": true,
                "priority": 10,
                "has_secret": true,
                "credential_source": "env",
                "usable": true
            },
            {
                "provider_name": "twilio",
                "enabled": false,
                "priority": 20,
                "has_secret": false,
                "credential_source": "none",
                "usable": false
            }
        ],
        "next_page_token": ""
    }

All providers are returned in one response, so ``next_page_token`` is always empty.


Get one number provider
-----------------------

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/number_providers/telnyx?token=<YOUR_AUTH_TOKEN>'


Store the settings of a provider
--------------------------------
Set the non-secret ``config`` (it can also come from ``TELNYX_CONNECTION_ID``, ``TELNYX_PROFILE_ID`` or ``TWILIO_ACCOUNT_SID``).

Request (settings)

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/number_providers/telnyx?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "config": {
                "connection_id": "1234567890123456789"
            }
        }'

Response (200 OK)

.. code::

    {
        "provider_name": "telnyx",
        "enabled": true,
        "priority": 10,
        "config": {
            "connection_id": "1234567890123456789"
        },
        "has_secret": true,
        "credential_source": "env",
        "usable": true,
        "updated_by": "550e8400-e29b-41d4-a716-446655440000",
        "tm_update": "2026-01-03T04:05:06.000000Z"
    }

``has_secret`` and ``credential_source`` follow the environment variable of the platform. The secret is never returned.


Change the purchase order or disable a provider
-----------------------------------------------

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/number_providers/telnyx?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "enabled": false,
            "priority": 5
        }'

To remove one ``config`` key, send it with an empty string value, for example ``{"config": {"profile_id": ""}}``.

See the account change warning in :ref:`Overview <number-provider-overview>` before you change the secret variable of a carrier that is in use.
