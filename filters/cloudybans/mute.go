package cloudybans

import "fmt"
import "time"
import "strings"

import "github.com/Craftserve/potoq"
import "github.com/Craftserve/potoq/packets"

func MuteFilter(handler *potoq.Handler, packet packets.Packet) error {
	msg := packet.(*packets.ChatMessagePacketSB)

	if strings.HasPrefix(msg.Message, "/") {
		return nil // pass commands
	}

	mute, _ := GetAbuse("mute", handler.Nickname, true)
	if mute == nil {
		return nil // not muted
	}

	resp := fmt.Sprintf(packets.COLOR_RED+"Nie mozesz pisac na chacie! Zostales wyciszony przez %s do %s z powodu '%s'", mute.Moderator, mute.Expires.Format(time.RFC822), mute.Comment)
	handler.SendChatMessage(resp)
	return potoq.ErrDropPacket
}
