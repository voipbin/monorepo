from loguru import logger

from routing_base import RoutingServiceBase


class RoutingTTSService(RoutingServiceBase):
    """Routes TTS processing to the appropriate provider based on active member."""

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            # Member has no TTS configured — keep using the previous member's TTS service.
            # In a voice call, this is safer than having no TTS (silence).
            logger.warning(f"Member {member_id} has no TTS service, keeping previous member's TTS")
            return
        self._active_id = member_id
        logger.info(f"TTS routing switched to member: {member_id}")
