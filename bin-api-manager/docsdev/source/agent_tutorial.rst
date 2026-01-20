.. _agent-tutorial:

Tutorial
========

Create a new agent
------------------

Create a new agent.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/agents?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "username": "test2",
            "password": "test2",
            "name": "test tag",
            "detail": "test tag example",
            "ring_method": "ringall",
            "permission": 0,
            "tag_ids": ["d7450dda-21e0-4611-b09a-8d771c50a5e6"]
        }'


Update agent's status
---------------------
Update agent's status to the available.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b/status?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "status": "available"
        }'

Update agent's addresses
------------------------
Update agent's addresses.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b/addresses?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "addresses": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ]
        }'
