.. _ai-overview: ai-overview

Overview
========
VoIPBin's AI is a built-in AI agent that enables automated, intelligent voice interactions during live calls. Designed for seamless integration within VoIPBin's flow, the AI utilizes ChatGPT as its AI engine to process and respond to user inputs in real time. This allows developers to create dynamic and interactive voice experiences without requiring manual intervention.

How it works
============

Action component
----------------

The AI is integrated as one of the configurable components within a VoIPBin flow. When a call reaches an AI action, the system triggers the AI to generate a response based on the provided prompt. The response is then processed and played back to the caller using text-to-speech (TTS). If the response is in a structured JSON format, VoIPBin executes the defined actions accordingly.

.. image:: _static/images/ai_overview_overview.png
    :alt: AI component in action builder
    :align: center

TTS/STT + AI Engine
-------------------

VoIPBin's AI is built using TTS/STT + AI Engine, where speech-to-text (STT) converts spoken words into text, and text-to-speech (TTS) converts responses back into audio. The system processes these in real time, enabling seamless conversations.

.. image:: _static/images/ai_overview_stt_tts.png
    :alt: AI implementation using TTS/STT + AI Engine
    :align: center

Voice Detection and Play Interruption:
--------------------------------------
In addition to basic TTS and STT functionalities, VoIPBin incorporates voice detection to create a more natural conversational flow. While the AI is speaking (i.e., playing TTS media), if the system detects the caller's voice, it immediately stops the TTS playback and routes the caller's speech (via STT) to the AI engine. This play interruption feature ensures that if the user starts talking, their input is prioritized, enabling a dynamic interaction that more closely resembles a real conversation.

Context Retention
-----------------
VoIPBin's AI supports context saving. During a conversation, the AI remembers prior exchanges, allowing it to maintain continuity and respond based on earlier parts of the interaction. This provides a more natural and human-like dialogue experience.

Multilingual support
--------------------
VoIPBin's AI supports multiple languages. See supported languages: :ref:`supported languages <transcribe-overview-supported_languages>`.

External AI Agent Integration
-----------------------------
For users who prefer to use external AI services, VoIPBin offers media stream access via MCP (Media Control Protocol). This allows third-party AI engines to process voice data directly, enabling deeper customization and advanced AI capabilities.

MCP Server
----------
A recommended open-source implementation is available here:

* https://github.com/nrjchnd/voipbin-mcp

Using the AI
=================

Initial Prompt
--------------
The initial prompt serves as the foundation for the AI's behavior. A well-crafted prompt ensures accurate and relevant responses. There is no enforced limit to prompt length, but we recommend keeping this confidential to ensure consistent performance and security.

Example Prompt:
+++++++++++++++

.. code::

    Pretend you are an expert customer service agent.

    Please respond kindly.

AI Talk
=======

**AI Talk** enables real-time conversational AI with voice in VoIPBin, powered by **ElevenLabs' voice engine** for natural-sounding speech.

.. image:: _static/images/ai_overview_ai_talk.png
    :alt: AI Talk component in action builder
    :align: center
    :width: 300px

Key Features
------------

* **Real-time Voice Interaction**: AI generates responses in real-time based on user input and delivers them as speech.
* **Interruption Detection & Listening**: If the other party speaks while the AI is talking, the system immediately **stops the AI's speech** and switches to capturing the user's voice via STT.  
  This ensures a smooth and continuous conversation flow.
* **Low Latency Response**: For longer prompts, AI Talk does not wait for the entire response to finish. Instead, it generates and plays speech in smaller chunks, **reducing perceived response time** for the user.
* **ElevenLabs Voice Engine**: High-quality, natural-sounding voice output ensures the AI feels like a real conversation partner.

Built-in ElevenLabs Voice IDs
---------------------------------
VoIPBin uses a predefined set of voice IDs for various languages and genders. Here are the default ElevenLabs Voice IDs currently in use:

