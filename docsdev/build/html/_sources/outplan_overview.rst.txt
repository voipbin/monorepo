.. _outplan-overview:

Overview
========
The VoIPBIN's outplan API provides a powerful tool for managing and customizing dialing strategies for outdial operations. The outplan defines how the system should handle dialing attempts and retries when making outbound calls to various destinations.

With the outplan API, users can create and configure multiple outplans, each with its own unique dialing strategy. The outplan allows users to set parameters such as dial timeout, try interval, and maximum try counts for different dialing attempts.

The outplan API is commonly used in conjunction with the outdial API to manage the dialing behavior of outbound calling campaigns. Users can apply specific outplans to different outdials, allowing for targeted and customized dialing strategies for different destinations or scenarios.

By utilizing the outplan API, businesses and developers can optimize their outbound calling operations, increase the success rate of dialing attempts, and effectively manage retries and timeouts. This flexibility and customization provided by the outplan API make it a valuable tool for organizations looking to streamline their outbound communication processes and enhance their customer engagement efforts.

Key features
------------
The outplan API offers several key features that allow users to finely tune the dialing behavior of their outbound calling campaigns:

* Dial Timeout: Users can specify the maximum duration that the system will wait for a call to be answered before considering it a timeout and marking it as such.
* Try Interval: The outplan API allows users to define the interval between consecutive dialing attempts. This feature gives users control over how often the system should retry dialing a destination if the call is not successful on the first attempt.
* Maximum Try Counts: Users can set the maximum number of times the system should try to connect to a destination. This parameter enables users to customize the level of persistence in attempting to establish a successful call.

