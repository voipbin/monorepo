.. _service_agent-tutorial:

Tutorial
========

This walkthrough builds a typical agent-console session end to end: log in as an agent, go available, pick up a case, work it, and close it out -- then join a team channel to ask for help.

Prerequisites
+++++++++++++

* An agent account with a username/password, created via ``POST /agents`` (admin/manager surface) beforehand.
* An authentication token for that agent, obtained via ``POST /auth/boot`` (see :ref:`Agent <agent-overview>` for the login flow).

.. note:: **AI Implementation Hint**

   Every request below uses the same JWT and the same base path prefix, ``/service_agents/*``. There is no separate "agent console API key" -- the identity and customer scope come entirely from the JWT. See :ref:`Agent Console Overview <service_agent-overview>` for the full authorization model.


1. Check your own profile and go available
--------------------------------------------

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/me?token=<token>'

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/service_agents/me/status?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "status": "available"
        }'


2. List open cases and pick one up
-------------------------------------

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/contact_cases?page_size=20&token=<token>'

Filter the ``result`` array client-side for ``"status": "open"`` and an empty/missing ``owner_id`` to build a "New" queue view. Once you've picked a case, assign it to yourself:

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>/assign?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "owner_id": "<your-own-agent-id>"
        }'

``<your-own-agent-id>`` is the ``id`` field from the ``GET /service_agents/me`` response in step 1.


3. Review interaction history for the case's peer
----------------------------------------------------
Before working the case, pull the interaction history for the peer it's scoped to (the ``peer`` field on the Case, from step 2):

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/contact_interactions?peer_type=tel&peer_target=%2B155****4567&token=<token>'

If the case is already attached to a resolved Contact (``contact_id`` is non-null), you can also pull the contact's full profile:

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/contacts/<contact-id>?token=<token>'


4. Leave a note for the next agent
--------------------------------------

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>/notes?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "text": "Customer confirmed the outage affects only their US-East line. Escalating to network team."
        }'

Notes are agent-only -- they never appear in any customer-facing response.


5. Attach the resolved contact and close the case
-------------------------------------------------------
If the case wasn't already linked to a Contact, attach one once you've identified the customer:

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "contact_id": "<contact-id>"
        }'

Once resolved, close the case:

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>/close?token=<token>'


6. Ask a teammate for help via Talk
---------------------------------------
Browse public channels and join the on-call channel:

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/talk_channels?token=<token>'

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_chats/<channel-id>/join?token=<token>'

Send a message:

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "chat_id": "<channel-id>",
            "type": "normal",
            "text": "Anyone else seeing US-East line outages? Case <case-id> escalated to network team."
        }'

See :ref:`Talk <talk-main>` for message threading, reactions, and media attachments.


7. Stay in sync with real-time updates
-------------------------------------------
Open a WebSocket connection scoped to your agent identity to receive new case updates and chat messages without polling:

.. code::

    $ wscat -c 'wss://api.voipbin.net/v1.0/service_agents/ws?token=<token>'

See :ref:`WebSocket <websocket-main>` for the event catalog.


8. Go offline at the end of your shift
-------------------------------------------

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/service_agents/me/status?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "status": "offline"
        }'
