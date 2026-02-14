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

* id: Number's ID.
* number: Number.
* type: Number's type. See detail :ref:`here <number-struct-number-type>`
* call_flow_id: Flow id for incoming call.
* message_flow_id: Flow id for incoming message.
* name: Number's name.
* detail: Number's detail description.
* status: Number's status. See detail :ref:`here <number-struct-number-status>`
* t38_enabled: T38 support.
* emergency_enabled: Emergency call support.

example
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

======= ===========
Type    Description
======= ===========
normal  Normal number purchased from a provider.
virtual Virtual number with +899 prefix. No provider purchase required.
======= ===========


.. _number-struct-number-status:

Status
------

======= ===========
Type    Description
======= ===========
active  Number is being used.
deleted Number has deleted.
======= ===========

