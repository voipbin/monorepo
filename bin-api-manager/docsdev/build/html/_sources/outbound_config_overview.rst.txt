.. _outbound-config-overview:

OutboundConfig Overview
=======================

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free (no billing impact)
   * **Async:** No

OutboundConfig controls per-customer outbound call behaviour:

- ``destination_whitelist`` — ISO 3166 alpha-2 country codes permitted for outbound PSTN calls. **Always enforced** — an empty list denies all PSTN calls. Customers must populate this before making any outbound PSTN call.
- ``codecs`` — Comma-separated codec preference list applied to **SIP outgoing calls** (e.g. ``PCMU,PCMA,G729``). Empty string = server default. **Not applied to PSTN calls** — PSTN trunks negotiate codecs directly with the carrier via SDP.
- ``default_outgoing_source_number_id`` — Fallback source number used when an outgoing call is placed without a valid customer-owned source (e.g., a SIP caller dialling out, or a missing/non-E.164 source). Must be the UUID of a normal active number owned by the same customer; ``00000000-0000-0000-0000-000000000000`` disables the fallback so such calls are rejected.

There is exactly one OutboundConfig per customer. It is **automatically created** (with an empty ``destination_whitelist``) when a customer account is created. All outbound PSTN calls remain blocked until the customer populates ``destination_whitelist``.

API endpoints follow the billing-account pattern — singular path for self-service, plural path for admin:

* ``GET https://api.voipbin.net/v1.0/outbound_config`` — Retrieve own OutboundConfig (authenticated customer, no ID required).
* ``PUT https://api.voipbin.net/v1.0/outbound_config`` — Update own OutboundConfig (authenticated customer, no ID required).
* ``GET https://api.voipbin.net/v1.0/outbound_configs`` — Admin only: list all configs.
* ``GET https://api.voipbin.net/v1.0/outbound_configs/{id}`` — Admin only: get by ID.
* ``PUT https://api.voipbin.net/v1.0/outbound_configs/{id}`` — Admin only: update by ID.
* ``DELETE https://api.voipbin.net/v1.0/outbound_configs/{id}`` — Admin only: delete by ID.
* ``POST https://api.voipbin.net/v1.0/outbound_configs`` — Admin only: create for a specific customer (for cases where auto-create did not fire).

.. note::

   **Deploy-day behaviour:** The Alembic migration ``b43193683818`` automatically creates an empty OutboundConfig for every active customer that does not yet have one. After migration, every customer will have a row with an empty ``destination_whitelist``, meaning all outbound PSTN calls remain blocked until the customer explicitly populates ``destination_whitelist``. No manual intervention is required.

.. note:: **AI Implementation Hint**

   Before placing an outbound PSTN call on behalf of a customer, verify the destination country is in ``destination_whitelist``. If not, add it via ``PUT https://api.voipbin.net/v1.0/outbound_config`` before attempting the call. A missing or empty whitelist causes a ``400 Bad Request`` with the message ``"outbound destination country not whitelisted"``.

Troubleshooting
---------------

* **400 Bad Request** — ``"outbound destination country not whitelisted"``:
    * **Cause:** Destination country not in ``destination_whitelist``, or whitelist is empty.
    * **Fix:** Add the country code via ``PUT https://api.voipbin.net/v1.0/outbound_config`` with ``{"destination_whitelist": ["us", "gb"]}``.

* **404 Not Found** on ``GET /v1.0/outbound_config``:
    * **Cause:** No OutboundConfig has been created for this customer yet.
    * **Fix:** Contact support — OutboundConfig should be auto-created on account creation.
