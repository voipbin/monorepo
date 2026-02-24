# Add Google Cloud TTS and STT to Pipecat Pipeline — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Google Cloud TTS and STT as provider options in the Pipecat Python pipeline.

**Architecture:** Wire `GoogleTTSService` and `GoogleSTTService` into the existing factory functions in `run.py`. Authentication uses GKE Application Default Credentials (no explicit credentials). Language strings are mapped to pipecat's `Language` enum for Google STT.

**Tech Stack:** Python, pipecat-ai (already installed with `[google]` extra), Google Cloud TTS/STT APIs.

---

## Context

- **Only file to modify:** `bin-pipecat-manager/scripts/pipecat/run.py`
- **Test file:** `bin-pipecat-manager/scripts/pipecat/test_run.py`
- `GoogleTTSService` is already imported at line 13 of `run.py`
- `GoogleSTTService` needs to be imported from `pipecat.services.google.stt`
- `Language` enum lives at `pipecat.transcriptions.language.Language` (values like `Language.EN_US`)
- String-to-enum conversion: `"en-US"` → `"EN_US"` → `Language["EN_US"]`
- `requirements.txt` already has `pipecat-ai[google]` — no changes needed
- No Go-side changes needed — `stt_type`/`tts_type` already passed as strings
- Existing tests use `unittest.mock.patch` for service constructors

---

### Task 1: Add Google STT import and helper

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:15-19` (STT imports section)

**Step 1: Add imports**

Add after line 18 (`from pipecat.services.whisper.stt import Model, WhisperSTTService`):

```python
from pipecat.services.google.stt import GoogleSTTService
from pipecat.transcriptions.language import Language
```

**Step 2: Add language string-to-enum helper**

Add before `create_tts_service()` (before line 229):

```python
def _parse_language(language_str: str) -> Language:
    """Convert language string (e.g., 'en-US') to pipecat Language enum."""
    try:
        return Language[language_str.replace("-", "_").upper()]
    except (KeyError, AttributeError):
        return Language.EN_US
```

**Step 3: Verify no syntax errors**

Run: `cd bin-pipecat-manager/scripts/pipecat && python3 -c "import ast; ast.parse(open('run.py').read())"`
Expected: No output (success)

---

### Task 2: Add Google case to `create_tts_service()`

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:229-247` (`create_tts_service` function)

**Step 1: Write the failing test**

Add to `test_run.py`:

```python
class TestCreateTTSService:
    """Tests for create_tts_service function."""

    @patch("run.GoogleTTSService")
    def test_google_tts_service_creation(self, mock_service):
        """Test Google TTS service is created with voice_id and no explicit credentials."""
        from run import create_tts_service

        create_tts_service("google", voice_id="en-US-Chirp3-HD-Charon")

        mock_service.assert_called_once_with(voice_id="en-US-Chirp3-HD-Charon")

    @patch("run.GoogleTTSService")
    def test_google_tts_default_voice(self, mock_service):
        """Test Google TTS uses default voice when none specified."""
        from run import create_tts_service

        create_tts_service("google")

        mock_service.assert_called_once_with(voice_id="default_voice_id")

    @patch("run.CartesiaTTSService")
    def test_cartesia_still_works(self, mock_service):
        """Test existing Cartesia provider is not broken."""
        from run import create_tts_service

        with patch.dict(os.environ, {"CARTESIA_API_KEY": "test-key"}):
            create_tts_service("cartesia", voice_id="test-voice", language="en")

        mock_service.assert_called_once_with(
            api_key="test-key",
            voice_id="test-voice",
            language="en",
        )

    def test_unsupported_tts_raises_error(self):
        """Test unsupported TTS provider raises ValueError."""
        from run import create_tts_service

        with pytest.raises(ValueError, match="Unsupported TTS service"):
            create_tts_service("nonexistent")
```

**Step 2: Run test to verify it fails**

Run: `cd bin-pipecat-manager/scripts/pipecat && python3 -m pytest test_run.py::TestCreateTTSService::test_google_tts_service_creation -v`
Expected: FAIL — `ValueError: Unsupported TTS service: google`

**Step 3: Add Google case to `create_tts_service()`**

In `create_tts_service()`, add before the `else` clause:

```python
    elif name == "google":
        return GoogleTTSService(
            voice_id=voice_id,
        )
```

**Step 4: Run tests to verify they pass**

Run: `cd bin-pipecat-manager/scripts/pipecat && python3 -m pytest test_run.py::TestCreateTTSService -v`
Expected: All 4 tests PASS

