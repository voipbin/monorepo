# Add Direct Hash RST Documentation

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a standalone RST documentation page explaining the direct hash feature as a cross-cutting platform concept, consolidating information currently scattered across 5 resource docs.

**Architecture:** Single `direct_hash_overview.rst` page added to "Core Concepts" in the docs index, with a `direct_hash.rst` wrapper following the existing pattern (webhook.rst, common.rst). No struct or tutorial pages — regeneration examples already exist in each resource's tutorial.

**Tech Stack:** RST (reStructuredText), Sphinx, existing VoIPBIN doc conventions

---

### Task 1: Create worktree

**Step 1: Create the worktree and switch to it**

```bash
cd ~/gitvoipbin/monorepo
git worktree add ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-direct-hash-rst-documentation -b NOJIRA-Add-direct-hash-rst-documentation
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-direct-hash-rst-documentation
```

Expected: Worktree created, branch checked out.

---

### Task 2: Create the wrapper file `direct_hash.rst`

**Files:**
- Create: `bin-api-manager/docsdev/source/direct_hash.rst`

**Step 1: Create the wrapper file**

```rst
.. _direct-hash-main:

***********
Direct Hash
***********
Generate simplified public SIP URIs for VoIPBIN resources, allowing external callers to reach extensions, agents, conferences, AIs, and teams without knowing customer-specific domains.

.. include:: direct_hash_overview.rst
```

**Step 2: Verify the file exists**

```bash
cat bin-api-manager/docsdev/source/direct_hash.rst
```

Expected: File contents match above.

---

### Task 3: Create `direct_hash_overview.rst`

**Files:**
- Create: `bin-api-manager/docsdev/source/direct_hash_overview.rst`

**Step 1: Create the overview file with full content**

