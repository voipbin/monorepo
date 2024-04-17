package arieventhandler

import (
	"monorepo/bin-call-manager/models/channel"
)

func (h *eventHandler) getChannelContextType(cn *channel.Channel) channel.ContextType {
	res := channel.ContextType(cn.StasisData[channel.StasisDataTypeContextType])
	return res
}
