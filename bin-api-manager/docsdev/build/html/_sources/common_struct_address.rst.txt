.. _common-struct-address:

Address
=======

.. _common-struct-address-address:

Address
-------
Defines source/destination address.

.. code::

    {
        "type": "<string>",
        "target": "<string>",
        "target_name": "<string>",
        "name": "<string>",
        "detail": "<string>"
    }

* *type*: Address type. See detail :ref:`here <common-struct-address-type>`.
* target: The address endpoint. Caller's destination address.
* target_name: The address's name. Caller's name.
* name: Name.
* detail: Detail description.

Example
+++++++

* The tel type address. The target is destination number.

.. code::

  {
    "type": "tel",
    "target": "+821100000001"
  }

* The extension type address. The target is extension number.

.. code::

  {
    "type": "extension",
    "target": "2001"
  }

* The agent type address. The target is agent's id.

.. code::

  {
    "type": "agent",
    "target": "eed6a98a-f18d-11ee-96d3-133e13eafff9"
  }

* The sip type address. The target is sip address.

.. code::

  {
    "type": "sip",
    "target": "testuser@example.com"
  }


.. _common-struct-address-type:

Type
------------
Defines types of address.

=========== ============
Type        Description
=========== ============
agent       Used for calling to the agent(target must be the agent's id)
extension   Used for calling to extension.
sip         SIP type address.
tel         Telephone number type address.
line        Line type address.
=========== ============

