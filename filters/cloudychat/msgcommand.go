package cloudychat

import "fmt"
import "github.com/Craftserve/potoq"
import "github.com/Craftserve/potoq/packets"
import "strings"

var MSG_FORMAT = packets.COLOR_GOLD + "[%s -> %s] " + packets.COLOR_GRAY + "%s"
var ADMIN_MSG_FORMAT = packets.COLOR_RED + "[%s -> %s] " + packets.COLOR_WHITE + "%s"
var SOCIALSPY_FORMAT = packets.COLOR_GRAY + "[%s -> %s] " + packets.COLOR_GRAY + "%s"

func msgCommand(handler *potoq.Handler, inPacket *packets.ChatMessagePacketSB, args []string) error {
	if !handler.HasPermission("cloudychat.msg.use") {
		handler.SendChatMessage(packets.COLOR_RED + "Nie masz uprawnien do tej komendy!")
		return potoq.ErrDropPacket
	}

	if len(args) < 3 {
		handler.SendChatMessage(packets.COLOR_RED + "Poprawne uzycie: /msg <nick> <wiadomosc>")
		return potoq.ErrDropPacket
	}

	recipient := potoq.Players.GetByNickname(args[1])
	if recipient == nil || (recipient.HasPermission("cloudychat.msg.disallow") && !handler.HasPermission("cloudychat.msg.bypass")) {
		handler.SendChatMessage(packets.COLOR_RED + "Gracz nie jest online!")
		return potoq.ErrDropPacket
	}

	recipient_user := getChatUser(recipient)
	if recipient_user.GetIgnored(handler.Nickname) && !handler.HasPermission("cloudychat.ignore.bypass") {
		handler.SendChatMessage(packets.COLOR_RED + "Ten gracz dodal cie do listy ignorowanych! Nie mozesz wysylac mu wiadomosci!")
		return potoq.ErrDropPacket
	}

	msg := strings.Join(args[2:], " ")
	spy_formatted := fmt.Sprintf(SOCIALSPY_FORMAT, handler.Nickname, recipient.Nickname, msg)

	if handler.HasPermission("cloudychat.color") { // admin
		recipient.SendChatMessage(fmt.Sprintf(ADMIN_MSG_FORMAT, handler.Nickname, "ja", msg))
	} else {
		recipient.SendChatMessage(fmt.Sprintf(MSG_FORMAT, handler.Nickname, "ja", msg))
	}

	handler.SendChatMessage(fmt.Sprintf(MSG_FORMAT, "ja", recipient.Nickname, msg))

	sender := getChatUser(handler)
	sender.AddPrivMsgHistory(recipient.Nickname, spy_formatted)
	recipient_user.SetMsgRecipient(handler.Nickname)

	socialspiesMutex.Lock()
	for spy, _ := range socialspies {
		h := potoq.Players.GetByNickname(spy)
		if h != nil && h.HasPermission("cloudychat.socialspy") {
			h.SendChatMessage(spy_formatted)
		}
	}
	socialspiesMutex.Unlock()

	return potoq.ErrDropPacket
}
