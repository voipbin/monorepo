.. _glossary:

********
Glossary
********

Terms
=====

.. _glossary-accesskey:

Accesskey
---------
A long-lived authentication credential that can be used instead of JWT tokens. Access keys can be created with specific expiration times and are useful for server-to-server integrations where generating JWT tokens repeatedly is inconvenient.

.. _glossary-action:

Action
------
A single step in a flow that defines what VoIPBIN should do during call or message handling. Examples include ``answer``, ``talk``, ``play``, ``digits_receive``, and ``branch``. Actions are executed sequentially unless interrupted or branched.

.. _glossary-activeflow:

Activeflow
----------
A running instance of a flow. When a flow is attached to an incoming call, outgoing call, or triggered via API, an activeflow is created. The activeflow maintains execution state, variables, and the current action cursor throughout its lifecycle.

.. _glossary-agent:

Agent
-----
A call center agent or representative who handles calls on behalf of the company. Agents have multiple addresses (phone numbers, SIP URIs) and can be assigned to queues based on tags. Agent status (available, unavailable, busy) determines their ability to receive calls.

.. _glossary-branch:

Branch
------
A flow action that enables conditional logic by evaluating variables and directing execution to different action IDs based on the result. Branches enable non-linear flow execution and decision trees.

.. _glossary-call-leg:

Call Leg
--------
In VoIPBIN's architecture, a traditional A-to-B call consists of two separate call legs: A → VoIPBIN (Call 1) and VoIPBIN → B (Call 2). Each leg is tracked independently with its own call ID, allowing for complex call scenarios like transfers and conferences.

.. _glossary-campaign:

Campaign
--------
An outbound calling or messaging operation that targets multiple destinations systematically. Campaigns can be associated with flows and track metrics like service levels and completion rates.

.. _glossary-conference:

Conference
----------
A multi-party call session where multiple participants can communicate simultaneously. Conferences support features like recording, transcription, and media streaming.

.. _glossary-did:

DID (Direct Inward Dialing)
---------------------------
A phone number that routes directly to a specific destination. In VoIPBIN, DIDs are managed through the Number resource and can be associated with flows.

.. _glossary-dtmf:

DTMF (Dual-Tone Multi-Frequency)
--------------------------------
Touch-tone signals generated when pressing phone keypad buttons. Used in IVR systems to capture user input. Also referred to as "digits" in VoIPBIN documentation.

.. _glossary-e164:

E.164
-----
International standard for phone number formatting. All phone numbers in VoIPBIN must use E.164 format: ``+`` followed by country code and number, with no spaces or special characters. Example: ``+16062067563``

.. _glossary-flow:

Flow
----
A template that defines a sequence of actions for handling calls, messages, or other communication events. Flows are reusable and can be attached to numbers, campaigns, or triggered via API.

.. _glossary-flow-fork:

Flow Fork
---------
When certain actions (like ``fetch_flow``, ``queue_join``) execute another flow, the execution "forks" to the new flow. After the forked flow completes, execution returns to the next action after the forking action.

.. _glossary-interrupt-action:

Interrupt Action
----------------
Special actions that can be triggered asynchronously via API at any point during flow execution, overriding the normal sequential flow. Examples include attended transfer, transcribe, recording, and TTS.

.. _glossary-ivr:

IVR (Interactive Voice Response)
--------------------------------
Automated telephone system that interacts with callers through voice prompts and DTMF input. VoIPBIN flows enable building sophisticated IVR systems using actions like ``talk``, ``play``, ``digits_receive``, and ``branch``.

.. _glossary-media-server:

Media Server
------------
Backend infrastructure that handles real-time audio/video processing including transcoding, mixing, echo cancellation, and jitter buffering. VoIPBIN's media servers ensure high-quality media streams.

.. _glossary-pstn:

PSTN (Public Switched Telephone Network)
----------------------------------------
Traditional telephone network for landlines and mobile phones. VoIPBIN connects to PSTN through gateways to enable calls to/from regular phone numbers.

.. _glossary-queue:

