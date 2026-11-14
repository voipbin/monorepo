.. _number-provider-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. These endpoints change settings only. A charge happens when a number is purchased through ``POST https://api.voipbin.net/v1.0/numbers``.
   * **Async:** No. Every request returns the updated provider immediately.
   * **Access:** ProjectSuperAdmin only. Customer-level tokens, access keys and direct tokens are refused.

The Number Provider API manages the carriers that VoIPBin purchases phone numbers from. It is different from ``/providers``, which manages the SIP trunks used to route calls.

With the Number Provider API you can:

- List the carriers the platform supports and see which of them can serve now
- Enable or disable a carrier for number purchase
- Set the purchase order (a lower ``priority`` is tried first)
- Set the non-secret settings of the carrier (``config``)
- See whether the carrier secret is present (the secret itself is set as an environment variable of the platform, not through this API)


How Providers Are Chosen
------------------------
When a number is purchased, VoIPBin tries the carriers that are *usable* in ascending ``priority`` (ties are ordered by ``provider_name``). A carrier is usable when it is enabled, has its secret in the environment, and has all the ``config`` keys it requires (``twilio`` needs ``sid`` and ``trunk_sid``).

VoIPBin falls back to the next carrier only when the current carrier definitively refused the order. If the outcome of an order is unknown (for example a timeout), the purchase stops instead of ordering from another carrier, so a number is never bought twice. To route around a failing carrier, disable it.

A fallback is useful only when the next carrier holds the same number. The available number search currently covers ``telnyx`` only.

.. note:: **AI Implementation Hint**

   ``telnyx`` and ``twilio`` can purchase numbers. ``twilio`` is listed with ``enabled`` false and takes part in a purchase only after its ``TWILIO_AUTH_TOKEN`` is set on the platform, its ``config`` (``sid`` and ``trunk_sid``) is complete (``sid`` can come from ``TWILIO_ACCOUNT_SID``), and an administrator enables it. A carrier without its secret variable cannot purchase, release or search.


Credentials
-----------
The secret of a carrier is read from the environment of ``number-manager`` only: ``TELNYX_API_KEY`` for ``telnyx`` and ``TWILIO_AUTH_TOKEN`` for ``twilio``. A secret that was stored through ``PUT /number_providers/{provider_name}/secret`` is not read and not used. The request is still accepted and changes the stored row only, so it has no effect on purchases, releases and searches. Change the environment variable and restart ``number-manager`` to rotate a secret.

The non-secret ``config`` of the stored setting is used first. A ``config`` key that the stored setting does not have (or has empty) is filled from the environment: ``connection_id`` from ``TELNYX_CONNECTION_ID``, ``profile_id`` from ``TELNYX_PROFILE_ID`` and ``sid`` from ``TWILIO_ACCOUNT_SID``. ``trunk_sid`` has no variable. An unset, empty or unresolved placeholder variable (a value that starts with ``[[``) counts as not set. The ``config`` in a response shows the stored setting only, so a provider that is usable only through the variables shows an empty ``config``. A ``config`` value that is stored hides a changed variable, so check the ``config`` after a change of ``TELNYX_CONNECTION_ID``, ``TELNYX_PROFILE_ID`` or ``TWILIO_ACCOUNT_SID``.

``credential_source`` tells where the secret comes from:

* ``env``: the secret variable of the provider is set.
* ``none``: the secret variable is not set. The carrier is not usable until it is set.

``db`` and ``db_undecryptable`` are no longer returned, because a stored secret is not read. They stay in the list of values so that an older client keeps working.

``has_secret`` tells whether the secret variable holds a value; a stored secret does not count. After a ``DELETE`` of the stored secret, ``has_secret`` stays true while the variable is set. The secret is never returned, logged or sent in an event.

.. warning:: **Changing the account**

   Numbers are released through the account that purchased them. Replacing the secret variable with the token of another account (or, for ``twilio``, the ``sid`` of another account) makes the release of the numbers bought through the previous account fail. Change the account only when no number of that provider remains, or when the new secret belongs to the same account.

Storing a secret through the API still needs the encryption key to be configured on the platform. Without it the request is answered with ``503 Service Unavailable``. The stored secret is not used.


Errors
------
.. list-table::
   :header-rows: 1

   * - Status
     - Meaning
   * - 400
     - The request is invalid: a body with none of ``enabled``, ``priority`` and ``config``, a ``priority`` outside 0 to 1000000, a ``config`` key the provider does not allow, more than 8 ``config`` keys or a value that is too long, a ``config`` change that removes a required key while a secret is stored, a ``secret`` that is empty or longer than 1024 bytes, a required ``config`` key that is not set yet when storing a secret, or a provider name that is not lowercase letters, digits and underscores (1 to 32 characters).
   * - 401
     - The request is not authenticated.
   * - 403
     - The caller is not a ProjectSuperAdmin, or uses a direct token.
   * - 404
     - The provider is unknown. ``DELETE /number_providers/{provider_name}/secret`` also answers 404 when no request has created a stored setting for a known provider yet (a rejected secret request can also create one).
   * - 503
     - The platform has no secret encryption key configured, so a secret cannot be stored. This is checked before the required ``config`` keys, so the same request can answer 503 and later 400.

The response of an error does not say which rule failed, so check the list above when you get a 400.


Concurrent changes
------------------
Changes to the ``config`` of the same provider are not serialized. If two administrators change it at the same time, the last write wins.
