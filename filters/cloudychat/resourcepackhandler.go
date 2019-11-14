package cloudychat

import (
	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

const textureAppliedKey = "filters/cloudyChat.textureApplied"

func resFilter(handler *potoq.Handler, rawPacket packets.Packet) error {
	_ = rawPacket.(*packets.JoinGamePacketCB)
	if globalChatConfig.Resource_pack_url == "" {
		return nil
	}

	_, ok := handler.FilterData.LoadOrStore(textureAppliedKey, textureAppliedKey)
	if ok {
		return nil
	}
	panic("ResourcePackSendPacketCB not implemented") // TODO: kod nizej nie dziala z 1.13
	return handler.InjectPackets(packets.ClientBound, nil, &packets.PluginMessagePacketCB{Channel: "MC|RPack", Payload: []byte(globalChatConfig.Resource_pack_url)})
}