Queue
-----
A call holding system that places callers on hold until an available agent is found. Queues execute waiting actions (music, announcements) and search for agents based on matching tags.

.. _glossary-rtp:

RTP (Real-time Transport Protocol)
----------------------------------
Network protocol for transmitting audio and video streams. VoIPBIN uses RTP for media transmission between endpoints.

.. _glossary-sip:

SIP (Session Initiation Protocol)
---------------------------------
Signaling protocol used to establish, manage, and terminate VoIP calls. VoIPBIN supports SIP for call control and integrates with SIP trunks.

.. _glossary-sip-trunk:

SIP Trunk
---------
A custom DNS hostname that accepts SIP traffic for your VoIPBIN account. Enables integration with external SIP systems and PBXes.

.. _glossary-stt:

STT (Speech-to-Text)
--------------------
Technology that converts spoken words into text. Used in VoIPBIN for transcription and AI voice assistant features.

.. _glossary-tts:

TTS (Text-to-Speech)
--------------------
Technology that converts text into spoken audio. VoIPBIN's ``talk`` action uses TTS to generate voice prompts in multiple languages and voices.

.. _glossary-variable:

Variable
--------
Dynamic values that can be referenced in flow actions using ``${variable.name}`` syntax. Variables include system-provided metadata (call info, message content) and custom values set via ``variable_set`` action.

.. _glossary-webhook:

Webhook
-------
HTTP callback that VoIPBIN sends to your server when events occur (call status changes, message received, etc.). Webhooks enable real-time notifications and integrations.

.. _glossary-webrtc:

WebRTC (Web Real-Time Communication)
------------------------------------
Browser-based standard for real-time audio/video communication. VoIPBIN supports WebRTC for in-browser calling without plugins.

.. _timestamp:

Timestamp
---------
All timestamps in VoIPBIN follow the format ``YYYY-MM-DD HH:MM:SS.microseconds`` in UTC timezone. Example: ``2022-05-01 15:10:38.785510878``


Requirement levels indicator
============================
This document strives to adhere to :rfc:`2119`. In particular should be noted that:


#. MUST   This word, or the terms "REQUIRED" or "SHALL", mean that the
   definition is an absolute requirement of the specification.

#. MUST NOT   This phrase, or the phrase "SHALL NOT", mean that the
   definition is an absolute prohibition of the specification.

#. SHOULD   This word, or the adjective "RECOMMENDED", mean that there
   may exist valid reasons in particular circumstances to ignore a
   particular item, but the full implications must be understood and
   carefully weighed before choosing a different course.

#. SHOULD NOT   This phrase, or the phrase "NOT RECOMMENDED" mean that
   there may exist valid reasons in particular circumstances when the
   particular behavior is acceptable or even useful, but the full
   implications should be understood and the case carefully weighed
   before implementing any behavior described with this label.

#. MAY   This word, or the adjective "OPTIONAL", mean that an item is
   truly optional.  One vendor may choose to include the item because a
   particular marketplace requires it or because the vendor feels that
   it enhances the product while another vendor may omit the same item.
   An implementation which does not include a particular option MUST be
   prepared to interoperate with another implementation which does
   include the option, though perhaps with reduced functionality. In the
   same vein an implementation which does include a particular option
   MUST be prepared to interoperate with another implementation which
   does not include the option (except, of course, for the feature the
   option provides.)

#. Guidance in the use of these Imperatives

   Imperatives of the type defined in this memo must be used with care
   and sparingly.  In particular, they MUST only be used where it is
   actually required for interoperation or to limit behavior which has
   potential for causing harm (e.g., limiting retransmisssions)  For
   example, they must not be used to try to impose a particular method
   on implementors where the method is not required for
   interoperability.

#. Security Considerations

   These terms are frequently used to specify behavior with security
   implications.  The effects on security of not implementing a MUST or
   SHOULD, or doing something the specification says MUST NOT or SHOULD
   NOT be done may be very subtle. Document authors should take the time
   to elaborate the security implications of not following
   recommendations or requirements as most implementors will not have
   had the benefit of the experience and discussion that produced the
   specification.
