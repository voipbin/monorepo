.. _conversation-tutorial:

Tutorial
========

Setup the conversation
----------------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/conversations/setup?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wNS0xMSAxMzoxNzo1MC42ODkyMzNcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NTU3Nzc3NDZ9.9oso_dm-i8U9QMeaCgop87T7PRosYD7gPKyN_xpVBrM' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "reference_type": "line"
    }'

Get list of conversations
-------------------------

Example
+++++++

.. code ::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/conversations?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wNS0xMSAxMzoxNzo1MC42ODkyMzNcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NTU3Nzc3NDZ9.9oso_dm-i8U9QMeaCgop87T7PRosYD7gPKyN_xpVBrM'

    {
        "result": [
            {
            "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
            "name": "conversation",
            "detail": "conversation detail",
            "reference_type": "line",
            "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
            "participants": [
                {
                "type": "line",
                "target": "",
                "target_name": "me",
                "name": "",
                "detail": ""
                },
                {
                "type": "line",
                "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "target_name": "Unknown",
                "name": "",
                "detail": ""
                }
            ],
            "tm_create": "2022-06-17 06:06:14.446158",
            "tm_update": "2022-06-17 06:06:14.446167",
            "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-06-17 06:06:14.446158"
    }

Get detail of conversation
--------------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wNS0xMSAxMzoxNzo1MC42ODkyMzNcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NTU3Nzc3NDZ9.9oso_dm-i8U9QMeaCgop87T7PRosYD7gPKyN_xpVBrM'

    {
        "id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "name": "conversation",
        "detail": "conversation detail",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "participants": [
            {
                "type": "line",
                "target": "",
                "target_name": "me",
                "name": "",
                "detail": ""
            },
            {
                "type": "line",
                "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "target_name": "Unknown",
                "name": "",
                "detail": ""
            }
        ],
        "tm_create": "2022-06-17 06:06:14.446158",
        "tm_update": "2022-06-17 06:06:14.446167",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Send a message to the conversation
----------------------------------

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/messages?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wNS0xMSAxMzoxNzo1MC42ODkyMzNcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NTU3Nzc3NDZ9.9oso_dm-i8U9QMeaCgop87T7PRosYD7gPKyN_xpVBrM' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "text": "hi, this is test message. Good to see you. hahaha :)"
        }'

    {
        "id": "0c8f23cb-e878-49bf-b69e-03f59252f217",
        "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
        "status": "sent",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "source_target": "",
        "text": "hi, this is test message. Good to see you. hahaha :)",
        "medias": [],
        "tm_create": "2022-06-20 03:07:11.372307",
        "tm_update": "2022-06-20 03:07:11.372315",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Get list of conversation messages
---------------------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/conversations/a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f/messages?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wNS0xMSAxMzoxNzo1MC42ODkyMzNcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NTU3Nzc3NDZ9.9oso_dm-i8U9QMeaCgop87T7PRosYD7gPKyN_xpVBrM'

    {
        "result": [
            {
                "id": "0c8f23cb-e878-49bf-b69e-03f59252f217",
                "conversation_id": "a7bc12b7-f95c-43e6-82a1-38f4b7ff9b3f",
                "status": "sent",
                "reference_type": "line",
                "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "source_target": "",
                "text": "hi, this is test message. Good to see you. hahaha :)",
                "medias": [],
                "tm_create": "2022-06-20 03:07:11.372307",
                "tm_update": "2022-06-20 03:07:11.372315",
                "tm_delete": "9999-01-01 00:00:00.000000"
            },
            ...
        ],
        "next_page_token": "2022-06-17 06:06:14.948432"
    }