```rst
.. _direct-hash-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low — Direct hash is a routing shortcut; no complex setup required.
   * **Cost:** Free. Direct hash creation and regeneration incur no charges.
   * **Async:** No. Regenerate returns immediately with the updated resource.

Direct hash provides simplified public SIP URIs for VoIPBIN resources. Instead of requiring callers to know a customer-specific domain (e.g., ``sip:office1@abc123.registrar.voipbin.net``), direct hash exposes a short, unique address on a shared domain: ``sip:direct.<hash>@sip.voipbin.net``. This allows external SIP devices, trunks, and partners to reach your resources without any customer-specific configuration.

Five resource types support direct hash: **extensions**, **agents**, **conferences**, **AIs**, and **teams**.

.. note:: **AI Implementation Hint**

   To place a call to a direct hash resource, use the SIP URI ``sip:direct.<hash>@sip.voipbin.net`` as the destination. The ``<hash>`` value is the ``direct_hash`` field from the resource's GET response. If ``direct_hash`` is empty, call the resource's ``direct-hash-regenerate`` endpoint first to create one.


How It Works
------------

**Hash Format**

Each direct hash consists of a ``direct.`` prefix followed by 12 hexadecimal characters generated from 6 cryptographically random bytes.

::

    Format:    direct.<12 hex chars>
    Example:   direct.a1b2c3d4e5f6
    SIP URI:   sip:direct.a1b2c3d4e5f6@sip.voipbin.net

**Routing Flow**

When an external caller dials a direct hash SIP URI, VoIPBIN resolves the hash to the underlying resource and routes the call accordingly.

::

    External Caller                         VoIPBIN                          Destination
         |                                     |                                  |
         | INVITE                               |                                  |
         | sip:direct.<hash>@sip.voipbin.net   |                                  |
         +------------------------------------>|                                  |
         |                                     |                                  |
         |                                     | 1. Lookup hash in database       |
         |                                     | 2. Resolve resource type and ID  |
         |                                     | 3. Route call to resource        |
         |                                     |                                  |
         |                                     | INVITE (to resolved resource)    |
         |                                     +--------------------------------->|
         |                                     |                                  |
         |                                     |           180 Ringing            |
         |                                     |<---------------------------------+
         |                                     |                                  |
         |           Ringback tone             |           200 OK                 |
         |<------------------------------------|<---------------------------------+
         |                                     |                                  |
         |           Call connected            |           Media flow             |
         |<------------------------------------|<-------------------------------->|

**Comparison with Standard URIs**

::

    Standard (requires customer domain knowledge):
    +-----------------------------------------------------------------------+
    | sip:{extension}@{customer-id}.registrar.voipbin.net                   |
    +-----------------------------------------------------------------------+

    Direct hash (public, simplified):
    +-----------------------------------------------------------------------+
    | sip:direct.<hash>@sip.voipbin.net                                     |
    +-----------------------------------------------------------------------+


Supported Resources
-------------------

+---------------+----------------+---------------------------------------------------+-------------------------------------------+
| Resource      | Auto-Created   | Regenerate Endpoint                               | Documentation                             |
+===============+================+===================================================+===========================================+
| Extension     | Yes            | ``POST /extensions/{id}/direct-hash-regenerate``  | :ref:`extension-overview-direct`          |
+---------------+----------------+---------------------------------------------------+-------------------------------------------+
| Conference    | Yes            | ``POST /conferences/{id}/direct-hash-regenerate`` | :ref:`conference-overview`                |
+---------------+----------------+---------------------------------------------------+-------------------------------------------+
| Team          | Yes            | ``POST /teams/{id}/direct-hash-regenerate``       | :ref:`team-overview`                      |
+---------------+----------------+---------------------------------------------------+-------------------------------------------+
| Agent         | No             | ``POST /agents/{id}/direct-hash-regenerate``      | :ref:`agent_overview`                     |
+---------------+----------------+---------------------------------------------------+-------------------------------------------+
| AI            | No             | ``POST /ais/{id}/direct-hash-regenerate``         | :ref:`ai-overview`                        |
+---------------+----------------+---------------------------------------------------+-------------------------------------------+

**Auto-Created** means the ``direct_hash`` field is populated automatically when the resource is created. For resources marked **No**, call the regenerate endpoint to create the initial hash.


Managing Direct Hashes
----------------------

**Creating a Direct Hash**

For extensions, conferences, and teams, a direct hash is generated automatically when the resource is created. For agents and AIs, call the regenerate endpoint to create one:

.. code::

    $ curl -k --location --request POST \
        'https://api.voipbin.net/v1.0/agents/<agent-id>/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

The response contains the full resource with the ``direct_hash`` field populated.

**Regenerating a Direct Hash**

To invalidate the current hash and generate a new one, call the same regenerate endpoint. The old hash is permanently invalidated — any SIP URIs using the old hash will stop working immediately.

.. code::

    $ curl -k --location --request POST \
        'https://api.voipbin.net/v1.0/extensions/<extension-id>/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

No request body is required. The response contains the updated resource with the new ``direct_hash``.

.. note:: **AI Implementation Hint**

   After regenerating a direct hash, update any stored SIP URIs that reference the old hash. The old hash is permanently invalidated — there is no way to restore it. If you manage multiple integrations pointing to the same resource, update all of them before relying on the new hash.


Use Cases
---------

- **External partner access**: Share a simple SIP address (``sip:direct.<hash>@sip.voipbin.net``) with partners or customers who need to reach your resources without configuring customer-specific domains.
- **SIP trunk compatibility**: Allow inbound calls from SIP trunks that cannot be configured with customer-specific domains. The shared ``sip.voipbin.net`` domain works universally.
- **AI agent dial-in**: Provide a public SIP address for an AI voice agent so external callers can reach it directly.
- **Conference bridge access**: Share a direct hash SIP URI as the conference dial-in number for participants.
- **Security rotation**: If a hash is compromised or shared unintentionally, regenerate it immediately. The old hash stops working and a new one is issued.


Security Considerations
-----------------------

- **Treat the hash as a credential.** Anyone with the direct hash SIP URI can initiate calls to the resource. Share it only with intended recipients.
- **Cryptographically random.** Hashes are generated using ``crypto/rand`` (6 bytes / 12 hex characters). They are not guessable or sequential.
- **Regenerate if compromised.** Call the regenerate endpoint to instantly invalidate the old hash and issue a new one. This is atomic — the old hash stops working immediately.
- **No expiration.** Direct hashes remain valid until explicitly regenerated or the resource is deleted.


Troubleshooting
---------------

* **404 when calling a direct hash SIP URI:**
    * **Cause:** The hash does not exist, was regenerated, or the resource was deleted.
    * **Fix:** Retrieve the current resource via ``GET /<resources>/{id}`` and verify the ``direct_hash`` field matches the URI you are dialing.

* **Empty ``direct_hash`` field in resource response:**
    * **Cause:** The resource type does not auto-create direct hashes (agent, AI), and the regenerate endpoint has not been called.
    * **Fix:** Call ``POST /<resources>/{id}/direct-hash-regenerate`` to create the initial hash.

* **Calls not reaching resource after regeneration:**
    * **Cause:** SIP devices or trunks are still configured with the old hash.
    * **Fix:** Update the SIP URI in all devices and trunk configurations to use the new ``direct_hash`` value.

* **404 on the regenerate endpoint itself:**
    * **Cause:** The resource UUID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET /<resources>`` list call with your authentication token.


Related Documentation
---------------------

- :ref:`Extension Overview — Direct Extension <extension-overview-direct>` — Detailed direct extension architecture and SIP flow diagrams
- :ref:`Extension Tutorial <extension-tutorial>` — Extension CRUD and direct hash regeneration example
- :ref:`Agent Tutorial <agent-tutorial>` — Agent direct hash regeneration example
- :ref:`Conference Tutorial <conference-tutorial>` — Conference direct hash regeneration example
- :ref:`AI Tutorial <ai-tutorial>` — AI direct hash regeneration example
- :ref:`Team Tutorial <team-tutorial>` — Team direct hash regeneration example
```

