.. _chat-tutorial:

Tutorial
========

Get list of chats
-----------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/chats?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2NDA0MDAwOX0.s2lRAtDRSn82FBOAC_w_hNWRyBWN-2boe4Pq76EdIjI'

    {
        "result": [
            {
                "id": "e8b2e976-f043-44c8-bb89-e214e225e813",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "type": "normal",
                "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
                "participant_ids": [
                    "47fe0b7c-7333-46cf-8b23-61e14e62490a",
                    "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"
                ],
                "name": "test chat normal",
                "detail": "test chat with agent 1 and agent2",
                "tm_create": "2022-09-22 02:41:44.884828",
                "tm_update": "2022-09-22 02:41:44.884828",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-09-22 02:41:44.884828"
    }

Get detail of chat
-------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/chats/e8b2e976-f043-44c8-bb89-e214e225e813?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2NDA0MDAwOX0.s2lRAtDRSn82FBOAC_w_hNWRyBWN-2boe4Pq76EdIjI'

    {
        "id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "normal",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "participant_ids": [
            "47fe0b7c-7333-46cf-8b23-61e14e62490a",
            "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"
        ],
        "name": "test chat normal",
        "detail": "test chat with agent 1 and agent2",
        "tm_create": "2022-09-22 02:41:44.884828",
        "tm_update": "2022-09-22 02:41:44.884828",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Create a new chat
-----------------

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/chats?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2NDA0MDAwOX0.s2lRAtDRSn82FBOAC_w_hNWRyBWN-2boe4Pq76EdIjI' \
        --header 'Content-Type: text/plain' \
        --data-raw '{
            "type": "normal",
            "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "participant_ids": [
                "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
                "47fe0b7c-7333-46cf-8b23-61e14e62490a"
            ],
            "name": "test chat normal",
            "detail": "test chat with agent 1 and agent2"
        }'

    {
        "id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "normal",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "participant_ids": [
            "47fe0b7c-7333-46cf-8b23-61e14e62490a",
            "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"
        ],
        "name": "test chat normal",
        "detail": "test chat with agent 1 and agent2",
        "tm_create": "2022-09-22 02:41:44.884828",
        "tm_update": "2022-09-22 02:41:44.884828",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Get list of chatrooms
---------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/chatrooms?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2NDA0MDAwOX0.s2lRAtDRSn82FBOAC_w_hNWRyBWN-2boe4Pq76EdIjI&owner_id=eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b'

    {
        "result": [
            {
                "id": "1e385680-0f41-4e2a-b154-a61c62bf830a",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "type": "normal",
                "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
                "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
                "participant_ids": [
                    "47fe0b7c-7333-46cf-8b23-61e14e62490a",
                    "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"
                ],
                "name": "test chat normal",
                "detail": "test chat with agent 1 and agent2",
                "tm_create": "2022-09-22 02:41:45.237021",
                "tm_update": "2022-09-22 02:41:45.237021",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-09-22 02:41:45.237021"
    }

Send chatmessage
----------------
Send a message to the chat.

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/chatmessages?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjdXN0b21lciI6IntcImlkXCI6XCI1ZTRhMDY4MC04MDRlLTExZWMtODQ3Ny0yZmVhNTk2OGQ4NWJcIixcInVzZXJuYW1lXCI6XCJhZG1pblwiLFwibmFtZVwiOlwiYWRtaW5cIixcImRldGFpbFwiOlwiYWRtaW4gYWNjb3VudFwiLFwid2ViaG9va19tZXRob2RcIjpcIlBPU1RcIixcIndlYmhvb2tfdXJpXCI6XCJodHRwczovL2VuN2V2YWp3aG1xYnQueC5waXBlZHJlYW0ubmV0XCIsXCJsaW5lX3NlY3JldFwiOlwiYmE1ZjA1NzVkODI2ZDViNGEwNTJhNDMxNDVlZjEzOTFcIixcImxpbmVfdG9rZW5cIjpcInRzZklpREIvMmNHSTVzSFJNSW9wN1MzU1M0S3NiRWxKL3VrUUtzNkxwSFkxWG9HMnBUTUhxZGl5TE51OGFNZGEycGkzdlRYc2NDS3A4WEdFdmZsNmRtSVQxbmZUVGRNa21ZODRpUkxJT0lBbDg1aUcvWFp1ZUkxV0JSdmNoZlY4VGxad0RtRUNiU1N6TCtXdXYrak8rZ2RCMDR0ODkvMU8vdzFjRG55aWxGVT1cIixcInBlcm1pc3Npb25faWRzXCI6W1wiMDM3OTZlMTQtN2NiNC0xMWVjLTlkYmEtZTcyMDIzZWZkMWM2XCJdLFwidG1fY3JlYXRlXCI6XCIyMDIyLTAyLTAxIDAwOjAwOjAwLjAwMDAwMFwiLFwidG1fdXBkYXRlXCI6XCIyMDIyLTA2LTE2IDA4OjM3OjE2Ljk1MjczOFwiLFwidG1fZGVsZXRlXCI6XCI5OTk5LTAxLTAxIDAwOjAwOjAwLjAwMDAwMFwifSIsImV4cCI6MTY2NDY5Njk1Nn0.Wztq5xC4CjyPoO4tsqBNq3-Nwfs1_lWn__3QUZejWY8' \
        --header 'Content-Type: text/plain' \
        --data-raw '{
            "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
            "source": {
                "type": "agent",
                "target": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"
            },
            "type": "normal",
            "text": "test message"
        }'

    {
        "id": "2b4acb7b-f1ba-43c5-ae43-0435a07d55ea",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "source": {
            "type": "agent",
            "target": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "type": "normal",
        "text": "test message",
        "medias": [],
        "tm_create": "2022-09-25 13:11:59.075363",
        "tm_update": "2022-09-25 13:11:59.075363",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
