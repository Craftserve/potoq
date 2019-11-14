package cloudychat

import "fmt"
import "github.com/Craftserve/potoq"
import "github.com/Craftserve/potoq/packets"
import "strings"

func replyCommand(handler *potoq.Handler, inPacket *packets.ChatMessagePacketSB, args []string) error {
	if !handler.HasPermission("cloudychat.reply.use") {
		handler.SendChatMessage(packets.COLOR_RED + "Nie masz permisji do tej komendy!")
		return potoq.ErrDropPacket
	}

	if len(args) < 2 {
		handler.SendChatMessage(packets.COLOR_RED + "Poprawne uzycie: /r <wiadomosc>")
		return potoq.ErrDropPacket
	}

	sender := getChatUser(handler)
	recipient := potoq.Players.GetByNickname(sender.GetMsgRecipient())
	if recipient == nil || (recipient.HasPermission("cloudychat.msg.disallow") && !handler.HasPermission("cloudychat.msg.bypass")) {
		handler.SendChatMessage(packets.COLOR_RED + "Gracz nie jest online!")
		return potoq.ErrDropPacket
	}

	msg := strings.Join(args[1:], " ")
	spy_formatted := fmt.Sprintf(SOCIALSPY_FORMAT, handler.Nickname, recipient.Nickname, msg)

	if handler.HasPermission("cloudychat.color") { // admin
		recipient.SendChatMessage(fmt.Sprintf(ADMIN_MSG_FORMAT, handler.Nickname, "ja", msg))
	} else {
		recipient.SendChatMessage(fmt.Sprintf(MSG_FORMAT, handler.Nickname, "ja", msg))
	}

	handler.SendChatMessage(fmt.Sprintf(MSG_FORMAT, "ja", recipient.Nickname, msg))

	sender.SetMsgRecipient(recipient.Nickname)
	getChatUser(recipient).SetMsgRecipient(handler.Nickname)

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
