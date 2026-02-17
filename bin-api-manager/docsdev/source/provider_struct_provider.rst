.. _provider-struct-provider:

Provider
========

.. _provider-struct-provider-provider:

Provider
--------

.. code::

    {
        "id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "type": "<string>",
        "hostname": "<string>",
        "tech_prefix": "<string">,
        "tech_postfix": "<string>",
        "tech_headers": {
            "<string>": "<string">,
        },
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The provider's unique identifier. Returned when creating via ``POST /providers`` or listing via ``GET /providers``.
* ``name`` (String): A human-readable label for the provider. Free-form text for organizational use.
* ``detail`` (String): A longer description of the provider's purpose or configuration notes.
* ``type`` (enum string): The provider's protocol type. See :ref:`Type <provider-struct-provider-type>`.
* ``hostname`` (String): The SIP server address for the provider (e.g., ``sip.telnyx.com``). Used as the destination for outbound SIP INVITE messages.
* ``tech_prefix`` (String): Prefix attached to the beginning of the dialed destination number. Valid only for type ``sip``. Leave as empty string if not required.
* ``tech_postfix`` (String): Postfix attached to the end of the dialed destination number. Valid only for type ``sip``. Leave as empty string if not required.
* ``tech_headers`` (Object): Key/value pairs of custom SIP headers sent with outbound calls. Valid only for type ``sip``. Use empty object ``{}`` if not required.
* ``tm_create`` (string, ISO 8601): Timestamp when the provider was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any provider property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the provider was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the provider has not been deleted.

Example
+++++++

.. code::

    {
        "id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "type": "sip",
        "hostname": "sip.telnyx.com",
        "tech_prefix": "",
        "tech_postfix": "",
        "tech_headers": {},
        "name": "telnyx basic",
        "detail": "telnyx basic",
        "tm_create": "2022-10-22 16:16:16.874761",
        "tm_update": "2022-10-24 04:53:14.171374",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _provider-struct-provider-type:

Type
----
Defines types of provider.

=========== ====================================================================
Type        Description
=========== ====================================================================
sip         SIP service provider. Uses Session Initiation Protocol for voice
            call routing to the PSTN. Examples: Telnyx, Twilio, Bandwidth.
=========== ====================================================================
