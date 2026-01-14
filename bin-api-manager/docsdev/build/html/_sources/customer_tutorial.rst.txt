.. _customer-tutorial:

Tutorial
========

Get list of customers
----------------------

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/customers?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY1NzAwNzU3NH0.huSlYAhXhpZzvD_IcSbC4XLL28g_shd74NK4UHAkdd8'

    {
        "result": [
            {
                "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "username": "admin",
                "name": "admin",
                "detail": "admin account",
                "webhook_method": "POST",
                "webhook_uri": "https://en7evasdjwhmqbt.x.pipedream.net",
                "line_secret": "ba5fsf0575d826d5b4asdf052a43145ef1391",
                "line_token": "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdisyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTsdfsfTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
                "permission_ids": [
                    "03796e14-7cb4-11ec-9dba-e72023efd1c6"
                ],
                "tm_create": "2022-02-01 00:00:00.000000",
                "tm_update": "2022-06-16 08:37:16.952738",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-02-01 00:00:00.000000"
    }

Get detail of customer
----------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/customers/5e4a0680-804e-11ec-8477-2fea5968d85b?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2Nzc4ODg2OX0.ZI8v3vgBaUQq7Qemlbb0m3hNEtacYzRHtEX98GCRTL0'

    {
        "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "username": "admin",
        "name": "admin",
        "detail": "admin account",
        "webhook_method": "POST",
        "webhook_uri": "https://en7evajwhmqbt.x.pipedream.net",
        "line_secret": "ba5f0575d826d5b4a0512a145ef1391",
        "line_token": "tsfIiDB/2cGI5sssaIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
        "permission_ids": [
            "03796e14-7cb4-11ec-9dba-e72023efd1c6"
        ],
        "tm_create": "2022-02-01 00:00:00.000000",
        "tm_update": "2022-06-16 08:37:16.952738",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Create a new customer
---------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/customers?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2Nzc4ODg2OX0.ZI8v3vgBaUQq7Qemlbb0m3hNEtacYzRHtEX98GCRTL0' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "username": "test1",
            "password": "ee5f3d14-5ac6-11ed-808e-6f7d676a444b",
            "name": "test 1",
            "detail": "test user 1",
            "webhook_method": "POST",
            "webhook_uri": "https://en7evajwhm11qbt.x.pipedream.net",
            "line_secret": "ba5f0575d826d5b4a051112a145ef1391",
            "line_token": "tsfIiDB/2cGI5sssaIop7S311SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
            "permission_ids": []
        }'

    {
        "id": "ff424526-f65d-483f-bc36-3b2357c6c6a9",
        "username": "test1",
        "name": "test 1",
        "detail": "test user 1",
        "webhook_method": "POST",
        "webhook_uri": "https://en7evajwhm11qbt.x.pipedream.net",
        "line_secret": "ba5f0575d826d5b4a051112a145ef1391",
        "line_token": "tsfIiDB/2cGI5sssaIop7S311SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
        "permission_ids": [],
        "tm_create": "2022-11-02 15:57:08.368093",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Delete customer
---------------

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/customers/ff424526-f65d-483f-bc36-3b2357c6c6a9?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2Nzc4ODg2OX0.ZI8v3vgBaUQq7Qemlbb0m3hNEtacYzRHtEX98GCRTL0' \

    {
        "id": "ff424526-f65d-483f-bc36-3b2357c6c6a9",
        "username": "test1",
        "name": "test 1",
        "detail": "test user 1",
        "webhook_method": "POST",
        "webhook_uri": "https://en7evajwhm11qbt.x.pipedream.net",
        "line_secret": "ba5f0575d826d5b4a051112a145ef1391",
        "line_token": "tsfIiDB/2cGI5sssaIop7S311SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
        "permission_ids": [],
        "tm_create": "2022-11-02 15:57:08.368093",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "2022-11-02 15:59:08.368093"
    }
