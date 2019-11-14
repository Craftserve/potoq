package cloudychat

import (
	"golang.org/x/time/rate"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

func reloadCommand(handler *potoq.Handler, inPacket *packets.ChatMessagePacketSB, args []string) error {
	if !handler.HasPermission("cloudychat.reload") {
		handler.SendChatMessage(packets.COLOR_RED + "Nie masz uprawnien do tej komendy!")
		return potoq.ErrDropPacket
	}

	config, err := LoadConfig("chat.yml")
	if err != nil {
		handler.SendChatMessage(packets.COLOR_RED + "Blad podczas reloadowania configu!")
		return potoq.ErrDropPacket
	}

	globalChatLock.Lock()
	defer globalChatLock.Unlock()

	globalChatConfig = config
	globalChatLimiter = rate.NewLimiter(rate.Limit(globalChatConfig.RateLimitHz), globalChatConfig.RateLimitBurst)

	handler.SendChatMessage(packets.COLOR_GOLD + "Config reloaded!")
	return potoq.ErrDropPacket
}
