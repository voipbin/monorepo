.. _tag-tutorial:

Tutorial
========

Create a new tag
----------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/tags?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDY4NzQwMzl9.LoPJ9Vv6GFAItYQ1AVV4lrEoOVtJaFOQx-tkauUR1-g' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJwZXJtaXNzaW9uX2lkc1wiOltcIjAzNzk2ZTE0LTdjYjQtMTFlYy05ZGJhLWU3MjAyM2VmZDFjNlwiXSxcInRtX2NyZWF0ZVwiOlwiMjAyMi0wMi0wMSAwMDowMDowMC4wMDAwMDBcIixcInRtX3VwZGF0ZVwiOlwiMjAyMi0wMi0wMiAxODowNToxMC42NTA1MjVcIixcInRtX2RlbGV0ZVwiOlwiOTk5OS0wMS0wMSAwMDowMDowMC4wMDAwMDBcIn0iLCJleHAiOjE2NDY4NzQwMzl9.LoPJ9Vv6GFAItYQ1AVV4lrEoOVtJaFOQx-tkauUR1-g' \
        --data-raw '{
            "name": "test tag",
            "detail": "test tag example"
        }'
