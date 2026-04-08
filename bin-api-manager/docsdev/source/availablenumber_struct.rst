.. _availablenumber-struct:

Available Number
================

.. _availablenumber-struct-availablenumber:

Available Number
----------------

.. code::

    {
        "number": "<string>",
        "country": "<string>",
        "region": "<string>",
        "postal_code": "<string>",
        "features": []
    }

* ``number`` (string, E.164): The phone number in E.164 format (e.g., ``+12025551234``). This is the number available for purchase.
* ``country`` (string): The two-letter ISO 3166-1 alpha-2 country code (e.g., ``US``, ``GB``, ``KR``).
* ``region`` (string): The region or state within the country (e.g., ``CA`` for California).
* ``postal_code`` (string): The postal/zip code associated with this number.
* ``features`` (array of enum string): The capabilities supported by this number. See :ref:`Features <availablenumber-struct-features>`.

.. note:: **AI Implementation Hint**

   Available numbers are returned by ``GET /available-numbers``. To purchase a number, use ``POST /numbers`` with the ``number`` value from this response. Not all numbers support all features — check the ``features`` array before purchasing if specific capabilities (e.g., SMS, MMS) are required.

.. _availablenumber-struct-features:

Features
--------

All possible values in the ``features`` array:

=========== ===========
Feature     Description
=========== ===========
emergency   Number supports emergency calling (E911/E112)
fax         Number supports fax transmission
mms         Number supports MMS (multimedia messaging)
sms         Number supports SMS (text messaging)
voice       Number supports voice calls
=========== ===========

Example
-------

.. code::

    {
        "number": "+12025551234",
        "country": "US",
        "region": "DC",
        "postal_code": "20001",
        "features": [
            "voice",
            "sms",
            "mms"
        ]
    }
