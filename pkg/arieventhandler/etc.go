package arieventhandler

import (
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

func (h *eventHandler) getChannelContextType(cn *channel.Channel) channel.ContextType {
	res := channel.ContextType(cn.StasisData[channel.StasisDataTypeContextType])
	return res
}
