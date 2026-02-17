.. _route-struct-route:

Route
========

.. _route-struct-route-route:

Route
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "provider_id": "<string>",
        "priority": <integer>,
        "target": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The route's unique identifier. Returned when creating via ``POST /routes`` or listing via ``GET /routes``.
* ``customer_id`` (UUID): The customer this route applies to. Obtained from ``GET /customers``. The special value ``00000000-0000-0000-0000-000000000001`` means this route applies to all customers (system default).
* ``provider_id`` (UUID): The provider used for outbound calls on this route. Obtained from the ``id`` field of ``GET /providers``.
* ``priority`` (Integer): Route priority for failover ordering. Lower values are attempted first (1 = highest priority). Multiple routes with different priorities enable automatic failover.
* ``target`` (String): Target country code prefix for destination matching (e.g., ``+82``, ``+1``). Set to ``"all"`` to match every destination.
* ``tm_create`` (string, ISO 8601): Timestamp when the route was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any route property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the route was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the route has not been deleted.

Example
+++++++

.. code::

    {
        "id": "491b6858-5357-11ed-b753-8fd49cd36340",
        "customer_id": "00000000-0000-0000-0000-000000000001",
        "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "priority": 1,
        "target": "all",
        "tm_create": "2022-10-22 16:16:16.874761",
        "tm_update": "2022-10-22 16:16:16.874761",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

