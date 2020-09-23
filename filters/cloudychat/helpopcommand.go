package cloudychat

import (
	"fmt"
	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
	"strings"
	"sync"
	"time"
)

var helpopFormat = packets.COLOR_AQUA + "[HelpOp] " + packets.COLOR_GRAY + "[%s] %s:" + packets.COLOR_WHITE + " %s"
var helpopHistoryFormat = packets.COLOR_RED + "%s " + packets.COLOR_GRAY + "[%s] %s:" + packets.COLOR_WHITE + " %s"
var helpopTimeFormat = "15:04:05"

var helpopHistory = make([]string, 20)
var helpopHistoryMutex = new(sync.Mutex)

func helpopCommand(handler *potoq.Handler, inPacket *packets.ChatMessagePacketSB, args []string) error {
	if !handler.HasPermission("cloudychat.helpop.use") {
		handler.SendChatMessage(packets.COLOR_RED + "Nie masz uprawnien do tej komendy!")
		return potoq.ErrDropPacket
	}

	if len(args) < 2 {
		if handler.HasPermission("cloudychat.helpop.history") {
			helpopHistoryMutex.Lock()
			var history string
			for _, s := range helpopHistory {
				if s != "" {
					history = history + "\n" + s
				}
			}
			helpopHistoryMutex.Unlock()
			handler.SendChatMessage(history)
		} else {
			handler.SendChatMessage(packets.COLOR_RED + "Poprawne uzycie: /helpop <wiadomosc>")
		}
		return potoq.ErrDropPacket
	}

	if !handler.HasPermission("cloudychat.slowmode.bypass") {
		user := getChatUser(handler)
		if slowmode, ok := checkSlowmode(user.LastHelpopTime); !ok {
			handler.SendChatMessage(packets.COLOR_RED + "Mozesz napisac na helpop za " + slowmode.String())
			return potoq.ErrDropPacket
		}
		user.LastHelpopTime = time.Now()
	}

	msg := fmt.Sprintf(helpopFormat, handler.UpstreamName, handler.Nickname, strings.Join(args[1:], " "))
	if !handler.HasPermission("cloudychat.helpop.view") {
		handler.SendChatMessage(msg)
	}
	potoq.Players.Broadcast("cloudychat.helpop.view", msg)
	invokeMessageHook(handler, "helpop", strings.Join(args[1:], " "))

	helpopHistoryMutex.Lock()
	now := time.Now().Format(helpopTimeFormat)
	msg = fmt.Sprintf(helpopHistoryFormat, now, handler.UpstreamName, handler.Nickname, strings.Join(args[1:], " "))
	n := copy(helpopHistory[:], helpopHistory[1:])
	helpopHistory[n] = msg // chyba bedzie dzialac z tym n xD
	helpopHistoryMutex.Unlock()

	return potoq.ErrDropPacket
}
