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

* id: Provider's id.
* name: Provider's name.
* detail: Provider's detail.
* type: Provider's type. See detail :ref:`here <provider-struct-provider-type>`.
* tech_prefix: Tech prefix. Will be attacehd to the beginning of the destination. Valid only the type sip.
* tech_postfix: Tech postfix. Will be attacehd to the end of the destination. Valid only the type sip.
* tech_headers: Key/value fair of SIP headers. Valid only the type sip.

example
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

=========== ============
Type        Description
=========== ============
sip         SIP service provider.
=========== ============
