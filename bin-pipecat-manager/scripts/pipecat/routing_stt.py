from loguru import logger

from routing_base import RoutingServiceBase


class RoutingSTTService(RoutingServiceBase):
    """Routes STT processing to the appropriate provider based on active member."""

    def set_active_member(self, member_id: str):
        if member_id not in self._services:
            # Member has no STT configured — keep using the previous member's STT service.
            # In a voice call, this is safer than having no STT (deaf).
            logger.warning(f"Member {member_id} has no STT service, keeping previous member's STT")
            return
        self._active_id = member_id
        logger.info(f"STT routing switched to member: {member_id}")
