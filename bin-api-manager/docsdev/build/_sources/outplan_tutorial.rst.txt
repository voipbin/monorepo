.. _outplan-tutorial:

Tutorial
========

Get list of outplans
--------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/outplans?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTUwNTQxMjYsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.uV26jlo9kdV-qxxj32cjNa99JRcD96HkFF0h_cuEXLA'

    {
        "result": [
            {
                "id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
                "name": "test outplan",
                "detail": "outplan for test use.",
                "source": {
                    "type": "tel",
                    "target": "+821021656521",
                    "target_name": "",
                    "name": "",
                    "detail": ""
                },
                "dial_timeout": 30000,
                "try_interval": 60000,
                "max_try_count_0": 5,
                "max_try_count_1": 5,
                "max_try_count_2": 5,
                "max_try_count_3": 5,
                "max_try_count_4": 5,
                "tm_create": "2022-04-28 01:50:23.414000",
                "tm_update": "2022-04-30 12:01:13.780469",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-04-28 01:50:23.414000"
    }

Get detail of outplan
---------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/outplans/d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTUwNTQxMjYsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.uV26jlo9kdV-qxxj32cjNa99JRcD96HkFF0h_cuEXLA'

    {
        "id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "name": "test outplan",
        "detail": "outplan for test use.",
        "source": {
            "type": "tel",
            "target": "+821021656521",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "dial_timeout": 30000,
        "try_interval": 60000,
        "max_try_count_0": 5,
        "max_try_count_1": 5,
        "max_try_count_2": 5,
        "max_try_count_3": 5,
        "max_try_count_4": 5,
        "tm_create": "2022-04-28 01:50:23.414000",
        "tm_update": "2022-04-30 12:01:13.780469",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Create a new outplan
--------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/outplans?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "test outplan",
            "detail": "outplan for test use.",
            "source": {
                "type": "tel",
                "target": "+821021656521"
            },
            "dial_timeout": 30000,
            "try_interval": 600000,
            "max_try_count_0": 5,
            "max_try_count_1": 5,
            "max_try_count_2": 5,
            "max_try_count_3": 5,
            "max_try_count_4": 5
        }'

Update outplan's dial info
--------------------------

Example

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/outplans/d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e/dial_info?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+821021656521"
            },
            "dial_timeout": 30000,
            "try_interval": 60000,
            "max_try_count_0": 5,
            "max_try_count_1": 5,
            "max_try_count_2": 5,
            "max_try_count_3": 5,
            "max_try_count_4": 5
        }'

Delete outplan
--------------

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/outplans/88334a03-bc6b-40b6-878f-46df2d9865db?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs'

Update outplan's basic info
---------------------------

Example

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/outplans/d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "test outplan",
            "detail": "outplan for test use"
        }'

    {
        "id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "name": "test outplan",
        "detail": "outplan for test use",
        "source": {
            "type": "tel",
            "target": "+821021656521",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "dial_timeout": 30000,
        "try_interval": 60000,
        "max_try_count_0": 5,
        "max_try_count_1": 5,
        "max_try_count_2": 5,
        "max_try_count_3": 5,
        "max_try_count_4": 5,
        "tm_create": "2022-04-28 01:50:23.414000",
        "tm_update": "2022-05-02 05:59:44.290658",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
