.. _outdial-struct-outdial:

Outdial
==============

.. _outdial-struct-outdial-outdial:

Outdial
-------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "campaign_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "data": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string or null>",
        "tm_delete": "<string or null>"
    }

* ``id`` (UUID): The outdial's unique identifier. Returned when creating via ``POST /outdials`` or listing via ``GET /outdials``.
* ``customer_id`` (UUID): The customer who owns this outdial. Obtained from the ``id`` field of ``GET /customers``.
* ``campaign_id`` (UUID): The campaign this outdial is attached to. Obtained from the ``id`` field of ``GET /campaigns``. Set to ``00000000-0000-0000-0000-000000000000`` if not attached to any campaign.
* ``name`` (String): Human-readable name for the outdial.
* ``detail`` (String): Detailed description of the outdial's purpose.
* ``data`` (String): Arbitrary data associated with the outdial. Can be used for custom metadata.
* ``tm_create`` (string, ISO 8601): Timestamp when the outdial was created.
* ``tm_update`` (string, ISO 8601, nullable): Timestamp of the last update to any outdial property. ``null`` if never updated.
* ``tm_delete`` (string, ISO 8601, nullable): Timestamp when the outdial was deleted. ``null`` if not deleted.

.. note:: **AI Implementation Hint**

   ``tm_update``/``tm_delete`` are ``null`` when the corresponding event has not occurred, not a sentinel timestamp.

Example
+++++++

.. code::

    {
        "id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "customer_id": "7a1b2c3d-4e5f-6789-abcd-ef0123456789",
        "campaign_id": "00000000-0000-0000-0000-000000000000",
        "name": "test outdial",
        "detail": "outdial for test use.",
        "data": "",
        "tm_create": "2022-04-28 01:41:40.503790",
        "tm_update": null,
        "tm_delete": null
    }