**Step 2: Verify the file exists and is well-formed**

```bash
wc -l bin-api-manager/docsdev/source/direct_hash_overview.rst
```

Expected: ~180 lines.

---

### Task 4: Add `direct_hash` to `index.rst`

**Files:**
- Modify: `bin-api-manager/docsdev/source/index.rst:23-31`

**Step 1: Add `direct_hash` after `webhook` in the Core Concepts toctree**

Change:

```rst
.. toctree::
   :maxdepth: 5
   :caption: Core Concepts

   flow
   variable
   webhook
   common
```

To:

```rst
.. toctree::
   :maxdepth: 5
   :caption: Core Concepts

   flow
   variable
   webhook
   direct_hash
   common
```

**Step 2: Verify the change**

```bash
grep -A 8 'Core Concepts' bin-api-manager/docsdev/source/index.rst
```

Expected: `direct_hash` appears between `webhook` and `common`.

---

### Task 5: Build HTML and verify

**Step 1: Rebuild HTML (clean build)**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: Build succeeds with no errors. Warnings about missing cross-reference targets are acceptable if the target pages exist (they do).

**Step 2: Verify the page renders**

```bash
ls bin-api-manager/docsdev/build/html/direct_hash.html
```

Expected: File exists.

**Step 3: Check for broken cross-references**

```bash
grep -c "direct-hash-overview" bin-api-manager/docsdev/build/html/direct_hash.html
```

Expected: At least 1 match (the anchor exists).

---

### Task 6: Commit and push

**Step 1: Stage all changes**

```bash
git add bin-api-manager/docsdev/source/direct_hash.rst
git add bin-api-manager/docsdev/source/direct_hash_overview.rst
git add bin-api-manager/docsdev/source/index.rst
git add -f bin-api-manager/docsdev/build/
```

Note: `git add -f` is required for `build/` because root `.gitignore` excludes `build/`.

**Step 2: Commit**

```bash
git commit -m "NOJIRA-Add-direct-hash-rst-documentation

Add standalone RST documentation page for the direct hash feature.

- bin-api-manager: Add direct_hash.rst wrapper and direct_hash_overview.rst
- bin-api-manager: Add direct_hash to Core Concepts in index.rst
- bin-api-manager: Rebuild HTML documentation"
```

**Step 3: Push**

```bash
git push -u origin NOJIRA-Add-direct-hash-rst-documentation
```

---

### Task 7: Create PR

```bash
gh pr create --title "NOJIRA-Add-direct-hash-rst-documentation" --body "$(cat <<'EOF'
Add standalone RST documentation page explaining the direct hash feature
as a cross-cutting platform concept.

- bin-api-manager: Add direct_hash.rst wrapper file
- bin-api-manager: Add direct_hash_overview.rst with full feature documentation
- bin-api-manager: Add direct_hash to Core Concepts toctree in index.rst
- bin-api-manager: Rebuild HTML documentation
EOF
)"
```
