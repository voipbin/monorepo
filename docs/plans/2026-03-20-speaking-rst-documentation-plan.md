# Speaking RST Documentation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add dedicated Speaking (TTS) documentation to the RST docs — overview, struct reference, and tutorial — following the established 3-file pattern.

**Architecture:** Create 4 new RST files (`speaking.rst`, `speaking_overview.rst`, `speaking_struct_speaking.rst`, `speaking_tutorial.rst`) in `bin-api-manager/docsdev/source/`, modify `index.rst` to add Speaking to the toctree, and add cross-references from `transcribe_overview.rst`.

**Tech Stack:** reStructuredText (Sphinx), built with `python3 -m sphinx -M html source build`

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation`

**Base path for all RST files:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev/source/`

---

### Task 1: Create `speaking.rst` (wrapper file)

**Files:**
- Create: `bin-api-manager/docsdev/source/speaking.rst`

**Reference:** `bin-api-manager/docsdev/source/transcribe.rst` (same pattern: anchor, title, description, API Reference link, includes)

**Step 1: Create the wrapper file**

```rst
.. _speaking-main:

**********
Speaking
**********
Real-time text-to-speech (TTS) injection into live calls and conferences, with support for multiple providers, voice selection, and directional audio control.

**API Reference:** `Speaking endpoints <https://api.voipbin.net/redoc/#tag/Speaking>`_

.. include:: speaking_overview.rst
.. include:: speaking_struct_speaking.rst
.. include:: speaking_tutorial.rst
```

**Key conventions:**
- `.. _speaking-main:` anchor matches pattern `<resource>-main`
- Asterisks above and below title (count must match title length + padding)
- Include order: overview → struct → tutorial (matches `recording.rst`)
- One-line description (no blank line before API Reference)

**Step 2: Verify file exists**

```bash
cat ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev/source/speaking.rst
```

Expected: File contents displayed without errors.

---

### Task 2: Create `speaking_overview.rst`

**Files:**
- Create: `bin-api-manager/docsdev/source/speaking_overview.rst`

**Reference files for formatting:**
- `transcribe_overview.rst` — peer resource, same structure
- `bin-tts-manager/models/speaking/status.go` — Status enum: initiating, active, stopped
- `bin-tts-manager/models/streaming/streaming.go` — Direction enum (in/out/both), ReferenceType enum (call/confbridge), VendorName (elevenlabs/gcp/aws)

**Step 1: Create the overview file**

The file must contain these sections in order:

1. **Anchor and heading** — `.. _speaking-overview:` then `Overview` with `====`
2. **AI Context block** — Complexity: Low, Cost: Chargeable (per TTS request), Async: Yes (poll for `active`)
3. **Intro paragraph** — What Speaking API does, 5 bullet capabilities
4. **How Speaking Works** section (`---`) — Architecture diagram (ASCII), key components
5. **Speaking Lifecycle** section (`---`) — Status state diagram: initiating → active → stopped, table of all states
6. **Providers** section (`---`) — Table: elevenlabs (default), gcp, aws, with descriptions
7. **Direction** section (`---`) — Table: in, out, both, with audio routing explanation
8. **Reference Types** section (`---`) — call and confbridge, how each works
9. **Best Practices** section (`---`) — 5-6 bullet points
10. **Troubleshooting** section (`---`) — HTTP error cause+fix pairs (400, 404, 409)
11. **Related Documentation** section (`---`) — Cross-refs to call, conference, transcribe, quickstart_realtime

**Content specifics:**

AI Context block:
```rst
.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (per TTS synthesis request)
   * **Async:** Yes. ``POST /speakings`` returns immediately with status ``initiating``. Poll ``GET /speakings/{id}`` until status is ``active`` before calling ``POST /speakings/{id}/say``.
```

Intro paragraph capabilities:
- Inject synthesized speech into live calls and conferences
- Choose from multiple TTS providers (ElevenLabs, Google Cloud, AWS)
- Select specific voices or use provider defaults
- Control audio direction (caller only, callee only, or both)
- Queue multiple speech segments with flush control

Architecture diagram:
```
+--------+        +----------------+        +-------------+
|  Call  |<-audio--|     TTS       |<-text--|  Your App   |
+--------+        |    Engine      |        | POST /say   |
                  +----------------+        +-------------+
+------------+           |
| Conference |<--audio---+
+------------+
```

Lifecycle diagram:
```
POST /speakings
       |
       v
+-------------+                    +-------------+
| initiating  |----setup done----->|   active    |
+-------------+                    +------+------+
                                          |
                        POST /speakings/{id}/stop or call hangup
                                          |
                                          v
                                   +-------------+
                                   |   stopped   |
                                   +-------------+
