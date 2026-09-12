.. _extension-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with extensions, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* An extension number (string) and password for the new SIP device.

.. note:: **AI Implementation Hint**

   When creating an extension, the ``extension`` field and ``username`` field are typically set to the same value. The ``password`` is used for SIP device authentication. After creation, configure SIP devices with the ``username``, ``password``, and the ``domain_name`` returned in the extension response (e.g. ``ab12.reg.voipbin.net``). Always use the returned ``domain_name`` verbatim — do not construct the domain from other fields.

Get list of extensions
----------------------

Gets the list of registered extensions for your account.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/extensions?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "0e7f8158-c770-4930-a98e-f2165b189c1f",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "name": "test domain",
                "detail": "test domain creation",
                "extension": "test11",
                "domain_name": "ab12.reg.voipbin.net",
                "username": "test11",
                "password": "bad79bd2-71e6-11eb-9577-c756bf092a88",
                "direct_hash": "",
                "tm_create": "2021-02-18 12:42:27.688282",
                "tm_update": "",
                "tm_delete": ""
            }
        ],
        "next_page_token": "2021-02-18 12:42:27.688282"
    }

Get detail of specified extension
---------------------------------

Gets the detail of registered extension.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/extensions/0e7f8158-c770-4930-a98e-f2165b189c1f?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "0e7f8158-c770-4930-a98e-f2165b189c1f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "test domain",
        "detail": "test domain creation",
        "extension": "test11",
        "domain_name": "ab12.reg.voipbin.net",
        "username": "test11",
        "password": "bad79bd2-71e6-11eb-9577-c756bf092a88",
        "direct_hash": "",
        "tm_create": "2021-02-18 12:42:27.688282",
        "tm_update": "",
        "tm_delete": ""
    }


Create a extension
------------------

Create a new extension.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/extensions?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test domain",
        "detail": "test domain creation",
        "extension": "test12",
        "password": "27a4d0f2-757c-11eb-bc8f-4f045857b89c"
    }'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "test domain",
        "detail": "test domain creation",
        "extension": "test12",
        "domain_name": "ab12.reg.voipbin.net",
        "username": "test12",
        "password": "27a4d0f2-757c-11eb-bc8f-4f045857b89c",
        "direct_hash": "",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "",
        "tm_delete": ""
    }

Update the extension
--------------------

Update the existing extension with given info.

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "update test extension name",
        "detail": "update test extension detail",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4"
    }'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "update test extension name",
        "detail": "update test extension detail",
        "extension": "test12",
        "domain_name": "ab12.reg.voipbin.net",
        "username": "test12",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4",
        "direct_hash": "",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "2021-02-23 02:11:03.992067",
        "tm_delete": ""
    }

Regenerate direct extension hash
---------------------------------

Regenerate the direct extension hash. This invalidates the previous SIP URI and creates a new one. If the extension has no existing direct hash, one is created automatically. Useful when the existing hash has been compromised or shared unintentionally.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "update test extension name",
        "detail": "update test extension detail",
        "extension": "test12",
        "domain_name": "ab12.reg.voipbin.net",
        "username": "test12",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4",
        "direct_hash": "f7e8d9c0b1a2",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "2021-02-23 02:18:45.334567",
        "tm_delete": ""
    }

.. note:: **AI Implementation Hint**

   This endpoint requires no request body. The ``direct_hash`` in the response is the new hash — the previous hash is permanently invalidated. Update any stored SIP URIs that reference the old hash.

Provision Linphone via QR code
------------------------------

Issue a short-lived provisioning token for the extension, then render the returned ``url`` as a QR code. Scanning the QR code with the Linphone mobile app configures the SIP account automatically. See :ref:`Softphone QR Provisioning <extension-overview-provisioning>` for concepts and security notes.

**Step 1: Issue a provisioning token**

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b/provisioning-token?token=<YOUR_AUTH_TOKEN>'

    {
        "token": "3f9a1c8e5b2d7f4a6c0e9b8d1f3a5c7e2b4d6f8a0c1e3b5d7f9a2c4e6b8d0f1a",
        "url": "https://api.voipbin.net/provisioning/extension?token=3f9a1c8e5b2d7f4a6c0e9b8d1f3a5c7e2b4d6f8a0c1e3b5d7f9a2c4e6b8d0f1a",
        "expire": "2026-08-24T12:10:00Z"
    }

**Step 2: Render the url as a QR code and scan it with Linphone**

Display the ``url`` value as a QR code (any QR library or generator works). In the Linphone mobile app, choose the QR code scan option on the account setup screen and scan the code. Linphone fetches the URL and applies the SIP account settings, then registers.

The URL serves the following Linphone configuration XML (``application/xml``). You do not need to handle this content yourself; Linphone consumes it directly:

.. code::

    <?xml version="1.0" encoding="UTF-8"?>
    <config xmlns="http://www.linphone.org/xsds/lpconfig.xsd">
      <section name="misc">
        <entry name="transient_provisioning" overwrite="true">1</entry>
      </section>
      <section name="proxy_0">
        <entry name="reg_proxy" overwrite="true">&lt;sip:ab12.reg.voipbin.net;transport=udp&gt;</entry>
        <entry name="reg_identity" overwrite="true">sip:test12@ab12.reg.voipbin.net</entry>
        <entry name="reg_expires" overwrite="true">3600</entry>
        <entry name="reg_sendregister" overwrite="true">1</entry>
        <entry name="publish" overwrite="true">0</entry>
      </section>
      <section name="auth_info_0">
        <entry name="username" overwrite="true">test12</entry>
        <entry name="domain" overwrite="true">ab12.reg.voipbin.net</entry>
        <entry name="passwd" overwrite="true">5316382a-757c-11eb-9348-bb32547e99c4</entry>
        <entry name="realm" overwrite="true">ab12.reg.voipbin.net</entry>
      </section>
    </config>

.. note:: **AI Implementation Hint**

   The issuing endpoint requires no request body and needs admin or manager permission. The returned ``url`` is complete; do not reassemble it from the ``token``. The token expires 10 minutes after issuance (``expire``, RFC 3339) and may be fetched multiple times within that window. The public ``GET /provisioning/extension`` endpoint has no ``/v1.0`` prefix and no authentication; any invalid or expired token returns a bare ``400`` with an empty body. The QR code works only with the Linphone app, and scanning on an app that already has a SIP account replaces account 0.

Delete the extension
--------------------

Delete the existing extension of given id.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b?token=<YOUR_AUTH_TOKEN>'

