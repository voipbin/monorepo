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
* target: The address endpoint. Caller's destinatino address.
* target_name: The address's name. Caller's name.
* name: Name.
* detail: Detail description.

.. _common-struct-address-type:

Type
------------
Defines types of address.

=========== ============
Type        Description
=========== ============
agent       Used for calling to the agent(target must be the agent's id)
endpoint    Used for calling to endpoint(extension@domain)
sip         SIP type address.
tel         Telephone type address.
line        Line type address.
=========== ============

