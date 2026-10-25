.. _number-provider-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

* An authentication token with **ProjectSuperAdmin** permission. Obtain one via ``POST /auth/login``.
* The carrier account values: the API token and, for Telnyx, the ``connection_id`` of the connection used for number orders.

.. note:: **AI Implementation Hint**

   The examples use obviously fake values. Never place a real secret in a document, a log or a chat message. The secret is accepted only in the request body of ``PUT /number_providers/{provider_name}/secret``, never in the URL.


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
                "has_secret": false,
                "key_version": 0,
                "credential_source": "env",
                "usable": true
            },
            {
                "provider_name": "twilio",
                "enabled": false,
                "priority": 20,
                "has_secret": false,
                "key_version": 0,
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


Store the settings and the secret of a provider
-----------------------------------------------
Set the non-secret ``config`` first, because storing a secret requires the ``config`` keys the provider needs.

Request (settings)

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/number_providers/telnyx?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "config": {
                "connection_id": "1234567890123456789"
            }
        }'

Request (secret)

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/number_providers/telnyx/secret?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "secret": "example-secret-token"
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
        "key_version": 1,
        "credential_source": "db",
        "usable": true,
        "updated_by": "550e8400-e29b-41d4-a716-446655440000",
        "secret_updated_by": "00000000-0000-4000-8000-000000000002",
        "tm_update": "2026-01-03T04:05:06.000000Z",
        "tm_secret_update": "2026-01-03T04:05:06.000000Z"
    }

The response shows ``has_secret`` and ``key_version`` only. The secret is never returned.


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


Delete the stored secret
------------------------
Use this to return to the platform environment credentials (``credential_source`` becomes ``none`` if the environment has none), or to recover a provider whose ``credential_source`` is ``db_undecryptable``.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/number_providers/telnyx/secret?token=<YOUR_AUTH_TOKEN>'

Response (200 OK)

.. code::

    {
        "provider_name": "telnyx",
        "enabled": true,
        "priority": 10,
        "has_secret": false,
        "key_version": 0,
        "credential_source": "env",
        "usable": true,
        "updated_by": "550e8400-e29b-41d4-a716-446655440000",
        "secret_updated_by": "00000000-0000-4000-8000-000000000002",
        "tm_update": "2026-01-04T05:06:07.000000Z",
        "tm_secret_update": "2026-01-04T05:06:07.000000Z"
    }

A known provider for which no request has created a stored setting yet answers ``404``. See the account change warning in :ref:`Overview <number-provider-overview>` before you delete a secret.
