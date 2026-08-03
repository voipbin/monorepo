.. _conference-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with conferences, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* (Optional) A flow ID (UUID) for ``pre_flow_id`` or ``post_flow_id``. Create one via ``POST /flows`` or obtain from ``GET /flows``.
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
                "pre_flow_id": "00000000-0000-0000-0000-000000000000",
                "post_flow_id": "00000000-0000-0000-0000-000000000000",
                "conferencecall_ids": [],
                "recording_id": "00000000-0000-0000-0000-000000000000",
                "recording_ids": [],
                "transcribe_id": "00000000-0000-0000-0000-000000000000",
                "transcribe_ids": [],
                "direct_hash": "",
                "tm_end": null,
                "tm_create": "2021-02-04 02:55:39.659316",
                "tm_update": "2021-02-04 02:56:07.525985",
                "tm_delete": null
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
        "pre_flow_id": "00000000-0000-0000-0000-000000000000",
        "post_flow_id": "00000000-0000-0000-0000-000000000000",
        "conferencecall_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "transcribe_id": "00000000-0000-0000-0000-000000000000",
        "transcribe_ids": [],
        "direct_hash": "",
        "tm_end": null,
        "tm_create": "2021-02-03 10:44:42.163464",
        "tm_update": "2021-02-03 10:52:08.488301",
        "tm_delete": null
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
        "pre_flow_id": "00000000-0000-0000-0000-000000000000",
        "post_flow_id": "00000000-0000-0000-0000-000000000000",
        "conferencecall_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "transcribe_id": "00000000-0000-0000-0000-000000000000",
        "transcribe_ids": [],
        "direct_hash": "",
        "tm_end": null,
        "tm_create": "2021-02-04 03:05:57.710583",
        "tm_update": "2021-02-04 03:05:57.710583",
        "tm_delete": null
    }

Update a conference
--------------------

.. note:: **AI Implementation Hint**

   ``PUT /conferences/{id}`` replaces ``name``, ``detail``, ``data``, ``timeout``, ``pre_flow_id``, and ``post_flow_id`` in a single call. ``type`` cannot be changed after creation. Fields you omit from the request body are reset to their zero value, so send the full set of fields you want to keep, not just the ones you're changing.

Example
+++++++

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/conferences/85252d7b-777b-4580-9420-4df8c6adfc30?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "test conference (renamed)",
            "detail": "updated conference detail",
            "data": {},
            "timeout": 3600,
            "pre_flow_id": "00000000-0000-0000-0000-000000000000",
            "post_flow_id": "00000000-0000-0000-0000-000000000000"
        }'

    {
        "id": "85252d7b-777b-4580-9420-4df8c6adfc30",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "conference",
        "status": "progressing",
        "name": "test conference (renamed)",
        "detail": "updated conference detail",
        "timeout": 3600,
        "pre_flow_id": "00000000-0000-0000-0000-000000000000",
        "post_flow_id": "00000000-0000-0000-0000-000000000000",
        "conferencecall_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "transcribe_id": "00000000-0000-0000-0000-000000000000",
        "transcribe_ids": [],
        "direct_hash": "",
        "tm_end": null,
        "tm_create": "2021-02-04 03:05:57.710583",
        "tm_update": "2021-02-04 03:07:12.881204",
        "tm_delete": null
    }

Kick the conferencecall from the conference
-------------------------------------------

.. note:: **AI Implementation Hint**

   Kicking is only meaningful for participants whose status is ``joining`` or ``joined``. If the conferencecall is already ``leaving`` or ``leaved``, ``DELETE /conferencecalls/{id}`` is a no-op: it returns ``200`` with the conferencecall's current (unchanged) state rather than an error. Obtain the conferencecall ID from the conference's ``conferencecall_ids`` array (via ``GET /conferences/{id}``) or from ``GET /conferencecalls``.

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
        "tm_delete": null
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
        "pre_flow_id": "00000000-0000-0000-0000-000000000000",
        "post_flow_id": "00000000-0000-0000-0000-000000000000",
        "conferencecall_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "transcribe_id": "00000000-0000-0000-0000-000000000000",
        "transcribe_ids": [],
        "direct_hash": "b3c4d5e6f7a8",
        "tm_end": null,
        "tm_create": "2022-02-03 06:08:56.672025",
        "tm_update": "2022-08-06 19:11:13.040418",
        "tm_delete": null
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

* **Kick request succeeds but participant was already gone:**
    * **Cause:** ``DELETE /conferencecalls/{id}`` was called on a participant whose status was already ``leaving`` or ``leaved``.
    * **Behavior:** This is not an error. The endpoint returns ``200`` with the conferencecall's unchanged current state. Check the returned ``status`` field to confirm whether the kick had any effect.

* **Conference has no participants:**
    * **Cause:** Participants join through flow actions (``conference_join``), not through the conference API.
    * **Fix:** Create a call with a flow containing a ``conference_join`` action that references the conference ID.
