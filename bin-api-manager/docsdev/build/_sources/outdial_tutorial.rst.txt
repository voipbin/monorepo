.. _outdial-tutorial:

Tutorial
========

Get list of outdials
--------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/outdials?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs'

    {
        "result": [
            {
                "id": "40bea034-1d17-474d-a5de-da00d0861c69",
                "campaign_id": "00000000-0000-0000-0000-000000000000",
                "name": "test outdial",
                "detail": "outdial for test use.",
                "data": "",
                "tm_create": "2022-04-28 01:41:40.503790",
                "tm_update": "9999-01-01 00:00:00.000000",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-04-28 01:41:40.503790"
    }

Get a detail of outdial
-----------------------

.. code::

    curl --location --request GET 'https://api.voipbin.net/v1.0/outdials/40bea034-1d17-474d-a5de-da00d0861c69?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs'

    {
        "id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "campaign_id": "00000000-0000-0000-0000-000000000000",
        "name": "test outdial",
        "detail": "outdial for test use.",
        "data": "",
        "tm_create": "2022-04-28 01:41:40.503790",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Create a new outdial
---------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/outdials?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs' \
        --data-raw '{
            "name": "test outdial",
            "detail": "outdial for test use.",
            "data": "test data"
        }'


Create a new outdialtarget
--------------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/outdials/40bea034-1d17-474d-a5de-da00d0861c69/targets?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs' \
        --data-raw '{
            "name": "test destination 0",
            "detail": "test detatination 0 detail",
            "data": "test data",
            "destination_0": {
                "type": "tel",
                "target": "+821021656521"
            }
        }'

    {
        "id": "1b3d7a92-7146-466d-90f5-4bc701ada4c0",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "name": "test destination 0",
        "detail": "test detatination 0 detail",
        "data": "test data",
        "status": "idle",
        "destination_0": {
            "type": "tel",
            "target": "+821021656521",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "destination_1": null,
        "destination_2": null,
        "destination_3": null,
        "destination_4": null,
        "try_count_0": 0,
        "try_count_1": 0,
        "try_count_2": 0,
        "try_count_3": 0,
        "try_count_4": 0,
        "tm_create": "2022-04-30 17:52:16.484341",
        "tm_update": "2022-04-30 17:52:16.484341",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Get list of outdialtargets
--------------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/outdials/40bea034-1d17-474d-a5de-da00d0861c69/targets?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VueDM4NTN6M2pnMnEueC5waXBlZHJlYW0ubmV0L1wiLFwicGVybWlzc2lvbl9pZHNcIjpbXCIwMzc5NmUxNC03Y2I0LTExZWMtOWRiYS1lNzIwMjNlZmQxYzZcIl0sXCJ0bV9jcmVhdGVcIjpcIjIwMjItMDItMDEgMDA6MDA6MDAuMDAwMDAwXCIsXCJ0bV91cGRhdGVcIjpcIjIwMjItMDQtMTQgMDE6Mjg6NDYuNDU0ODk3XCIsXCJ0bV9kZWxldGVcIjpcIjk5OTktMDEtMDEgMDA6MDA6MDAuMDAwMDAwXCJ9IiwiZXhwIjoxNjUxNTU2OTA2fQ.hQ1WXO7Ionnw7FL9_keqZ2Np__Djm3lkIH5BJl1QSMs'

    {
        "result": [
            {
                "id": "1b3d7a92-7146-466d-90f5-4bc701ada4c0",
                "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
                "name": "test destination 0",
                "detail": "test detatination 0 detail",
                "data": "test data",
                "status": "done",
                "destination_0": {
                    "type": "tel",
                    "target": "+821021656521",
                    "target_name": "",
                    "name": "",
                    "detail": ""
                },
                "destination_1": null,
                "destination_2": null,
                "destination_3": null,
                "destination_4": null,
                "try_count_0": 1,
                "try_count_1": 0,
                "try_count_2": 0,
                "try_count_3": 0,
                "try_count_4": 0,
                "tm_create": "2022-04-30 17:52:16.484341",
                "tm_update": "2022-04-30 17:53:51.183345",
                "tm_delete": "9999-01-01 00:00:00.000000"
            },
            ...
        ],
        "next_page_token": "2022-04-28 01:44:07.212667"
    }