```

Status table:
```
=========== ============
Status      Description
=========== ============
initiating  TTS session is being set up. Provider connection is being established. Do not call ``/say`` in this state.
active      TTS session is ready. You can send text via ``POST /speakings/{id}/say``. Audio is being injected into the call.
stopped     TTS session has ended. Either stopped explicitly via ``POST /speakings/{id}/stop`` or the call was hung up.
=========== ============
```

Provider table:
```
=========== ============
Provider    Description
=========== ============
elevenlabs  ElevenLabs TTS. High-quality neural voices with natural intonation. Default provider if omitted.
gcp         Google Cloud Text-to-Speech. Wide language support with WaveNet and Neural2 voices.
aws         Amazon Polly. Neural and standard voices with SSML support.
=========== ============
```

Direction table:
```
=========== ============
Direction   Description
=========== ============
in          Audio injected toward the caller (remote party hears it, local party does not).
out         Audio injected toward the callee/local side (local party hears it, remote party does not).
both        Audio injected to both sides of the call. Both parties hear the synthesized speech.
=========== ============
```

AI Implementation Hint (at least one, place after lifecycle section):
```rst
.. note:: **AI Implementation Hint**

   Always poll ``GET /speakings/{id}`` until ``status`` is ``active`` before calling ``POST /speakings/{id}/say``. Sending text while status is ``initiating`` will fail. Typical setup time is 2-3 seconds. Only one active speaking session per call is allowed — create a new session only after the previous one is ``stopped``.
```

Related Documentation:
```rst
Related Documentation
---------------------

