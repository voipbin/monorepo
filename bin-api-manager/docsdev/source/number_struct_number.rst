.. _number-struct-number:

Number
======

.. _number-struct-number-number:

Number
------

.. code::

    {
        "id": "<string>",
        "number": "<string>",
        "type": "<string>",
        "call_flow_id": "<string>",
        "message_flow_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "status": "<string>",
        "t38_enabled": <boolean>,
        "emergency_enabled": <boolean>,
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The number's unique identifier. Returned when creating a number via ``POST /numbers`` or listing via ``GET /numbers``.
* ``number`` (String, E.164): The phone number in E.164 format (e.g., ``+15551234567``). Must start with ``+``. Virtual numbers use the ``+899`` prefix (e.g., ``+899100000001``).
* ``type`` (enum string): The number's type. See :ref:`Type <number-struct-number-type>`.
* ``call_flow_id`` (UUID): The flow to execute for inbound calls. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
* ``message_flow_id`` (UUID): The flow to execute for inbound messages. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
* ``name`` (String): A human-readable label for the number. Free-form text for organizational use.
* ``detail`` (String): A longer description of the number's purpose or configuration notes.
* ``status`` (enum string): The number's current status. See :ref:`Status <number-struct-number-status>`.
* ``t38_enabled`` (Boolean): Whether T.38 fax support is enabled on this number.
* ``emergency_enabled`` (Boolean): Whether emergency calling (e.g., 911) is enabled on this number.
* ``tm_create`` (string, ISO 8601): Timestamp when the number was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any number property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the number was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the number has not been deleted.

Example
+++++++

.. code::

    {
        "id": "0b266038-844b-11ec-97d8-63ba531361ce",
        "number": "+821100000001",
        "type": "normal",
        "call_flow_id": "d157ce07-0360-4cad-9007-c8ab89fccf9c",
        "message_flow_id": "00000000-0000-0000-0000-000000000000",
        "name": "test talk",
        "detail": "simple number for talk flow",
        "status": "active",
        "t38_enabled": false,
        "emergency_enabled": false,
        "tm_create": "2022-02-01 00:00:00.000000",
        "tm_update": "2022-03-20 19:37:53.135685",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


.. _number-struct-number-type:

Type
----

All possible values for the ``type`` field:

======= ===========
Type    Description
======= ===========
normal  A standard phone number purchased from a provider (Telnyx or Twilio). Routed via PSTN. Supports inbound calls and messages from external callers. Incurs provider purchase and usage charges.
virtual A virtual number with ``+899`` prefix. No provider purchase required. Routed internally within VoIPBIN only. Designed for non-PSTN callers such as AI calls, WebRTC calls, and internal routing. Free to create but subject to tier-based limits.
======= ===========


.. _number-struct-number-status:

Status
------

All possible values for the ``status`` field:

================ ===========
Status           Description
================ ===========
active           The number is provisioned and ready to receive inbound calls and messages. Flows assigned to this number will execute when triggered.
purchase-pending The number purchase has been submitted to the provider but not yet confirmed. This is a transient state for normal numbers only. Poll ``GET /numbers/{id}`` until status changes to ``active``.
suspended        The number is temporarily disabled. Inbound calls and messages will not be handled. Can be reactivated.
deleted          The number has been released and is no longer active. Returned after calling ``DELETE /numbers/{id}``.
================ ===========

