.. _conference-tutorial: conference-tutorial

Tutorial
========

Install channel: line
---------------------

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/conversations/setup?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wNS0xMSAxMzoxNzo1MC42ODkyMzNcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NTU3Nzc3NDZ9.9oso_dm-i8U9QMeaCgop87T7PRosYD7gPKyN_xpVBrM' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "reference_type": "line"
        }'


Get list of conferences
-----------------------

Example
+++++++

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/conferences?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM'

    {
        "result": [
            {
                "id": "17039950-eab0-421d-a5f5-05acd1ac6801",
                "user_id": 1,
                "type": "conference",
                "status": "",
                "name": "",
                "detail": "",
                "conferencecall_ids": [],
                "recording_id": "00000000-0000-0000-0000-000000000000",
                "recording_ids": null,
                "tm_create": "2021-02-04 02:55:39.659316",
                "tm_update": "2021-02-04 02:56:07.525985",
                "tm_delete": ""
            },
            ...
        ],
        "next_page_token": "2021-02-03 09:33:58.077756"
    }


Get detail of conference
------------------------

Example
+++++++

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/conferences/0e7112d7-6ddc-47ea-bba5-223a3a55ff79?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM'

    {
        "id": "0e7112d7-6ddc-47ea-bba5-223a3a55ff79",
        "user_id": 1,
        "type": "conference",
        "status": "",
        "name": "",
        "detail": "",
        "conferencecall_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "tm_create": "2021-02-03 10:44:42.163464",
        "tm_update": "2021-02-03 10:52:08.488301",
        "tm_delete": ""
    }


Create a new conference
-----------------------

Example
+++++++

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/conferences?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
        --data-raw '{
            "type": "conference",
            "name": "test conference",
            "detail": "test conference for example"
        }'

    {
        "id": "85252d7b-777b-4580-9420-4df8c6adfc30",
        "user_id": 1,
        "type": "conference",
        "status": "",
        "name": "test conference",
        "detail": "test conference for example",
        "conferencecall_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": null,
        "tm_create": "2021-02-04 03:05:57.710583",
        "tm_update": "",
        "tm_delete": ""
    }

Kick the conferencecall from the conference
-------------------------------------------

Example
+++++++

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/conferencecalls/4833755c-f5d0-4bf2-a101-7d3a7e5e586f?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2MDMxNzcwN30.iEjWGY5LrSe9qR5Wq3SDkGNxCtjYJ9Y43hlHYZbeYVo'

    {
        "id": "4833755c-f5d0-4bf2-a101-7d3a7e5e586f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "conference_id": "99accfb7-c0dd-4a54-997d-dd18af7bc280",
        "reference_type": "call",
        "reference_id": "153c2866-ade0-4a55-a5a7-027e463d9207",
        "status": "leaving",
        "tm_create": "2022-08-09 03:53:49.142446",
        "tm_update": "2022-08-09 03:54:10.035297",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
