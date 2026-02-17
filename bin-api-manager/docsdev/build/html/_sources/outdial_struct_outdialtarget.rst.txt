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

* ``id`` (UUID): The outdialtarget's unique identifier. Returned when creating via ``POST /outdials/{id}/targets`` or listing via ``GET /outdials/{id}/targets``.
* ``outdial_id`` (UUID): The parent outdial this target belongs to. Obtained from the ``id`` field of ``GET /outdials``.
* ``name`` (String): Human-readable name for the target (e.g., contact name).
* ``detail`` (String): Detailed description of the target.
* ``data`` (String): Arbitrary data associated with the target. Can be used for custom metadata.
* ``status`` (enum string): The outdialtarget's current status. See :ref:`Status <outdial-struct-outdialtarget-status>`.
* ``destination_0`` (Object or null): Primary destination address. See :ref:`Address <common-struct-address-address>`. Set to ``null`` if not configured.
* ``destination_1`` (Object or null): Secondary destination address. See :ref:`Address <common-struct-address-address>`. Set to ``null`` if not configured.
* ``destination_2`` (Object or null): Third destination address. See :ref:`Address <common-struct-address-address>`. Set to ``null`` if not configured.
* ``destination_3`` (Object or null): Fourth destination address. See :ref:`Address <common-struct-address-address>`. Set to ``null`` if not configured.
* ``destination_4`` (Object or null): Fifth destination address. See :ref:`Address <common-struct-address-address>`. Set to ``null`` if not configured.
* ``try_count_0`` (Integer): Current number of dial attempts made to ``destination_0``. Read-only, incremented by the system.
* ``try_count_1`` (Integer): Current number of dial attempts made to ``destination_1``. Read-only.
* ``try_count_2`` (Integer): Current number of dial attempts made to ``destination_2``. Read-only.
* ``try_count_3`` (Integer): Current number of dial attempts made to ``destination_3``. Read-only.
* ``try_count_4`` (Integer): Current number of dial attempts made to ``destination_4``. Read-only.
* ``tm_create`` (string, ISO 8601): Timestamp when the outdialtarget was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any outdialtarget property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the outdialtarget was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Each outdialtarget supports up to 5 destinations (``destination_0`` through ``destination_4``). The campaign dials destinations in order, starting with ``destination_0``. When all retries for a destination are exhausted (``try_count_N`` reaches the outplan's ``max_try_count_N``), it moves to the next destination. A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` is a sentinel meaning the resource has **not** been deleted.

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
The outdialtarget's current processing status. This is a read-only field managed by the system.

=========== ============
Type        Description
=========== ============
idle        The outdialtarget is idle and available to be dialed. This is the initial state after creation and the state it returns to between retry attempts.
progressing The outdialtarget is currently being dialed. A campaigncall is active for this target.
done        The outdialtarget has completed processing. Either the call was answered successfully or all retry attempts have been exhausted.
=========== ============

**state diagram**

.. image:: _static/images/outdial_struct_outdialtarget_status.png
