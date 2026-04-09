.. _conference-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with conferences, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* (Optional) A flow ID (UUID) for ``pre_actions`` or ``post_actions``. Create one via ``POST /flows`` or obtain from ``GET /flows``.
* (Optional) To add participants, you need an active call. Create one via ``POST /calls`` or obtain from ``GET /calls``.

.. note:: **AI Implementation Hint**

   Conferences are created with ``POST /conferences`` and begin in ``starting`` status, quickly transitioning to ``progressing``. Participants do not join via the conference API directly -- they join through flow actions (``conference_join``). To remove a participant, use ``DELETE /conferencecalls/{conferencecall_id}``. To terminate the entire conference, use ``DELETE /conferences/{conference_id}``.

Get list of conferences
-----------------------

Example
+++++++

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/conferences?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "17039950-eab0-421d-a5f5-05acd1ac6801",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "type": "conference",
                "status": "progressing",
                "name": "team standup",
                "detail": "Daily standup conference",
                "conferencecall_ids": [],
                "recording_ids": [],
                "transcribe_ids": [],
                "direct_hash": "",
                "tm_end": null,
                "tm_create": "2021-02-04 02:55:39.659316",
                "tm_update": "2021-02-04 02:56:07.525985",
                "tm_delete": "9999-01-01 00:00:00.000000"
            },
            ...
        ],
        "next_page_token": "2021-02-03 09:33:58.077756"
    }


Get detail of conference
------------------------

Example
+++++++

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/conferences/0e7112d7-6ddc-47ea-bba5-223a3a55ff79?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "0e7112d7-6ddc-47ea-bba5-223a3a55ff79",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "conference",
        "status": "progressing",
        "name": "team standup",
        "detail": "Daily standup conference",
        "conferencecall_ids": [],
        "recording_ids": [],
        "transcribe_ids": [],
        "direct_hash": "",
        "tm_end": null,
        "tm_create": "2021-02-03 10:44:42.163464",
        "tm_update": "2021-02-03 10:52:08.488301",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Create a new conference
-----------------------

Example
+++++++

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/conferences?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "type": "conference",
            "name": "test conference",
            "detail": "test conference for example"
        }'

    {
        "id": "85252d7b-777b-4580-9420-4df8c6adfc30",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "conference",
        "status": "starting",
        "name": "test conference",
        "detail": "test conference for example",
        "conferencecall_ids": [],
        "recording_ids": [],
        "transcribe_ids": [],
        "direct_hash": "",
        "tm_end": null,
        "tm_create": "2021-02-04 03:05:57.710583",
        "tm_update": "2021-02-04 03:05:57.710583",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Kick the conferencecall from the conference
-------------------------------------------

.. note:: **AI Implementation Hint**

   You can only kick participants whose status is ``joining`` or ``joined``. Attempting to delete a conferencecall in ``leaving`` or ``leaved`` status will fail. Obtain the conferencecall ID from the conference's ``conferencecall_ids`` array (via ``GET /conferences/{id}``) or from ``GET /conferencecalls``.

Example
+++++++

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/conferencecalls/4833755c-f5d0-4bf2-a101-7d3a7e5e586f?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "4833755c-f5d0-4bf2-a101-7d3a7e5e586f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "conference_id": "99accfb7-c0dd-4a54-997d-dd18af7bc280",
        "reference_type": "call",
        "reference_id": "153c2866-ade0-4a55-a5a7-027e463d9207",
        "status": "leaving",
        "tm_create": "2022-08-09 03:53:49.142446",
        "tm_update": "2022-08-09 03:54:10.035297",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Regenerate direct conference hash
----------------------------------

Regenerate the direct hash for a conference. This invalidates the previous SIP URI and creates a new one. If the conference has no existing direct hash, one is created automatically.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/conferences/99accfb7-c0dd-4a54-997d-dd18af7bc280/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "99accfb7-c0dd-4a54-997d-dd18af7bc280",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "conference",
        "status": "progressing",
        "name": "test conference",
        "detail": "test conference for example.",
        "conferencecall_ids": [],
        "recording_ids": [],
        "transcribe_ids": [],
        "direct_hash": "b3c4d5e6f7a8",
        "tm_end": null,
        "tm_create": "2022-02-03 06:08:56.672025",
        "tm_update": "2022-08-06 19:11:13.040418",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. note:: **AI Implementation Hint**

   This endpoint requires no request body. The ``direct_hash`` in the response is the new hash — the previous hash is permanently invalidated. The direct SIP URI format is ``sip:direct.<hash>@sip.voipbin.net``.

Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** Invalid conference creation parameters (e.g., missing ``type`` field).
    * **Fix:** Ensure the request body includes ``"type": "conference"`` and a ``name`` string.

* **404 Not Found:**
    * **Cause:** The conference UUID or conferencecall UUID does not exist or belongs to a different customer.
    * **Fix:** Verify the UUID was obtained from ``GET /conferences`` or ``GET /conferencecalls``.

* **409 Conflict (kick participant):**
    * **Cause:** Attempted to kick a participant whose status is ``leaving`` or ``leaved``.
    * **Fix:** Check participant status via ``GET /conferencecalls/{id}`` before attempting deletion. Only ``joining`` or ``joined`` participants can be kicked.

* **Conference has no participants:**
    * **Cause:** Participants join through flow actions (``conference_join``), not through the conference API.
    * **Fix:** Create a call with a flow containing a ``conference_join`` action that references the conference ID.
