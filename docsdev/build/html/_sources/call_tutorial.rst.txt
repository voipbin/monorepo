.. _call-tutorial: call-tutorial

Tutorial
========

Simple outbound call with TTS
-----------------------------

Making an outbound call with TTS(Text-to-Speech) action.
When the destination answer the call, it will speak the given text message.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDcyNjM5MjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.py7AwXIO0ZNBWSS1PN-05L9oYEREjGgbkkE6CcVyuzw' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+821028286521"
            },
            "destination": {
                "type": "tel",
                "target": "+821021656521"
            },
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "hello. welcome to voipbin. This is test message. This audio file is generated dynamically by the tts module. Please enjoy the voipbin service. Thank you. Bye",
                        "gender": "female",
                        "language": "en-US"
                    }
                }
            ]
        }'


Simple outbound call with media file play
-----------------------------------------

Making an outbound call with media file play action.
When the destination answer the call, it will play the given media file.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTI4NDIyMjcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.OWJihCRfaRtQKtV9fmfgxtpMk6TMQQtq9cSefln7vxM' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "source": {
            "type": "tel",
            "target": "+821028286521"
        },
        "destination": {
            "type": "tel",
            "target": "+821021656521"
        },
        "actions": [
            {
                "type": "play",
                "option": {
                    "stream_urls": [
                        "https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"
                    ]
                }
            }
        ]
    }'

    {
        "id": "a023bfa8-1091-4e94-8eaa-7f01fbecc71a",
        "user_id": 1,
        "flow_id": "f089791a-ac78-4ea0-be88-8a8e131f9fc5",
        "conf_id": "00000000-0000-0000-0000-000000000000",
        "type": "flow",
        "master_call_id": "00000000-0000-0000-0000-000000000000",
        "chained_call_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "source": {
            "type": "tel",
            "target": "+821028286521",
            "name": ""
        },
        "destination": {
            "type": "tel",
            "target": "+821021656521",
            "name": ""
        },
        "status": "dialing",
        "direction": "outgoing",
        "hangup_by": "",
        "hangup_reason": "",
        "tm_create": "2021-02-04 04:44:20.904662",
        "tm_update": "",
        "tm_progressing": "",
        "tm_ringing": "",
        "tm_hangup": ""
    }
