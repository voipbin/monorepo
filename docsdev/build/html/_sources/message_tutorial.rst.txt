.. _message-tutorial:

Tutorial
========

Send a message
--------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/messages?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDc3ODQ2ODR9.ynnHtevhobwPYoXeDzfhWzaYrOX_kfiNA7zBMFl_nwo' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+821021656521"
        },
        "destinations": [
            {
                "type": "tel",
                "target":"+31616818985"
            }
        ],
        "text": "hello, this is test message."
    }'

Get list of messages
--------------------

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/messages?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDc3ODQ2ODR9.ynnHtevhobwPYoXeDzfhWzaYrOX_kfiNA7zBMFl_nwo&page_size=10'

    {
    "result": [
        {
            "id": "a5d2114a-8e84-48cd-8bb2-c406eeb08cd1",
            "type": "sms",
            "source": {
                "type": "tel",
                "target": "+821028286521",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "targets": [
                {
                    "destination": {
                        "type": "tel",
                        "target": "+821021656521",
                        "target_name": "",
                        "name": "",
                        "detail": ""
                    },
                    "status": "sent",
                    "parts": 1,
                    "tm_update": "2022-03-13 15:11:06.497184184"
                }
            ],
            "text": "Hello, this is test message.",
            "direction": "outbound",
            "tm_create": "2022-03-13 15:11:05.235717",
            "tm_update": "2022-03-13 15:11:06.497278",
            "tm_delete": "9999-01-01 00:00:00.000000"
        },
        ...
    ]
