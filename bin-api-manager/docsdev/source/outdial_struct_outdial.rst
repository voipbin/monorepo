.. _outdial-struct-outdial:

Outdial
==============

.. _outdial-struct-outdial-outdial:

Outdial
-------

.. code::

    {
        "id": "<string>",
        "campaign_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "data": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The outdial's unique identifier. Returned when creating via ``POST /outdials`` or listing via ``GET /outdials``.
* ``campaign_id`` (UUID): The campaign this outdial is attached to. Obtained from the ``id`` field of ``GET /campaigns``. Set to ``00000000-0000-0000-0000-000000000000`` if not attached to any campaign.
* ``name`` (String): Human-readable name for the outdial.
* ``detail`` (String): Detailed description of the outdial's purpose.
* ``data`` (String): Arbitrary data associated with the outdial. Can be used for custom metadata.
* ``tm_create`` (string, ISO 8601): Timestamp when the outdial was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any outdial property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the outdial was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` means the resource has **not** been deleted. This is a sentinel value, not a real timestamp.

Example
+++++++

.. code::

    {
        "id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "campaign_id": "00000000-0000-0000-0000-000000000000",
        "name": "test outdial",
        "detail": "outdial for test use.",
        "data": "",
        "tm_create": "2022-04-28 01:41:40.503790",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