=========================== ==================================== =================================== =================================
Language                    Male Voice ID (Name)                 Female Voice ID (Name)              Neutral Voice ID (Name)
=========================== ==================================== =================================== =================================
English (Default)           ``21m00Tcm4TlvDq8ikWAM`` (Adam)      ``EXAVITQu4vr4xnSDxMaL`` (Rachel)   ``EXAVITQu4vr4xnSDxMaL`` (Rachel)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Japanese                    ``Mv8AjrYZCBkdsmDHNwcB`` (Ishibashi) ``PmgfHCGeS5b7sH90BOOJ`` (Fumi)     ``PmgfHCGeS5b7sH90BOOJ`` (Fumi)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Chinese                     ``MI36FIkp9wRP7cpWKPTl`` (Evan)      ``ZL9dtgFhmkTzAHUUtQL8`` (Xiao)     ``ZL9dtgFhmkTzAHUUtQL8`` (Xiao)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
German                      ``uM8iMoqaSe1eDaJiWfxf`` (Felix)     ``nF7t9cuYo0u3kuVI9q4B`` (Dana)     ``nF7t9cuYo0u3kuVI9q4B`` (Dana)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
French                      ``IPgYtHTNLjC7Bq7IPHrm`` (Alexandre) ``SmWACbi37pETyxxMhSpc``            ``SmWACbi37pETyxxMhSpc``
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Hindi                       ``IvLWq57RKibBrqZGpQrC`` (Leo)       ``MF4J4IDTRo0AxOO4dpFR`` (Devi)     ``MF4J4IDTRo0AxOO4dpFR`` (Devi)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Korean                      ``nbrxrAz3eYm9NgojrmFK`` (Minjoon)   ``AW5wrnG1jVizOYY7R1Oo`` (Jiyoung)  ``AW5wrnG1jVizOYY7R1Oo`` (Jiyoung)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Italian                     ``iLVmqjzCGGvqtMCk6vVQ``             ``b8jhBTcGAq4kQGWmKprT`` (Sami)     ``b8jhBTcGAq4kQGWmKprT`` (Sami)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Spanish (Spain)             ``JjHBC66wF58p4ogebCNA`` (Eduardo)   ``UOIqAnmS11Reiei1Ytkc`` (Carolina) ``UOIqAnmS11Reiei1Ytkc`` (Carolina)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Portuguese (Brazil)         ``NdHRjGnnDKGnnm2c19le`` (Tiago)     ``CZD4BJ803C6T0alQxsR7`` (Andreia)  ``CZD4BJ803C6T0alQxsR7`` (Andreia)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Dutch                       ``G53Wkf3yrsXvhoQsmslL`` (James)     ``YUdpWWny7k5yb4QCeweX`` (Ruth)     ``YUdpWWny7k5yb4QCeweX`` (Ruth)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Russian                     ``qJBO8ZmKp4te7NTtYgzz`` (Egor)      ``ymDCYd8puC7gYjxIamPt``            ``ymDCYd8puC7gYjxIamPt``
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Arabic                      ``s83SAGdFTflAwJcAV81K`` (Adeeb)     ``EXAVITQu4vr4xnSDxMaL`` (Farah)    ``4wf10lgibMnboGJGCLrP`` (Farah)
--------------------------- ------------------------------------ ----------------------------------- ---------------------------------
Polish                      ``H5xTcsAIeS5RAykjz57a`` (Alex)      ``W0sqKm1Sfw1EzlCH14FQ`` (Beata)    ``W0sqKm1Sfw1EzlCH14FQ`` (Beata)
=========================== ==================================== =================================== =================================

Other ElevenLabs Voice ID Options
---------------------------------
Voipbin allows you to personalize the text-to-speech output by specifying a custom ElevenLabs Voice ID. By setting the *voipbin.tts.elevenlabs.voice_id* variable, you can override the default voice selection.

..

    voipbin.tts.elevenlabs.voice_id: <Your Custom Voice ID>

See how to set the variables :ref:`here <variable_overview>`.

AI Summary
==========

The AI Summary feature in VoIPBin generates structured summaries of call transcriptions, recordings, or conference discussions. It provides a concise summary of key points, decisions, and action items based on the provided transcription source.

.. image:: _static/images/ai_overview_summary.png
    :alt: AI summary component in action builder
    :align: center

Supported Resources
-------------------

AI summaries work with a single resource at a time. The supported resources are:

Real-time Summary: 
* Live call transcription
* Live conference transcription

Non-Real-time Summary:
* Transcribed recordings (post-call)
* Recorded conferences (post-call)

Choosing Between Real-time and Non-Real-time Summaries
------------------------------------------------------

Developers must decide whether to use a real-time or non-real-time summary based on their needs:

=========================== ============= ==============================================
Use Case                    Summary Type  Recommendation
=========================== ============= ==============================================
Live call monitoring        Real-time     Use AI summary with a live call transcription
--------------------------- ------------- ----------------------------------------------
Live conference insights    Real-time     Use AI summary with a live conference transcription
--------------------------- ------------- ----------------------------------------------
Post-call analysis          Non-real-time Use AI summary with transcribe_id from a completed call
--------------------------- ------------- ----------------------------------------------
Recorded conference summary Non-real-time Use AI summary with recording_id
=========================== ============= ==============================================

AI Summary Behavior
-------------------

* The summary action processes only one resource at a time.
* If multiple AI summary actions are used in a flow, each executes independently.
* If an AI summary action is triggered multiple times for the same resource, it only returns the most recent segment.
* In conference calls, the summary is unified across all participants rather than per speaker.

Ensuring Full Coverage
----------------------

Since starting an AI summary action late in the call results in missing earlier conversations, developers should follow best practices:
* Enable transcribe_start early: This ensures that transcriptions are available even if an AI summary action is triggered later.
* Use transcribe_id instead of call_id: This allows summarizing a full transcription rather than just the latest segment.
* For post-call summaries, use recording_id: This ensures that the full conversation is summarized from the recorded audio.