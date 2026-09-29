"""Per-tool Gemini validation of advertised tools (design 15.5, part E).

One tool whose schema the google-genai client rejects makes every Gemini turn
of the session fail before any network request, built-ins included. This
module validates the tool set with the installed Gemini adapter and drops
only the tools the validator rejects. A failure of the validator itself never
removes tools: every unexpected path returns the input unchanged.

Only loguru is imported at module level so the module imports without
pipecat, google-genai or pydantic.
"""

from loguru import logger

_MAX_DROP_MESSAGE_LEN = 300


def drop_gemini_invalid_tools(schemas, pipeline_id="", adapter_factory=None, config_factory=None):
    """Return schemas minus the tools the Gemini schema validator rejects.

    adapter_factory/config_factory default to pipecat's GeminiLLMAdapter and
    google-genai's GenerateContentConfig. They are injectable for tests; a
    real import happens only for a factory argument that is None.
    """
    if not schemas:
        return schemas

    try:
        import pydantic
        from pipecat.adapters.schemas.tools_schema import ToolsSchema

        if adapter_factory is None:
            from pipecat.adapters.services.gemini_adapter import GeminiLLMAdapter

            adapter_factory = GeminiLLMAdapter
        if config_factory is None:
            from google.genai.types import GenerateContentConfig

            config_factory = GenerateContentConfig
    except Exception as e:
        logger.warning(
            f"Gemini tool validation skipped, validator import failed: {type(e).__name__}. "
            f"All {len(schemas)} tools kept. pipeline id={pipeline_id}"
        )
        return schemas

    def validate(tools):
        adapter = adapter_factory()
        config_factory(tools=adapter.to_provider_tools_format(ToolsSchema(standard_tools=tools)))

    try:
        try:
            validate(schemas)
            return schemas
        except pydantic.ValidationError:
            pass

        kept = []
        for fs in schemas:
            try:
                validate([fs])
            except pydantic.ValidationError as e:
                logger.warning(f"{_drop_message(fs.name, e)}. pipeline id={pipeline_id}")
                continue
            kept.append(fs)

        try:
            validate(kept)
        except pydantic.ValidationError:
            logger.warning(
                f"Gemini tool validation failed for the filtered tool set; all {len(schemas)} tools kept. "
                f"pipeline id={pipeline_id}"
            )
            return schemas
    except Exception as e:
        logger.warning(
            f"Gemini tool validation failed unexpectedly: {type(e).__name__}. "
            f"All {len(schemas)} tools kept. pipeline id={pipeline_id}"
        )
        return schemas

    if len(kept) != len(schemas):
        logger.info(f"Gemini tool validation dropped {len(schemas) - len(kept)} of {len(schemas)} tools. pipeline id={pipeline_id}")
    return kept


def _drop_message(name, err):
    errors = err.errors()
    first = errors[0] if errors else {}
    loc = ".".join(str(p) for p in first.get("loc", ()))
    msg = first.get("msg", "")
    text = f"Dropped tool '{name}' rejected by the Gemini schema validator: {len(errors)} error(s), first: {loc}: {msg}"
    return text[:_MAX_DROP_MESSAGE_LEN]
