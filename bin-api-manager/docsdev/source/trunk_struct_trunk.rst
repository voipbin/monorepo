.. _trunk-struct-trunk:

Trunk
=====

.. _trunk-struct-trunk-trunk:

Trunk
-----

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "domain_name": "<string>",
        "auth_types": [],
        "username": "<string>",
        "password": "<string>",
        "allowed_ips": [],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The trunk's unique identifier. Returned when creating via ``POST /trunks`` or listing via ``GET /trunks``.
* ``customer_id`` (UUID): The customer who owns this trunk. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (string): A human-readable name for this trunk.
* ``detail`` (string): Additional description or notes about this trunk.
* ``domain_name`` (string): The SIP domain name for this trunk. Must match the pattern ``^([a-zA-Z]{1})([a-zA-Z0-9\-\.]{1,30})$``.
* ``auth_types`` (array of enum string): The authentication methods configured for this trunk. See :ref:`Auth Types <trunk-struct-trunk-auth-types>`.
* ``username`` (string): The SIP authentication username (for basic authentication).
* ``password`` (string): The SIP authentication password (for basic authentication).
* ``allowed_ips`` (array of string): IP addresses allowed for IP-based authentication. Each entry is an IPv4 address (e.g., ``203.0.113.1``).
* ``tm_create`` (string, ISO 8601): Timestamp when this trunk was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this trunk.
* ``tm_delete`` (string, ISO 8601): Timestamp when this trunk was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   A trunk can support multiple authentication types simultaneously (e.g., both ``basic`` and ``ip``). When creating a trunk with IP-based authentication, provide the ``allowed_ips`` array. When creating with basic authentication, provide ``username`` and ``password``.

.. _trunk-struct-trunk-auth-types:

Auth Types
----------

All possible values in the ``auth_types`` array:

====== ===========
Type   Description
====== ===========
basic  Username and password authentication
ip     IP address-based authentication
====== ===========

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Primary Carrier",
        "detail": "Main PSTN trunk for outbound calls",
        "domain_name": "carrier.example.com",
        "auth_types": [
            "basic",
            "ip"
        ],
        "username": "trunk_user",
        "password": "trunk_pass",
        "allowed_ips": [
            "203.0.113.1",
            "203.0.113.2"
        ],
        "tm_create": "2024-03-01T10:00:00.000000Z",
        "tm_update": "2024-03-01T10:00:00.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }
