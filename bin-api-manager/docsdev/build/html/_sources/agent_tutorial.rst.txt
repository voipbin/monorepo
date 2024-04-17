.. _agent-tutorial:

Tutorial
========

Create a new agent
------------------

Create a new agent.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/agents?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDY4NzQwMzl9.LoPJ9Vv6GFAItYQ1AVV4lrEoOVtJaFOQx-tkauUR1-g' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDY4NzQwMzl9.LoPJ9Vv6GFAItYQ1AVV4lrEoOVtJaFOQx-tkauUR1-g' \
        --data-raw '{
            "username": "test2",
            "password": "test2",
            "name": "test tag",
            "detail": "test tag example",
            "ring_method": "ringall",
            "permission": 0,
            "tag_ids": ["d7450dda-21e0-4611-b09a-8d771c50a5e6"]
        }'


Update agent's status
---------------------
Update agent's status to the available.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b/status?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDY4NzQwMzl9.LoPJ9Vv6GFAItYQ1AVV4lrEoOVtJaFOQx-tkauUR1-g' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDYyNDExOTF9.SE9ddn1ZKdZ9zeRpw0a--XUKLvK4JhVU-pBLZCkMXbY' \
        --data-raw '{
            "status": "available"
        }'

Update agent's addresses
------------------------
Update agent's addresses.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/agents/eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b/addresses?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDY4NzQwMzl9.LoPJ9Vv6GFAItYQ1AVV4lrEoOVtJaFOQx-tkauUR1-g' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDY4NzQwMzl9.LoPJ9Vv6GFAItYQ1AVV4lrEoOVtJaFOQx-tkauUR1-g' \
        --data-raw '{
            "addresses": [
                {
                    "type": "tel",
                    "target": "+821021656521"
                }
            ]
        }'
