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
- Store the carrier credentials (write-only) and the non-secret settings of the carrier
- Delete the stored credentials to return to the credentials of the platform environment


How Providers Are Chosen
------------------------
When a number is purchased, VoIPBin tries the carriers that are *usable* in ascending ``priority`` (ties are ordered by ``provider_name``). A carrier is usable when it is enabled, has a credential, and (for a stored credential) has all the ``config`` keys it requires.

VoIPBin falls back to the next carrier only when the current carrier definitively refused the order. If the outcome of an order is unknown (for example a timeout), the purchase stops instead of ordering from another carrier, so a number is never bought twice. To route around a failing carrier, disable it.

A fallback is useful only when the next carrier holds the same number. The available number search currently covers ``telnyx`` only.

.. note:: **AI Implementation Hint**

   Only ``telnyx`` can purchase numbers today. ``twilio`` is listed with ``enabled`` false and cannot purchase; even when it is enabled and ``usable`` is true, it is never used for purchase.


Credentials
-----------
A carrier uses one credential source at a time:

* ``db``: the secret stored through ``PUT /number_providers/{provider_name}/secret``, together with the ``config`` of the provider.
* ``env``: the credentials of the platform environment, used while no secret is stored.
* ``none``: no credential exists.
* ``db_undecryptable``: a secret is stored but cannot be decrypted (for example the encryption key was removed). The provider is unusable and is **not** replaced by the environment credentials. Restore the key, or delete the stored secret with ``DELETE /number_providers/{provider_name}/secret``.

The secret is write-only. It is never returned, logged or sent in an event. A response only tells ``has_secret``, the ``key_version`` of the key that encrypted it, and who changed it.

.. warning:: **Changing the account**

   Numbers are released through the account that purchased them. Replacing the stored secret with the token of another account, or deleting the stored secret so the environment account is used, makes the release of the numbers bought through the previous account fail. Change the account only when no number of that provider remains, or when the new secret belongs to the same account.

Storing a secret needs the encryption key to be configured on the platform. Without it the request is answered with ``503 Service Unavailable``.


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