- :ref:`Call Overview <call-overview>` - Attaching TTS to calls
- :ref:`Conference Overview <conference-overview>` - Attaching TTS to conferences
- :ref:`Transcribe Overview <transcribe-overview>` - Speech-to-text (the listen counterpart)
- :ref:`Quickstart: Real-Time Voice <quickstart-realtime>` - End-to-end speaking + transcription example
```

**Step 2: Verify file**

```bash
wc -l ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev/source/speaking_overview.rst
```

Expected: ~200-300 lines.

---

### Task 3: Create `speaking_struct_speaking.rst`

**Files:**
- Create: `bin-api-manager/docsdev/source/speaking_struct_speaking.rst`

**Reference files:**
- `transcribe_struct.rst` — same structure: anchor, heading, JSON template, field list, example, enum sections
- `bin-tts-manager/models/speaking/webhook.go` — WebhookMessage fields (11 fields, no PodID)

**Step 1: Create the struct file**

The file must contain:

1. **Anchor:** `.. _speaking-struct-speaking:`
2. **Heading:** `Speaking` with `====`, then `Speaking` with `----`
3. **JSON template** — all 12 fields as `"<string>"` placeholders (id, customer_id, reference_type, reference_id, language, provider, voice_id, direction, status, tm_create, tm_update, tm_delete)
4. **Field descriptions** — each with type, provenance for IDs, `:ref:` for enums
5. **AI Implementation Hint** — about tm_delete sentinel value (matching transcribe pattern)
6. **Example** — realistic JSON with actual-looking UUIDs and timestamps
7. **Enum sections** — each with anchor, heading, table:
   - `.. _speaking-struct-speaking-reference_type:` — call, confbridge
   - `.. _speaking-struct-speaking-provider:` — elevenlabs, gcp, aws
   - `.. _speaking-struct-speaking-status:` — initiating, active, stopped
   - `.. _speaking-struct-speaking-direction:` — in, out, both

Field descriptions (exact format):
```rst
* ``id`` (UUID): The speaking session's unique identifier. Returned when creating via ``POST /speakings`` or listing via ``GET /speakings``.
* ``customer_id`` (UUID): The customer who owns this speaking session. Obtained from ``GET /customer``.
* ``reference_type`` (enum string): The type of resource receiving TTS audio. See :ref:`Reference Type <speaking-struct-speaking-reference_type>`.
* ``reference_id`` (UUID): The ID of the resource receiving TTS audio. Depending on ``reference_type``, obtained from ``GET /calls`` or ``GET /conferences``.
* ``language`` (String, BCP47): The language and locale for TTS synthesis (e.g., ``en-US``, ``ko-KR``). Must match the provider's supported languages.
* ``provider`` (enum string, optional): The TTS provider used for synthesis. See :ref:`Provider <speaking-struct-speaking-provider>`. If omitted, defaults to ``elevenlabs``.
* ``voice_id`` (String, optional): A provider-specific voice identifier. If omitted, the provider's default voice for the specified language is used. Obtain available voices from the provider's documentation.
* ``direction`` (enum string): The audio routing direction. See :ref:`Direction <speaking-struct-speaking-direction>`.
* ``status`` (enum string): The speaking session's current status. See :ref:`Status <speaking-struct-speaking-status>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the speaking session was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any speaking property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the speaking session was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.
```

Example JSON:
```json
{
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
    "reference_type": "call",
    "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f",
    "language": "en-US",
    "provider": "elevenlabs",
    "voice_id": "",
    "direction": "both",
    "status": "active",
    "tm_create": "2025-06-15 14:30:00.123456",
    "tm_update": "2025-06-15 14:30:02.456789",
    "tm_delete": "9999-01-01 00:00:00.000000"
}
```

**Step 2: Verify file**

```bash
wc -l ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev/source/speaking_struct_speaking.rst
```

Expected: ~130-180 lines.

---

### Task 4: Create `speaking_tutorial.rst`

**Files:**
- Create: `bin-api-manager/docsdev/source/speaking_tutorial.rst`

**Reference files:**
- `transcribe_tutorial.rst` — same structure
- `/home/pchero/gitvoipbin/tmp/tutorials/tutorials.md` — source tutorial content (adapt curl examples)

**Key design decision:** This tutorial does NOT repeat the basic create → say → stop hello-world path (already in `quickstart_realtime.rst`). It covers advanced usage.

**Step 1: Create the tutorial file**

The file must contain these sections:

1. **Anchor and heading:** `.. _speaking-tutorial:` then `Tutorial` with `====`

2. **Prerequisites** (`+++`) — list required resources with provenance:
   - Auth token (from login or accesskey)
   - Active call or conference (from `POST /calls` or `POST /conferences`)
   - Language code (BCP47)
   - (Optional) Provider-specific voice ID

3. **AI Implementation Hint** for prerequisites:
   ```rst
   .. note:: **AI Implementation Hint**

      The call must be in ``progressing`` status before attaching a speaking session.
      Poll ``GET /calls/{id}`` until ``status`` is ``progressing``. If the call reaches
      ``hangup`` status, the call was not answered and you must retry.
   ```

4. **Create a Speaking Session** (`---`) — curl example with `POST /speakings`, full request + response, poll for `active`

5. **Send Text to Speak** (`---`) — curl example with `POST /speakings/{id}/say`, response, note about queuing multiple texts

6. **Choose a TTS Provider** (`---`) — Three curl examples showing `"provider": "elevenlabs"`, `"provider": "gcp"`, `"provider": "aws"` with notes on each

7. **Select a Voice** (`---`) — curl example with `voice_id` field, notes on provider-specific voice IDs

8. **Control Audio Direction** (`---`) — Three examples:
   - `"direction": "both"` — both parties hear TTS (default, good for announcements)
   - `"direction": "out"` — only local party hears (good for agent coaching)
   - `"direction": "in"` — only remote party hears (good for IVR replacement)

9. **Flush the Speech Queue** (`---`) — curl `POST /speakings/{id}/flush`, when to use (interrupt current speech)

10. **Attach Speaking to a Conference** (`---`) — curl with `"reference_type": "confbridge"`, `"reference_id"` from `GET /conferences`

11. **Stop and Delete a Speaking Session** (`---`) — curl `POST /speakings/{id}/stop` then `DELETE /speakings/{id}`, cleanup order

12. **Lifecycle Management** (`---`) — Polling pattern, status transitions, only one active session per call

13. **Troubleshooting** (`---`) — HTTP error cause+fix pairs:
    - 400 Bad Request: invalid language, empty text in say
    - 404 Not Found: speaking ID doesn't exist or belongs to another customer
    - 409 Conflict: call not in progressing state, or another speaking session already active

**Auth in all curl examples:** Use `?token=<your-token>` (matching existing conventions).

**Curl format** (matching existing convention):
```bash
curl --location --request POST 'https://api.voipbin.net/v1.0/speakings?token=<your-token>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "reference_type": "call",
    "reference_id": "<call-id>",
    "language": "en-US",
    "provider": "elevenlabs",
    "direction": "both"
}'
```

**Step 2: Verify file**

```bash
wc -l ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev/source/speaking_tutorial.rst
```

Expected: ~250-350 lines.

---

### Task 5: Add `speaking` to `index.rst` toctree

**Files:**
- Modify: `bin-api-manager/docsdev/source/index.rst` (line 40, after `transcribe`)

**Step 1: Add speaking to the Voice & Real-Time toctree**

Current lines 32-41:
```rst
.. toctree::
   :maxdepth: 5
   :caption: Voice & Real-Time

   call
   conference
   queue
   recording
   transcribe
   mediastream
