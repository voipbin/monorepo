# VoIPBin
Welcome to voipbin project.
This repository serves as a monorepo for all VoIPbin backend services. 

It provides a centralized location for managing and developing various backend components that power the VoIPbin platform.

# Demo
* https://admin.voipbin.net/ : Console admin page.
* https://talk.voipbin.net/ : Simple voipbin agent page.
 
# Features

## Core Services

* Authentication Service: This service manages user accounts. It handles tasks like user registration, login, password resets, and session management. It ensures only authorized users can access VoIPbin features.
* Call Routing Service: The heart of VoIPbin's call functionality, this service routes incoming and outgoing calls based on user configurations and network conditions. It efficiently connects callers and ensures smooth call establishment.
* Media Processing Service: This service handles the processing of audio and video data during calls. It might involve tasks like codec conversion, noise cancellation, echo cancellation, and media mixing to optimize call quality.
* Recording Service (Optional): If VoIPbin offers call recording features, this service manages the recording process. It allows users to record calls for later playback or reference.
* Billing Service (Optional): For VoIPbin models with paid plans, this service keeps track of user accounts and billing information. It might handle tasks like subscription management, call charges, and invoice generation.

## Benefits of VoIPbin Services

* Seamless Communication: The backend services work together to enable seamless voice and video calls between users.
* Scalability and Reliability: The architecture is designed to handle a high volume of calls and ensure platform reliability.
* Security: Security measures are likely implemented within these services to protect user data and ensure call privacy.

## Additional Considerations
This is a general overview, and the specific functionalities offered by each service might vary depending on VoIPbin's features.
VoIPbin likely offers APIs that developers can integrate with to leverage these backend services in their applications.

# Merge existing projects
The monorepo concist of many sub projects. Most of projects were merged from the existing projects using the following command.
```
$ git subtree add -P <destination repository directory> <source repository> <source repository branch name>
```

## Exmaple
```
$ git subtree add -P bin-agent-manager ../../../gitlab/voipbin/bin-manager/agent-manager master
```

# Environment variables
The voipbin uses the environment variables for the k8s deployment.
## CircleCI
```
    CC_AUTHTOKEN_MESSAGEBIRD: Messagebird(bird.com)'s authtoken.
    CC_AUTHTOKEN_OPENAI: OpenAI's authtoken.

    CC_SSL_PRIVKEY_API_BASE64: Base64 encoded SSL Privkey for api server.
    CC_SSL_CERT_API_BASE64: Base64 encoded SSL cert for api server.
    CC_SSL_PRIVKEY_HOOK_BASE64: Base64 encoded SSL Privkey for event hook server.
    CC_SSL_CERT_HOOK_BASE64: Base64 encoded SSL cert for event hook server.

    CC_TWILIO_SID: Twilio's SID
    CC_TWILIO_TOKEN: Twilio's token

    CC_TELNYX_CONNECTION_ID: Telnyx's connection id
    CC_TELNYX_PROFILE_ID: Telnyx's profile id
    CC_TELNYX_TOKEN: Telnyx's token
```

# Links
* http://voipbin.net/ : Voipbin project page
* https://api.voipbin.net/docs/ : Voipbin API documentation
* https://admin.voipbin.net/ : Console admin page.
* https://talk.voipbin.net/ : Simple voipbin agent page.
  