---

### Task 3: Add Google case to `create_stt_service()`

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/run.py:250-264` (`create_stt_service` function)

**Step 1: Write the failing test**

Add to `test_run.py`:

```python
class TestCreateSTTService:
    """Tests for create_stt_service function."""

    @patch("run.GoogleSTTService")
    def test_google_stt_service_creation(self, mock_service):
        """Test Google STT service is created with language mapped to Language enum."""
        from run import create_stt_service

        create_stt_service("google", language="en-US")

        mock_service.assert_called_once()
        call_kwargs = mock_service.call_args[1]
        assert "params" in call_kwargs
        params = call_kwargs["params"]
        assert params.languages == [Language.EN_US]

    @patch("run.GoogleSTTService")
    def test_google_stt_default_language(self, mock_service):
        """Test Google STT defaults to EN_US when no language provided."""
        from run import create_stt_service

        create_stt_service("google")

        mock_service.assert_called_once()
        call_kwargs = mock_service.call_args[1]
        params = call_kwargs["params"]
        assert params.languages == [Language.EN_US]

    @patch("run.GoogleSTTService")
    def test_google_stt_unknown_language_fallback(self, mock_service):
        """Test Google STT falls back to EN_US for unknown language strings."""
        from run import create_stt_service

        create_stt_service("google", language="xx-YY")

        mock_service.assert_called_once()
        call_kwargs = mock_service.call_args[1]
        params = call_kwargs["params"]
        assert params.languages == [Language.EN_US]

    @patch("run.DeepgramSTTService")
    def test_deepgram_still_works(self, mock_service):
        """Test existing Deepgram provider is not broken."""
        from run import create_stt_service

        with patch.dict(os.environ, {"DEEPGRAM_API_KEY": "test-key"}):
            create_stt_service("deepgram", language="en")

        mock_service.assert_called_once()

    def test_unsupported_stt_raises_error(self):
        """Test unsupported STT provider raises ValueError."""
        from run import create_stt_service

        with pytest.raises(ValueError, match="Unsupported STT service"):
            create_stt_service("nonexistent")
```

**Step 2: Run test to verify it fails**

Run: `cd bin-pipecat-manager/scripts/pipecat && python3 -m pytest test_run.py::TestCreateSTTService::test_google_stt_service_creation -v`
Expected: FAIL — `ValueError: Unsupported STT service: google`

**Step 3: Add Google case to `create_stt_service()`**

In `create_stt_service()`, add before the `else` clause:

```python
    elif name == "google":
        lang = _parse_language(language) if language else Language.EN_US
        return GoogleSTTService(
            params=GoogleSTTService.InputParams(
                languages=[lang],
                model="latest_long",
                enable_automatic_punctuation=True,
                enable_interim_results=True,
            ),
        )
```

**Step 4: Run all tests to verify they pass**

Run: `cd bin-pipecat-manager/scripts/pipecat && python3 -m pytest test_run.py -v`
Expected: ALL tests PASS (existing + new)

---

### Task 4: Run full test suite and commit

**Step 1: Run all tests**

Run: `cd bin-pipecat-manager/scripts/pipecat && python3 -m pytest test_run.py -v`
Expected: All tests pass

**Step 2: Verify AST parse (no syntax errors)**

Run: `cd bin-pipecat-manager/scripts/pipecat && python3 -c "import ast; ast.parse(open('run.py').read()); print('OK')"`
Expected: `OK`

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-google-tts-stt-pipecat
git add bin-pipecat-manager/scripts/pipecat/run.py bin-pipecat-manager/scripts/pipecat/test_run.py docs/plans/2026-02-24-add-google-tts-stt-pipecat-design.md
git commit -m "NOJIRA-add-google-tts-stt-pipecat

Add Google Cloud TTS and STT as provider options in the Pipecat voice pipeline.

- bin-pipecat-manager: Add Google case to create_tts_service() using GoogleTTSService (streaming, Chirp 3 HD voices)
- bin-pipecat-manager: Add Google case to create_stt_service() using GoogleSTTService with Language enum mapping
- bin-pipecat-manager: Add _parse_language() helper for string-to-Language enum conversion
- bin-pipecat-manager: Add GoogleSTTService import and Language enum import
- bin-pipecat-manager: Add tests for Google TTS and STT service creation
- docs: Add design document for Google TTS/STT integration"
```

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-add-google-tts-stt-pipecat
```