```

Change to:
```rst
.. toctree::
   :maxdepth: 5
   :caption: Voice & Real-Time

   call
   conference
   queue
   recording
   transcribe
   speaking
   mediastream
```

**Only change:** Add `speaking` on a new line after `transcribe` (line 41) and before `mediastream`.

**Step 2: Verify the edit**

```bash
grep -n "speaking" ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev/source/index.rst
```

Expected: One line showing `speaking` in the toctree.

---

### Task 6: Add Speaking cross-reference to `transcribe_overview.rst`

**Files:**
- Modify: `bin-api-manager/docsdev/source/transcribe_overview.rst` (line ~643, Related Documentation section)

**Step 1: Add cross-reference**

Current Related Documentation (lines 637-643):
```rst
Related Documentation
---------------------

- :ref:`Call Overview <call-overview>` - Transcribing calls
- :ref:`Conference Overview <conference-overview>` - Transcribing conferences
- :ref:`Recording Overview <recording-overview>` - Recording and transcribing together
- :ref:`Flow Actions <flow-struct-action-transribe_start>` - Transcribe flow actions
```

Change to:
```rst
Related Documentation
---------------------

- :ref:`Call Overview <call-overview>` - Transcribing calls
- :ref:`Conference Overview <conference-overview>` - Transcribing conferences
- :ref:`Recording Overview <recording-overview>` - Recording and transcribing together
- :ref:`Speaking Overview <speaking-overview>` - Text-to-speech (the speak counterpart to transcription)
- :ref:`Flow Actions <flow-struct-action-transribe_start>` - Transcribe flow actions
```

**Only change:** Add one line for Speaking cross-reference, placed before Flow Actions.

**Step 2: Verify the edit**

```bash
grep -n "speaking" ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev/source/transcribe_overview.rst
```

Expected: One line showing the Speaking cross-reference.

---

### Task 7: Build HTML and verify

**Files:**
- Read: `bin-api-manager/docsdev/build/` (generated HTML)

**Step 1: Clean rebuild HTML**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev && \
rm -rf build && \
python3 -m sphinx -M html source build
```

Expected: Build completes without errors or warnings about Speaking files. Some existing warnings about other files are acceptable.

**Step 2: Check for Speaking-specific warnings**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev && \
python3 -m sphinx -M html source build 2>&1 | grep -i "speaking"
```

Expected: No warnings referencing speaking files. If there are warnings about broken `:ref:` targets (e.g., `quickstart-realtime` anchor doesn't exist), fix the anchor name.

**Step 3: Verify Speaking page was generated**

```bash
ls -la ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation/bin-api-manager/docsdev/build/html/speaking.html
```

Expected: File exists.

---

### Task 8: Commit all changes

**Step 1: Check git status**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation && git status
```

Expected: New files in `bin-api-manager/docsdev/source/` (speaking*.rst) and modified files (index.rst, transcribe_overview.rst), plus build output.

**Step 2: Stage and commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation && \
git add bin-api-manager/docsdev/source/speaking.rst \
        bin-api-manager/docsdev/source/speaking_overview.rst \
        bin-api-manager/docsdev/source/speaking_struct_speaking.rst \
        bin-api-manager/docsdev/source/speaking_tutorial.rst \
        bin-api-manager/docsdev/source/index.rst \
        bin-api-manager/docsdev/source/transcribe_overview.rst \
        docs/plans/2026-03-20-speaking-rst-documentation-design.md \
        docs/plans/2026-03-20-speaking-rst-documentation-plan.md && \
git add -f bin-api-manager/docsdev/build/
```

Note: `git add -f` is required for build/ because root `.gitignore` contains `build/`.

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation && \
git commit -m "NOJIRA-Add-speaking-rst-documentation

Add dedicated Speaking (TTS) resource documentation to the RST developer docs.

- bin-api-manager: Create speaking.rst wrapper with overview, struct, and tutorial includes
- bin-api-manager: Create speaking_overview.rst with lifecycle, providers, directions, and best practices
- bin-api-manager: Create speaking_struct_speaking.rst with all WebhookMessage fields and enum references
- bin-api-manager: Create speaking_tutorial.rst with provider selection, voice config, and advanced usage
- bin-api-manager: Add speaking to Voice & Real-Time toctree in index.rst
- bin-api-manager: Add Speaking cross-reference in transcribe_overview.rst Related Documentation
- bin-api-manager: Rebuild HTML documentation
- docs: Add design document and implementation plan"
```

**Step 3: Push and create PR**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-speaking-rst-documentation && \
git push -u origin NOJIRA-Add-speaking-rst-documentation
```

Then create PR (user will confirm).
