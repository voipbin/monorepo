.. _outdial-struct-outdialtarget:

Outdialtarget
====================

.. _outdial-struct-outdialtarget-outdialtarget:

Outdialtarget
-------------

.. code::

    {
        "id": "<string>",
        "outdial_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "data": "<string>",
        "status": "<string>",
        "destination_0": {
            ...
        },
        "destination_1": {
            ...
        },
        "destination_2": {
            ...
        },
        "destination_3": {
            ...
        },
        "destination_4": {
            ...
        },
        "try_count_0": <number>,
        "try_count_1": <number>,
        "try_count_2": <number>,
        "try_count_3": <number>,
        "try_count_4": <number>,
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: outdialtarget's ID.
* outdial_id: outdial's ID.
* name: outdialtarget's name.
* detail: outdialtarget's detail.
* data: outdialtarget's data.
* *status*: outdialtarget's status. See detail :ref:`here <outdial-struct-outdialtarget-status>`.
* *destination_0*: outdialtarget's destination. See detail :ref:`here <common-struct-address-address>`.
* *destination_1*: outdialtarget's destination. See detail :ref:`here <common-struct-address-address>`.
* *destination_2*: outdialtarget's destination. See detail :ref:`here <common-struct-address-address>`.
* *destination_3*: outdialtarget's destination. See detail :ref:`here <common-struct-address-address>`.
* *destination_4*: outdialtarget's destination. See detail :ref:`here <common-struct-address-address>`.
* try_count_0: destination 0's try count.
* try_count_1: destination 1's try count.
* try_count_2: destination 2's try count.
* try_count_3: destination 3's try count.
* try_count_4: destination 4's try count.

Example
+++++++

.. code::

    {
        "id": "1b3d7a92-7146-466d-90f5-4bc701ada4c0",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "name": "test destination 0",
        "detail": "test detatination 0 detail",
        "data": "test data",
        "status": "done",
        "destination_0": {
            "type": "tel",
            "target": "+821100000001",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "destination_1": null,
        "destination_2": null,
        "destination_3": null,
        "destination_4": null,
        "try_count_0": 1,
        "try_count_1": 0,
        "try_count_2": 0,
        "try_count_3": 0,
        "try_count_4": 0,
        "tm_create": "2022-04-30 17:52:16.484341",
        "tm_update": "2022-04-30 17:53:51.183345",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _outdial-struct-outdialtarget-status:

Status
------
Outdialtarget's status.

=========== ============
Type        Description
=========== ============
idle        The outdialtarget is idle
progressing The outdialtarget is calling
done        The outdialtarget has done to dialing.
=========== ============

**state diagram**

.. image:: _static/images/outdial_struct_outdialtarget_status.png
