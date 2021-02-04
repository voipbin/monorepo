.. _recording-tutorial: recording-tutorial

Tutorial
========

Get list of recordings
----------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/recordings?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTIyMzIxOTcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.of3jiawHOaTaq5t7USc25aVcSag-RhuXfYNdItXrDds&page_size=10&page_token=2021-05-03+21%3A35%3A02.809'

    {
        "result": [
            {
                "id": "348d988b-2ac9-4702-84f0-ae81301ad349",
                "user_id": 1,
                "type": "call",
                "reference_id": "531bc6f4-c695-4c6d-a478-f28b88dfc2ca",
                "status": "ended",
                "format": "wav",
                "filename": "call_531bc6f4-c695-4c6d-a478-f28b88dfc2ca_2021-01-29T05:31:45Z.wav",
                "tm_start": "2021-01-29 05:31:47.870000",
                "tm_end": "2021-01-29 05:31:58.932000",
                "tm_create": "2021-01-29 05:31:45.051136",
                "tm_update": "2021-01-29 05:31:58.943456",
                "tm_delete": ""
            },
            {
                "id": "142e8ef8-392c-4514-abf0-8656da5d2fdf",
                "user_id": 1,
                "type": "call",
                "reference_id": "f457951b-9918-44af-a834-2216b1cc31bc",
                "status": "ended",
                "format": "wav",
                "filename": "call_f457951b-9918-44af-a834-2216b1cc31bc_2021-01-29T03:18:07Z.wav",
                "tm_start": "2021-01-29 03:18:10.790000",
                "tm_end": "2021-01-29 03:18:22.131000",
                "tm_create": "2021-01-29 03:18:07.950164",
                "tm_update": "2021-01-29 03:18:22.144432",
                "tm_delete": ""
            },
            {
                "id": "f27d65bc-2f10-49e1-a49d-a7762965df13",
                "user_id": 1,
                "type": "call",
                "reference_id": "5f7a0eff-9de9-4c41-a018-08bffd4a19aa",
                "status": "ended",
                "format": "wav",
                "filename": "call_5f7a0eff-9de9-4c41-a018-08bffd4a19aa_2021-01-28T09:16:58Z.wav",
                "tm_start": "2021-01-28 09:17:00.814000",
                "tm_end": "2021-01-28 09:17:11.883000",
                "tm_create": "2021-01-28 09:16:58.076735",
                "tm_update": "2021-01-28 09:17:11.890500",
                "tm_delete": ""
            }
        ],
        "next_page_token": "2021-01-28 09:16:58.076735"
    }


Get detail of recording
-----------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/recordings/f27d65bc-2f10-49e1-a49d-a7762965df13?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTIyMzIxOTcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.of3jiawHOaTaq5t7USc25aVcSag-RhuXfYNdItXrDds'

    {
        "id": "f27d65bc-2f10-49e1-a49d-a7762965df13",
        "user_id": 1,
        "type": "call",
        "reference_id": "5f7a0eff-9de9-4c41-a018-08bffd4a19aa",
        "status": "ended",
        "format": "wav",
        "filename": "call_5f7a0eff-9de9-4c41-a018-08bffd4a19aa_2021-01-28T09:16:58Z.wav",
        "tm_start": "2021-01-28 09:17:00.814000",
        "tm_end": "2021-01-28 09:17:11.883000",
        "tm_create": "2021-01-28 09:16:58.076735",
        "tm_update": "2021-01-28 09:17:11.890500",
        "tm_delete": ""
    }


Simple recordingfile download
-----------------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/recordingfiles/348d988b-2ac9-4702-84f0-ae81301ad349?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTIyMzIxOTcsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.of3jiawHOaTaq5t7USc25aVcSag-RhuXfYNdItXrDds' -o tmp.wav

    $ play tmp.wav                                                                                                                                                                                     11s

    tmp.wav:

    File Size: 170k      Bit Rate: 128k
    Encoding: Signed PCM
    Channels: 1 @ 16-bit
    Samplerate: 8000Hz
    Replaygain: off
    Duration: 00:00:10.62

    In:100%  00:00:10.62 [00:00:00.00] Out:85.0k [      |      ] Hd:4.4 Clip:0
    Done.

