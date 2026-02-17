.. _common-struct-address:

Address
=======

.. _common-struct-address-address:

Address
-------
Defines source/destination address. This structure is used throughout VoIPBIN wherever communication endpoints are specified, including calls (``POST /calls``), messages (``POST /messages``), flows (connect/call actions), and number configurations.

.. code::

    {
        "type": "<string>",
        "target": "<string>",
        "target_name": "<string>",
        "name": "<string>",
        "detail": "<string>"
    }

* ``type`` (enum string): Address type. Determines how ``target`` is interpreted. See :ref:`Type <common-struct-address-type>`.
* ``target`` (String): The address endpoint. Format depends on ``type``: E.164 phone number for ``tel`` (e.g., ``+15551234567``), extension number for ``extension`` (e.g., ``2001``), agent UUID for ``agent`` (obtained from ``GET /agents``), or SIP URI for ``sip`` (e.g., ``user@domain.com``).
* ``target_name`` (String): Display name associated with the address. For ``tel`` type, this is the caller ID name. Optional.
* ``name`` (String): A human-readable label for this address. Optional.
* ``detail`` (String): Additional description or metadata for this address. Optional.

.. note:: **AI Implementation Hint**

   When constructing an Address, only ``type`` and ``target`` are required. For ``tel`` type, the ``target`` must be in E.164 format with a leading ``+`` and no spaces or special characters. For ``agent`` type, the ``target`` must be a valid agent UUID obtained from ``GET /agents``.

Example
+++++++

* The ``tel`` type address. The target is a phone number in E.164 format.

.. code::

  {
    "type": "tel",
    "target": "+821100000001"
  }

* The ``extension`` type address. The target is an extension number.

.. code::

  {
    "type": "extension",
    "target": "2001"
  }

* The ``agent`` type address. The target is the agent's UUID, obtained from ``GET /agents``.

.. code::

  {
    "type": "agent",
    "target": "eed6a98a-f18d-11ee-96d3-133e13eafff9"
  }

* The ``sip`` type address. The target is a SIP URI.

.. code::

  {
    "type": "sip",
    "target": "testuser@example.com"
  }


.. _common-struct-address-type:

Type
------------
Defines types of address.

=========== =====================================================================
Type        Description
=========== =====================================================================
agent       Address for calling an agent. ``target`` must be the agent's UUID, obtained from ``GET /agents``.
extension   Address for calling an extension number. ``target`` is the extension number string (e.g., ``2001``).
sip         SIP protocol address. ``target`` is a SIP URI (e.g., ``user@domain.com``).
tel         Telephone number address. ``target`` must be in E.164 format (e.g., ``+15551234567``).
line        Line type address. Used for LINE messaging integrations. ``target`` is the LINE user or channel ID.
=========== =====================================================================

