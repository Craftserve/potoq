package cloudychat

import "github.com/Craftserve/potoq"
import "github.com/Craftserve/potoq/packets"

func ignoreCommand(handler *potoq.Handler, inPacket *packets.ChatMessagePacketSB, args []string) error {
	if !handler.HasPermission("cloudychat.ignore.use") {
		handler.SendChatMessage(packets.COLOR_RED + "Nie masz uprawnien do tej komendy!")
		return potoq.ErrDropPacket
	}

	if len(args) < 2 {
		handler.SendChatMessage(packets.COLOR_RED + "Poprawne uzycie: /ignore <nick>")
		return potoq.ErrDropPacket
	}

	user := getChatUser(handler)
	val := !user.GetIgnored(args[1])
	user.SetIgnored(args[1], val)

	switch val {
	case true:
		handler.SendChatMessage(packets.COLOR_GREEN + "Gracz jest teraz ignorowany")
	case false:
		handler.SendChatMessage(packets.COLOR_GREEN + "Gracz nie jest juz ignorowany")
	}

	return potoq.ErrDropPacket
}
