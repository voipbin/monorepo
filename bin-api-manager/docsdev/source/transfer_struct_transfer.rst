.. _transfer-struct-transfer:

Transfer Struct
===============

.. _transfer-struct-transfer-transfer:

Transfer
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "type": "<string>",
        "transferer_call_id": "<string>",
        "transferee_addresses": [],
        "transferee_call_id": "<string>",
        "groupcall_id": "<string>",
        "confbridge_id": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The transfer's unique identifier.
* ``customer_id`` (UUID): The customer who owns this transfer. Obtained from the ``id`` field of ``GET /customers``.
* ``type`` (enum string): The type of call transfer. See :ref:`Type <transfer-struct-transfer-type>`.
* ``transferer_call_id`` (UUID): The call ID of the party initiating the transfer. Obtained from the ``id`` field of ``GET /calls``.
* ``transferee_addresses`` (array of Address): The destination addresses for the transfer target. Each address follows the :ref:`Address <common-struct-address>` format.
* ``transferee_call_id`` (UUID): The call ID of the party receiving the transfer (created after transfer execution). Obtained from the ``id`` field of ``GET /calls``.
* ``groupcall_id`` (UUID): The group call ID created for the transfer operation. Obtained from the ``id`` field of ``GET /groupcalls``.
* ``confbridge_id`` (UUID): The conference bridge used during the transfer. Obtained from the ``id`` field of ``GET /conferences``.
* ``tm_create`` (string, ISO 8601): Timestamp when this transfer was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this transfer.
* ``tm_delete`` (string, ISO 8601): Timestamp when this transfer was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Attended transfers allow the transferer to speak with the transferee before completing the transfer. Blind transfers immediately connect the transferee without consultation.

.. _transfer-struct-transfer-type:

Type
----

All possible values for the ``type`` field:

========== ===========
Type       Description
========== ===========
attended   Attended transfer. The transferer can consult with the transferee before completing the transfer.
blind      Blind transfer. The call is immediately transferred to the destination without consultation.
========== ===========

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "attended",
        "transferer_call_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "transferee_addresses": [
            {
                "type": "tel",
                "target": "+12025551234"
            }
        ],
        "transferee_call_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "groupcall_id": "d4e5f6a7-b8c9-0123-defa-234567890123",
        "confbridge_id": "e5f6a7b8-c9d0-1234-efab-345678901234",
        "tm_create": "2024-03-01T10:00:00.000000Z",
        "tm_update": "2024-03-01T10:00:15.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }
