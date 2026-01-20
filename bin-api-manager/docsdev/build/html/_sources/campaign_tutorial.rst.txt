.. _campaign-tutorial:

Tutorial
========

Get list of campaigns
---------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/campaigns?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUyMjkwNDYyfQ.-jaqJyjISxKmyDxRiFYopD0FA8vlZ_jJ1Sd9mqxCun0'

    {
        "result": [
            {
                "id": "183c0d5c-691e-42f3-af2b-9bffc2740f83",
                "type": "call",
                "name": "test campaign",
                "detail": "test campaign detail",
                "status": "stop",
                "service_level": 100,
                "end_handle": "stop",
                "actions": [
                    {
                        "id": "00000000-0000-0000-0000-000000000000",
                        "next_id": "00000000-0000-0000-0000-000000000000",
                        "type": "talk",
                        "option": {
                            "text": "Hello. This is outbound campaign's test calling. Please wait until the agent answer the call. Thank you.",
                            "gender": "female",
                            "language": "en-US"
                        }
                    }
                ],
                "outplan_id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
                "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
                "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
                "next_campaign_id": "00000000-0000-0000-0000-000000000000",
                "tm_create": "2022-04-28 02:16:39.712142",
                "tm_update": "2022-04-30 17:53:51.685259",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-04-28 02:16:39.712142"
    }


Get detail of campaign
----------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/campaigns/183c0d5c-691e-42f3-af2b-9bffc2740f83?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "183c0d5c-691e-42f3-af2b-9bffc2740f83",
        "type": "call",
        "name": "test campaign",
        "detail": "test campaign detail",
        "status": "stop",
        "service_level": 100,
        "end_handle": "stop",
        "actions": [
            {
                "id": "00000000-0000-0000-0000-000000000000",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "talk",
                "option": {
                    "text": "Hello. This is outbound campaign's test calling. Please wait until the agent answer the call. Thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            }
        ],
        "outplan_id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "next_campaign_id": "00000000-0000-0000-0000-000000000000",
        "tm_create": "2022-04-28 02:16:39.712142",
        "tm_update": "2022-04-30 17:53:51.685259",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Create a new campaign
---------------------

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/campaigns?token=eyJhbGciOiJIUzI1NiIsInR5cCIJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUyMjkwNDYyfQ.-jaqJyjISxKmyDxRiFYopD0FA8vlZ_jJ1Sd9mqxCun0' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test campaign",
        "detail": "test campaign detail",
        "type": "call",
        "service_level": 100,
        "end_handle": "stop",
        "actions": [
            {
                "type": "talk",
                "option": {
                    "text": "Hello. This is outbound campaign's test calling. Please wait until the agent answer the call. Thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            }
        ],
        "outplan_id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb"
    }'
