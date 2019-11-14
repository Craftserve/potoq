package cloudychat

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

var socialspies = make(map[string]bool)
var socialspiesMutex = new(sync.Mutex)

func socialspyCommand(handler *potoq.Handler, inPacket *packets.ChatMessagePacketSB, args []string) error {
	if !handler.HasPermission("cloudychat.socialspy") {
		handler.SendChatMessage(packets.COLOR_RED + "Nie masz uprawnien do tej komendy!")
		return potoq.ErrDropPacket
	}

	if len(args) > 1 {
		h := potoq.Players.GetByNickname(args[1])
		if h == nil {
			handler.SendChatMessage(packets.COLOR_RED + "Nie odnaleziono gracza online!")
			return potoq.ErrDropPacket
		}
		user := getChatUser(h)
		user.Lock()
		defer user.Unlock()
		handler.SendChatMessage(fmt.Sprint(
			"Wyslane przez ", h.Nickname, ": \n", packets.COLOR_DARK_BLUE,
			strings.Join(user.SentPrivMsgs, "\n"+packets.COLOR_DARK_BLUE),
		))
		return potoq.ErrDropPacket
	}

	socialspiesMutex.Lock()
	defer socialspiesMutex.Unlock()

	switch socialspies[handler.Nickname] {
	case true:
		delete(socialspies, handler.Nickname)
		handler.SendChatMessage(packets.COLOR_GOLD + "Socialspy wylaczony!")
	case false:
		socialspies[handler.Nickname] = true
		handler.SendChatMessage(packets.COLOR_GOLD + "Socialspy wlaczony!")
	}

	return potoq.ErrDropPacket
}
