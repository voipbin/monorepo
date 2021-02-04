.. _flow-tutorial: flow-tutorial

Tutorial
========

Get list of flows
-----------------

Gets the list of registered flows.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/flows?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM'

    {
        "result": [
            {
                "id": "decc2634-0b2a-11eb-b38d-87a8f1051188",
                "name": "default flow",
                "detail": "default flow for voipbin incoming calls",
                "actions": [
                    {
                        "id": "b34aa8a4-0b30-11eb-8016-1f5bc75b1c04",
                        "type": "play",
                        "option": {
                            "stream_url": [
                                "https://github.com/pchero/asterisk-medias/raw/master/voipbin/welcome.wav"
                            ]
                        }
                    },
                    {
                        "id": "57a3dcd2-0b2b-11eb-94a6-a7129b64693c",
                        "type": "play",
                        "option": {
                            "stream_url": [
                                "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
                            ]
                        }
                    }
                ],
                "tm_create": "2020-10-11 01:00:00.000001",
                "tm_update": "",
                "tm_delete": ""
            },
            {
                "id": "af9dae94-ef07-11ea-a101-8f52e568f39b",
                "name": "test flow",
                "detail": "manual flow test",
                "actions": [
                    {
                        "id": "00000000-0000-0000-0000-000000000000",
                        "type": "echo"
                    }
                ],
                "tm_create": "2020-09-04 23:53:14.496918",
                "tm_update": "",
                "tm_delete": ""
            }
        ],
        "next_page_token": "2020-09-04 23:53:14.496918"
    }


Get detail of specified flow
----------------------------

Gets the detail of registered flows.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/flows/decc2634-0b2a-11eb-b38d-87a8f1051188?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM'

    {
        "id": "decc2634-0b2a-11eb-b38d-87a8f1051188",
        "name": "default flow",
        "detail": "default flow for voipbin incoming calls",
        "actions": [
            {
                "id": "b34aa8a4-0b30-11eb-8016-1f5bc75b1c04",
                "type": "play",
                "option": {
                    "stream_url": [
                        "https://github.com/pchero/asterisk-medias/raw/master/voipbin/welcome.wav"
                    ]
                }
            },
            {
                "id": "57a3dcd2-0b2b-11eb-94a6-a7129b64693c",
                "type": "play",
                "option": {
                    "stream_url": [
                        "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
                    ]
                }
            }
        ],
        "tm_create": "2020-10-11 01:00:00.000001",
        "tm_update": "",
        "tm_delete": ""
    }

Create a flow
-------------

Create a new flow for incoming call requests.
When the call is comming, this flow will answer the call first, then will speech the welcome text.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/flows?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test flow",
        "detail": "test voipbin flow example",
        "actions": [
            {
                "type": "answer"
            },
            {
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin'\''s service. thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            }
        ]
    }'

    {
        "id": "24013a0e-d15b-4b5e-9a96-04221a8c6a15",
        "name": "test flow",
        "detail": "test voipbin flow example",
        "actions": [
            {
                "id": "9461bda1-54fd-4e27-ab04-4186c6f72830",
                "type": "answer"
            },
            {
                "id": "69af787e-f5fa-4a1b-9d12-f0b43b86dae6",
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin's service. thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            }
        ],
        "tm_create": "2021-02-04 06:47:01.139361",
        "tm_update": "",
        "tm_delete": ""
    }
