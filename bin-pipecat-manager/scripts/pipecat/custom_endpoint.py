"""pipecat LLM service for a customer specified OpenAI-compatible endpoint (VOIP-1599).

Imports pipecat; the network code lives in custom_endpoint_net.py (no pipecat import).
run.py imports this module lazily inside the customendpoint branch of
create_llm_service. Design section 3.8 and 3.9.
"""

import asyncio

from pipecat.services.openai.llm import OpenAILLMService
from pipecat.utils.errors import classify_http_exception

import custom_endpoint_net as net


class CustomEndpointLLMService(OpenAILLMService):
    """OpenAILLMService whose client can only reach validated public addresses.

    One slot (net.SLOTS) is held per instance from construction until release().
    release() is idempotent and gives the slot back before it closes the client.
    """

    # Test hook (a ValidatingBackend with a fake inner backend); production code never sets it.
    _http_backend = None

    def __init__(self, *, customer_id: str, base_url: str, api_key: str, model: str, **kwargs):
        # create_client() is called inside super().__init__(): everything it needs is set first.
        self._ce_customer_id = customer_id
        self._ce_pipeline_id = None
        self._ce_released = False
        self._ce_slot_held = False
        super().__init__(
            api_key=api_key if isinstance(api_key, str) else "",
            base_url=base_url,
            settings=self.Settings(model=model),
            **kwargs,
        )

    def create_client(self, api_key=None, base_url=None, organization=None, project=None, default_headers=None, **kwargs):
        # organization, project and default_headers of the caller are ignored on purpose (the
        # organization and project headers are Omit()ted, see net.build_openai_client).
        return net.build_openai_client(
            api_key=api_key, base_url=base_url, customer_id=self._ce_customer_id, backend=type(self)._http_backend
        )

    # --- slot and client lifecycle (R18) ---

    def attach_slot(self, pipeline_id: str) -> None:
        """Called by the factory once the constructor succeeded and the client was verified."""
        self._ce_slot_held = True
        self._ce_pipeline_id = pipeline_id
        net.register_service(pipeline_id, self)

    def release_slot(self) -> None:
        """Give the slot back and forget the registration. Synchronous and idempotent."""
        if not self._ce_slot_held:
            return
        self._ce_slot_held = False
        net.SLOTS.release(self._ce_customer_id)
        net.unregister_service(self._ce_pipeline_id, self)

    async def release(self) -> None:
        """Idempotent: slot first (synchronously), then close the client (and so the pool)."""
        if self._ce_released:
            return
        self._ce_released = True
        self.release_slot()
        client = getattr(self, "_client", None)
        if client is None:
            return
        try:
            # shielded: a cancellation of the caller must not abandon a half closed pool
            await asyncio.shield(client.close())
        except Exception:
            pass

    async def cleanup(self):
        # pipecat calls cleanup() more than once, and the cleanup chain around it is a
        # sequence of awaits in which an earlier failure skips the later steps.
        try:
            await super().cleanup()
        finally:
            await self.release()

    async def push_error(
        self, error_msg, exception=None, fatal=False, category=None, force_treat_as_permanent=False
    ):
        """Keep the fixed phrases the classifier needs; drop the customer server's response body.

        pipecat logs the error text at error level unconditionally and an HTTP error text
        includes the body. The exception is not forwarded (its text carries the body too);
        the category is worked out from it first so the pipeline policy is unchanged.
        """
        if category is None and exception is not None:
            try:
                category = self._classify_error(exception) or classify_http_exception(exception)
            except Exception:
                category = None
        await super().push_error(
            error_msg=net.sanitize_error_text(error_msg),
            exception=None,
            fatal=fatal,
            category=category,
            force_treat_as_permanent=force_treat_as_permanent,
        )


def build_custom_endpoint_llm(model: str, *, base_url, api_key, customer_id, pipeline_id) -> CustomEndpointLLMService:
    """Build the service or raise ValueError(FIXED_ERROR) (callers add `from None`).

    The slot is taken before the constructor and given back if it fails. The legacy
    key of the session is never an argument: only the stored endpoint key (or "").
    """
    if not isinstance(model, str) or not model:
        raise ValueError(net.FIXED_ERROR)
    if not isinstance(pipeline_id, str) or not pipeline_id:
        raise ValueError(net.FIXED_ERROR)
    customer = net.parse_customer_id(customer_id)  # required: the per customer limit
    url = net.check_runner_base_url(base_url)
    if not net.SLOTS.acquire(customer):
        raise ValueError(net.FIXED_ERROR)
    try:
        service = CustomEndpointLLMService(
            customer_id=customer, base_url=url, api_key=api_key if isinstance(api_key, str) else "", model=model
        )
    except BaseException:
        net.SLOTS.release(customer)
        raise
    service.attach_slot(pipeline_id)
    try:
        net.verify_openai_client(service._client, url)
    except BaseException:
        service.release_slot()  # never used: the pool holds no connection to close
        raise
    return service
