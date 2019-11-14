package bungeecord

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/filters/permissions"
	"github.com/Craftserve/potoq/packets"
)

const INSUFFICIENT_PERMS = packets.COLOR_RED + "Insufficient permissions!"

// tutaj maja byc obslugiwane polecenia z http://www.spigotmc.org/wiki/bungeecord-commands/
func ChatCommands(handler *potoq.Handler, packet packets.Packet) error {
	msg := packet.(*packets.ChatMessagePacketSB)

	var resp string = ""
	words := strings.Split(msg.Message, " ")

	switch strings.ToLower(words[0]) {
	case "/server":
		resp = serverCmd(handler, words[1:])
	case "/send":
		resp = sendCmd(handler, words[1:])
	case "/alert":
		resp = alertCmd(handler, words[1:])
	case "/glist":
		resp = glistCmd(handler, words[1:])
	case "/ip":
		resp = ipCmd(handler, words[1:])
	case "/find":
		resp = findCmd(handler, words[1:])
	case "/gperms":
		resp = permsCmd(handler, words[1:])
	case "/proxystats":
		resp = proxystatsCmd(handler, words[1:])
	case "/greload":
		resp = reloadupstreamsCmd(handler, words[1:])
	case "/bungee":
		resp = packets.COLOR_RED + "To nie bungee, czego tu szukasz? :P"
	default:
		return nil // unknown command, just pass it
	}

	if resp != "" {
		handler.SendChatMessage(resp)
	}

	return potoq.ErrDropPacket
}

func reloadupstreamsCmd(handler *potoq.Handler, args []string) string {
	if len(args) != 0 {
		return fmt.Sprint(packets.COLOR_RED, "Poprawne uzycie: /greload")
	}

	if !handler.HasPermission("bungeecord.command.greload") {
		return INSUFFICIENT_PERMS
	}

	err := potoq.LoadUpstreams("upstreams.yml")
	if err != nil {
		return fmt.Sprintf("Blad podczas przeladowywania! Error: %s", err.Error())
	}

	return "Przeladowano pomyslnie!"
}

func proxystatsCmd(handler *potoq.Handler, args []string) string {
	if len(args) != 0 {
		return fmt.Sprint(packets.COLOR_RED, "Poprawne uzycie: /proxystats")
	}

	if !handler.HasPermission("bungeecord.command.proxystats") {
		return INSUFFICIENT_PERMS
	}

	downPack := atomic.LoadUint64(&handler.DownstreamPacketsCount)
	upPack := atomic.LoadUint64(&handler.UpstreamPacketsCount)

	go func() {
		time.Sleep(time.Second)

		cDownPack := atomic.LoadUint64(&handler.DownstreamPacketsCount)
		cUpPack := atomic.LoadUint64(&handler.UpstreamPacketsCount)

		handler.SendChatMessage(fmt.Sprint(packets.COLOR_GREEN, "Przeslane pakiety w ciagu ostatniej sekundy:"))
		handler.SendChatMessage(fmt.Sprint(packets.COLOR_GREEN, "C -> P: ", (cDownPack - downPack)))
		handler.SendChatMessage(fmt.Sprint(packets.COLOR_GREEN, "P -> S: ", (cUpPack - upPack)))

	}()

	handler.SendChatMessage(fmt.Sprint(packets.COLOR_GREEN, "Lacznie przeslane pakiety:"))
	handler.SendChatMessage(fmt.Sprint(packets.COLOR_GREEN, "C -> P: ", downPack))
	handler.SendChatMessage(fmt.Sprint(packets.COLOR_GREEN, "P -> S: ", upPack))
	handler.SendChatMessage(fmt.Sprint(packets.COLOR_GREEN, "Oczekiwanie na czasowe wyniki..."))

	return ""
}

func serverCmd(handler *potoq.Handler, args []string) string {
	switch len(args) {
	case 0:
		var servers string = "<hidden>"
		if handler.HasPermission("bungeecord.command.server") {
			servers = strings.Join(serverNames(), ", ")
		}
		return fmt.Sprintf("%sYour server is: %s\n%sOther servers: %s", packets.COLOR_GREEN, handler.UpstreamName, packets.COLOR_GREEN, servers)
	case 1:
		if !handler.HasPermission("bungeecord.command.server.switch") {
			return INSUFFICIENT_PERMS
		}
		potoq.UpstreamLock.Lock()
		reconnect := potoq.UpstreamServerMap[args[0]]
		potoq.UpstreamLock.Unlock()
		if reconnect != nil {
			handler.PushCommand(reconnect, true)
			return "Reconnected to: " + args[0]
		} else {
			return "Unkown server: " + args[0]
		}
	}
	return "Bad command syntax!"
}

