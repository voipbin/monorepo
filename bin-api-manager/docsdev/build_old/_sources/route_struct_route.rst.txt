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

* id: Provider's id.
* customer_id: Customer's id. If the customer id is "00000000-0000-0000-0000-000000000001", then it used for every customer.
* provider_id: Provider's id.
* priority: Route priority.
* target: Target country code. If it set to "all", it used for every destination target.

example
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

