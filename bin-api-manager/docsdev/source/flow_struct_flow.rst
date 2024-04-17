.. _flow-struct-flow:

Flow
====

.. _flow-struct-flow-flow:

Flow
----

.. code::

    {
        "id": "<string>",
        "type": "flow",
        "name": "test conference_join",
        "detail": "test flow for conference_join",
        "actions": [
            ...
        ],
        "tm_create": "2022-02-03 05:37:48.545532",
        "tm_update": "2022-02-03 06:10:23.604222",
        "tm_delete": "9999-01-01 00:00:00.000000"
    },

* id: Flow's ID.
* type: Flow's types. See detail :ref:`here <flow-struct-flow-type>`.
* name: Flow's name.
* detail: Flow's detail.
* actions: List of actions. See detail :ref:`here <flow-struct-action>`.

**Example**
.. code::

    {
        "id": "ff8e4528-a743-4913-948c-81abaf563f80",
        "type": "flow",
        "name": "test flow for message sending",
        "detail": "test scenario for sending a test message.",
        "actions": [
            {
                "id": "605f5650-ba92-4dcd-bdac-91fcf6260939",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "message_send",
                "option": {
                    "text": "hello, this is test message.",
                    "source": {
                        "type": "tel",
                        "target": "+821021656521"
                    },
                    "destinations": [
                        {
                            "type": "tel",
                            "target": "+31616818985"
                        }
                    ]
                }
            }
        ],
        "tm_create": "2022-03-21 02:11:15.033396",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


.. _flow-struct-flow-type:

Type
----

======================= ==================
type                    Description
======================= ==================
flow                    Normal flow.
======================= ==================
