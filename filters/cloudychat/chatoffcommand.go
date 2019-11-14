package cloudychat

import "github.com/Craftserve/potoq"
import "github.com/Craftserve/potoq/packets"
import "strings"

func chatoffCommand(handler *potoq.Handler, inPacket *packets.ChatMessagePacketSB, args []string) error {
	if !handler.HasPermission("cloudychat.chatoff.use") {
		handler.SendChatMessage(packets.COLOR_RED + "Nie masz uprawnien do tej komendy!")
		return potoq.ErrDropPacket
	}

	if len(args) != 2 {
		handler.SendChatMessage(packets.COLOR_RED + "Poprawne uzycie: /chat off|on")
		return potoq.ErrDropPacket
	}

	var state bool

	switch strings.ToLower(args[1]) {
	case "off":
		handler.SendChatMessage(packets.COLOR_GOLD + "Chat wylaczony!")
		state = true
	case "on":
		handler.SendChatMessage(packets.COLOR_GOLD + "Chat wlaczony!")
		state = false
	default:
		handler.SendChatMessage(packets.COLOR_RED + "Poprawne uzycie: /chat off|on")
		return potoq.ErrDropPacket
	}

	globalChatLock.Lock()
	globalChatConfig.ChatOff = state
	globalChatLock.Unlock()

	return potoq.ErrDropPacket
}
