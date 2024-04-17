.. _extension-overview: mediastream_overview

Overview
========
In VoipBin, the media stream feature empowers users to directly control media transmission without relying on SIP (Session Initiation Protocol) signaling. Traditionally, SIP signaling is used to establish, modify, and terminate communication sessions in VoIP systems. However, the media stream feature in VoipBin introduces an alternative method for managing media streams independently of SIP signaling.

With the media stream feature, users can initiate, manipulate, and terminate media streams directly through the VoipBin platform, bypassing the need for SIP signaling. This capability offers several advantages:

Flexibility: Users have greater flexibility in controlling media streams, as they can manage them independently of SIP signaling. This flexibility allows for more dynamic and customizable communication experiences.

Efficiency: By eliminating the dependency on SIP signaling for media control, the media stream feature can streamline the process of initiating and managing media streams. This can lead to more efficient use of resources and reduced latency in media transmission.

Scalability: The media stream feature can enhance the scalability of VoipBin by reducing the overhead associated with SIP signaling for media control. This can support a larger number of concurrent media streams and accommodate higher traffic volumes.

Enhanced User Experience: By enabling direct control over media streams, VoipBin can offer users a more seamless and responsive communication experience. Users can adjust media settings in real-time without the constraints imposed by SIP signaling.

Overall, the media stream feature in VoipBin empowers users with greater control and flexibility in managing media transmission, enhancing the efficiency, scalability, and user experience of the platform. This capability enables a wide range of applications, from real-time communication to multimedia streaming, with minimal reliance on traditional SIP signaling mechanisms.

Available resources
-------------------
Currently, support the below types of resources.
* Call: See detail `here <call-overview>`.
* Conference: See detail `here <conference-overview>`.

Bi-directional streaming
------------------------
Bi-directional streaming allows for simultaneous transmission and reception of media. To establish bi-directional streaming, an additional API call is necessary:

Media stream for call

.. code::

    GET https://api.voipbin.net/v1.0/calls/<call-id>/media_stream?encapsulation=<encapsulation-type>&token=<token>

    https://api.voipbin.net/v1.0/calls/652af662-eb45-11ee-b1a5-6fde165f9226/media_stream?encapsulation=rtp&token=some_token


Media stream for conference

.. code::

    GET https://api.voipbin.net/v1.0/conferences/<conference-id>/media_stream?encapsulation=<encapsulation-type>&token=<token>

    https://api.voipbin.net/v1.0/conferences/1ed12456-eb4b-11ee-bba8-1bfb2838807a/media_stream?encapsulation=rtp&token=some_token


By making this API call, a WebSocket connection is created, facilitating both the reception and transmission of media data. This means that both the server (VoipBin) and the client can send and receive media through the WebSocket connection.

.. code::

    +-----------------+        Websocket         +-----------------+
    |                 |--------------------------|                 |
    |     voipbin     |<----  Media In/Out  ---->|     Client      |
    |                 |--------------------------|                 |
    +-----------------+                          +-----------------+

Uni-directional streaming
-------------------------
Uni-directional streaming involves the transmission of media in one direction only. This can be achieved through the use of a flow action known as "external media start". This flow action initiates the transmission of media from the server to the client, without the need for the client to send media back.
See detail :ref:`here <flow-struct-action-external_media_start>`.

Encapsulation Types
-------------------
There are two encapsulation types supported for media streaming:

* rtp: RTP (Real-time Transport Protocol). This is the standard protocol for transmitting audio and video over IP networks. It provides a means for transporting media streams (e.g., voice) from one endpoint to another.
* sln: Signed Linear Mono. This media stream format does not include headers or padding. ulaw allowed.
* audiosocket: AudioSocket. This is a protocol specific to Asterisk, known as Asterisk's AudioSocket type. It is designed to facilitate simple audio streaming with minimal overhead. More details about AudioSocket can be found in the Asterisk AudioSocket Documentation(https://docs.asterisk.org/Configuration/Channel-Drivers/AudioSocket/).

Codec
------
* VoipBin currently supports the ulaw codec. always 16-bit, 8kHz signed linear mono.
* For AudioSocket, it uses 16-bit, 8kHz, mono PCM (little-endian).