func sendCmd(handler *potoq.Handler, args []string) string {
	if !handler.HasPermission("bungeecord.command.send") {
		return INSUFFICIENT_PERMS
	}
	if len(args) != 2 {
		return packets.COLOR_RED + "Bad command syntax! /send <player|current|all> <target>"
	}

	potoq.UpstreamLock.Lock()
	reconnect := potoq.UpstreamServerMap[args[1]]
	potoq.UpstreamLock.Unlock()
	if reconnect == nil {
		return packets.COLOR_RED + "Unkown server: " + args[1]
	}

	if (args[0] == "all") || (args[0] == "current") {
		var upstream string = ""
		if args[0] == "current" {
			upstream = handler.UpstreamName
		}

		var count int
		potoq.Players.Range(func(player *potoq.Handler) bool {
			if player.UpstreamName == upstream || upstream == "" {
				if player.PushCommand(reconnect, false) {
					count += 1
				}
			}
			return true
		})

		return fmt.Sprintf(packets.COLOR_GREEN+"Sent %d players to %s", count, reconnect.Name)
	} else { // one player
		player := potoq.Players.GetByNickname(args[0])
		if player == nil {
			return packets.COLOR_RED + "Player not found: " + args[0]
		}
		if player.PushCommand(reconnect, false) {
			return packets.COLOR_GREEN + "Sent player " + player.Nickname + " to " + args[1]
		} else {
			return packets.COLOR_RED + "Cannot send player " + player.Nickname + " to " + args[1]
		}
	}
}

func alertCmd(handler *potoq.Handler, args []string) string {
	if !handler.HasPermission("bungeecord.command.alert") {
		return INSUFFICIENT_PERMS
	}
	if len(args) < 1 {
		return packets.COLOR_RED + "Bad command syntax! /alert <message...>"
	}
	potoq.Players.Broadcast("", packets.COLOR_RED+"UWAGA: "+strings.Join(args, " "))
	return ""
}

func glistCmd(handler *potoq.Handler, args []string) string {
	if !handler.HasPermission("bungeecord.command.glist") {
		return INSUFFICIENT_PERMS
	}

	potoq.UpstreamLock.Lock()
	resp := make([]string, 1, 2+len(potoq.UpstreamServerMap))
	potoq.UpstreamLock.Unlock()
	resp[0] = packets.COLOR_GREEN + "Upstream Servers:"

	sum := 0
	counts := make(map[string]int)
	potoq.UpstreamLock.Lock()
	for k, _ := range potoq.UpstreamServerMap {
		counts[k] = 0
	}
	potoq.UpstreamLock.Unlock()

	potoq.Players.Range(func(player *potoq.Handler) bool {
		counts[player.UpstreamName] += 1
		sum += 1
		return true
	})

	for k, v := range counts {
		s := fmt.Sprintf("%s> %s: %d", packets.COLOR_GREEN, k, v)
		resp = append(resp, s)
	}
	resp = append(resp, fmt.Sprintf("%s> ONLINE SUM: %d", packets.COLOR_GREEN, sum))

	return strings.Join(resp, "\n")
}

func ipCmd(handler *potoq.Handler, args []string) string {
	if len(args) > 0 {
		if !handler.HasPermission("bungeecord.command.ip") {
			return INSUFFICIENT_PERMS
		}
		player := potoq.Players.GetByNickname(args[0])
		if player == nil {
			return packets.COLOR_RED + "Player not found!"
		}
		return fmt.Sprintf("%sIP of %s is: %s", packets.COLOR_GREEN, player.Nickname, player.DownstreamAddr)
	} else {
		return fmt.Sprintf("%sYour IP is: %s", packets.COLOR_GREEN, handler.DownstreamAddr)
	}
}

func permsCmd(handler *potoq.Handler, args []string) string {
	if !handler.HasPermission("bungeecord.command.perms") {
		return INSUFFICIENT_PERMS
	}

	var player *potoq.Handler
	if len(args) > 0 {
		player = potoq.Players.GetByNickname(args[0])
		if player == nil {
			return packets.COLOR_RED + "Player not found!"
		}
	} else {
		player = handler
	}

	permset, ok := player.Permissions.(permissions.PermSet)
	if !ok {
		return packets.COLOR_RED + "Invalid handler.Permissions type"
	}

	// uzywanie handler.Permissions handlera innego niz my bez synchronizacji nie jest zbyt madre
	// bo to zwykla mapa, ale jak narazie to nie modyfikujemy ich po zalogowaniu wiec powinno byc ok
	var b strings.Builder
	fmt.Fprintf(&b, "%sPermissions of %s:\n", packets.COLOR_GREEN, player.Nickname)
	for k, v := range permset {
		fmt.Fprintf(&b, "%s> %s: %t\n", packets.COLOR_GREEN, k, v)
	}
	return b.String()
}

func findCmd(handler *potoq.Handler, args []string) string {
	if !handler.HasPermission("bungeecord.command.find") {
		return INSUFFICIENT_PERMS
	}
	if len(args) != 1 {
		return packets.COLOR_RED + "Bad command syntax! /find <nickname>"
	}

	player := potoq.Players.GetByNickname(args[0])
	if player == nil {
		return packets.COLOR_RED + "Player not found!"
	}
	return fmt.Sprintf("%sPlayer %s is on %s", packets.COLOR_GREEN, player.Nickname, player.UpstreamName)
}

func serverNames() []string {
	var names []string
	potoq.UpstreamLock.Lock()
	for _, v := range potoq.UpstreamServerMap {
		names = append(names, v.Name)
	}
	potoq.UpstreamLock.Unlock()
	return names
}
