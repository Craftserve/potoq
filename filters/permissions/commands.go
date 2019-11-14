package permissions

import (
	"fmt"
	"strings"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

const INSUFFICIENT_PERMS = packets.COLOR_RED + "Insufficient permissions!"

func onChatMessage(handler *potoq.Handler, packet packets.Packet) error {
	msg := packet.(*packets.ChatMessagePacketSB)

	var resp string

	if !strings.HasPrefix(msg.Message, "/gperm ") {
		return nil
	}

	words := strings.Fields(msg.Message)
	if len(words) >= 2 {
		switch strings.ToLower(words[1]) {
		case "reload":
			resp = reloadCmd(handler, words[2:])
		case "stats":
			resp = statsCmd(handler, words[2:])
		default:
			resp = packets.COLOR_RED + "Unkown /gperm subcommand!"
		}
	} else {
		resp = packets.COLOR_RED + "Usage: /gperm <subcommand>!"
	}

	handler.SendChatMessage(resp)
	return potoq.ErrDropPacket
}

func reloadCmd(handler *potoq.Handler, args []string) string {
	if !handler.HasPermission("permissions.reload") {
		return INSUFFICIENT_PERMS
	}

	err := LoadPermissions()
	if err != nil {
		return packets.COLOR_RED + "Permission loading error: " + err.Error()
	}
	return packets.COLOR_GREEN + "Permissions reloaded!"
}

func statsCmd(handler *potoq.Handler, args []string) string {
	if !handler.HasPermission("permissions.stats") {
		return INSUFFICIENT_PERMS
	}
	if len(args) != 1 {
		return packets.COLOR_RED + "Usage: /gperm stats <permission>"
	}

	var all, set int
	potoq.Players.Range(func(player *potoq.Handler) bool {
		all++
		if player.HasPermission(args[0]) {
			set++
		}
		return true
	})

	return fmt.Sprintf(packets.COLOR_GREEN+"Permission %s: %d / %d -> %.2f%%", args[0], set, all, (float32(set)/float32(all))*100.0)
}
