.. _auth-main:

****
Auth
****
The Auth API covers the full identity lifecycle outside normal resource CRUD: self-service signup, email verification, credential-based login (token/boot), password recovery, self-service account deletion (unregister), and superadmin delegate access. Every endpoint in this group lives under ``/auth`` (no ``/v1.0`` prefix) and most are unauthenticated by design, since they exist to *establish* authentication in the first place.

**API Reference:** `Auth endpoints <https://api.voipbin.net/redoc/#tag/Auth>`_

.. note:: **AI Implementation Hint**

   For step-by-step walkthroughs with example ``curl`` commands, see the :ref:`Signup quickstart <quickstart-signup>` and :ref:`Authentication quickstart <quickstart-authentication>`. This page is the field-by-field reference: exact request/response schemas, validation rules, and error causes for every ``/auth/*`` endpoint.

.. toctree::
   :maxdepth: 2

   auth_overview
