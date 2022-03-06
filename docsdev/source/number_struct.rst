.. _number-struct:

Struct
======

.. _number-struct-number:

Number
------

.. code::

    {
        "id": "<string>",
        "number": "<string>",
        "flow_id": "<string>",
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
* name: Number's name.
* detail: Number's detail description.
* status: Number's status. See detail :ref:`here <number-struct-status>`
* t38_enabled: T38 support.
* emergency_enabled: Emergency call support.

.. _number-struct-status:

Status
------

======= ===========
Type    Description
======= ===========
active  Number is being used.
deleted Number has deleted.
======= ===========

