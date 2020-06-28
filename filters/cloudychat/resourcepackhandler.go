package cloudychat

import (
	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
	"time"
)

const textureAppliedKey = "filters/cloudyChat.textureApplied"

func resFilter(handler *potoq.Handler, rawPacket packets.Packet) error {
	_ = rawPacket.(*packets.JoinGamePacketCB)
	resourcePack := globalChatConfig.ResourcePack
	if resourcePack.Url == "" {
		return nil
	}

	_, ok := handler.FilterData.LoadOrStore(textureAppliedKey, textureAppliedKey)
	if ok {
		return nil
	}
	resourcePacket := &packets.ResourcePackSendCB{
		Url:  resourcePack.Url,
		Hash: resourcePack.Hash,
	}
	go func() {
		time.Sleep(1 * time.Second)
		_ = handler.InjectPackets(packets.ClientBound, nil, resourcePacket)
	}()
	return nil
}
