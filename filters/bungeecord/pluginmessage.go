package bungeecord

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

func PluginMessage(handler *potoq.Handler, packet packets.Packet) error {
	msg := packet.(*packets.PluginMessagePacketCB)
	if msg.Channel != "bungeecord:main" {
		return nil
	}

	reader := msg.MakePayloadReader()
	subchannel, _ := ReadJavaUTF(reader)
	var output bytes.Buffer

	handler.Log().WithFields(log.Fields{
		"subchannel": subchannel,
		"length": reader.Len(),
	}).Debug("BungeeCord_PluginMessage")

	switch subchannel {
	/*case "Forward":
	var target, fwd_channel string
	target, _ = ReadJavaUTF(reader)
	fwd_channel, err = ReadJavaUTF(reader)
	if err != nil {
		return
	}

	WriteJavaUTF(&output, fwd_channel)
	io.Copy(output, reader)

	inject := &potoq.InjectPacketCommand{
		Direction: packets.ServerBound,
		Payload:   &packets.PluginMessagePacketSB{Channel: "BungeeCord", Payload: output.Bytes()},
	}
	output = nil // we dont want to send packet back to source

	done_upstreams := make(map[string]bool)

	potoq.Players.Lock()
	for _, h := range potoq.Players.M {
		if ((target == "ALL") || (h.UpstreamName == target)) && (h != handler) && !done_upstreams[h.UpstreamName] {
			done_upstreams[h.UpstreamName] = h.PushCommand(inject, false) // TODO: Nie bierzemy tu w ogole pod uwage tego, ze upstream moze sie zmienic,
			// wiec dobrze by bylo dopisac weryfikacje w InjectPacketCommand.execute(), albo dropowac wszystkie zbuforowane polecenia po reconnect
		}
	}
	potoq.Players.Unlock()*/
	case "Connect":
		upstream_name, err := ReadJavaUTF(reader)
		if err != nil {
			return err
		}
		potoq.UpstreamLock.Lock()
		cmd, ok := potoq.UpstreamServerMap[upstream_name]
		potoq.UpstreamLock.Unlock()
		if !ok {
			return fmt.Errorf("Unknown bungee upstream name: %s", upstream_name)
		}
		handler.PushCommand(cmd, true)
	case "ConnectOther":
		nickname, err := ReadJavaUTF(reader)
		if err != nil {
			return err
		}
		server, err := ReadJavaUTF(reader)
		if err != nil {
			return err
		}
		potoq.UpstreamLock.Lock()
		cmd, ok := potoq.UpstreamServerMap[server]
		potoq.UpstreamLock.Unlock()
		if !ok {
			return fmt.Errorf("Unknown bungee upstream server: %s", server)
		}
		other := potoq.Players.GetByNickname(nickname)
		if other == nil {
			return potoq.ErrDropPacket
		}
		if !other.PushCommand(cmd, false) {
			handler.Log().WithFields(log.Fields{
				"target": server,
				"nickname": nickname,
			}).Warn("Cannot inject ConnectOther")
		}
	case "IP":
		WriteJavaUTF(&output, "IP")
		WriteJavaUTF(&output, handler.DownstreamAddr)
		var port int32
		if addr, ok := handler.DownstreamC.RemoteAddr().(*net.TCPAddr); ok {
			port = int32(addr.Port)
		}
		packets.WriteInt(&output, port)
	case "PlayerCount":
		server, err := ReadJavaUTF(reader)
		if err != nil {
			return err
		}

		var count int32
		potoq.Players.Range(func(player *potoq.Handler) bool {
			if server == "ALL" || player.UpstreamName == server {
				count++
			}
			return true
		})

		WriteJavaUTF(&output, "PlayerCount")
		WriteJavaUTF(&output, server)
		packets.WriteInt(&output, count)
	case "PlayerList":
		server, err := ReadJavaUTF(reader)
		if err != nil {
			return err
		}

		var nicknames []string
		potoq.Players.Range(func(player *potoq.Handler) bool {
			if server == "ALL" || player.UpstreamName == server {
				nicknames = append(nicknames, player.Nickname)
			}
			return true
		})

		WriteJavaUTF(&output, "PlayerList")
		WriteJavaUTF(&output, server)
		WriteJavaUTF(&output, strings.Join(nicknames, ","))
	case "GetServer":
		WriteJavaUTF(&output, "GetServer")
		WriteJavaUTF(&output, handler.UpstreamName)
	case "GetServers":
		potoq.UpstreamLock.Lock()
		list := make([]string, 0, len(potoq.UpstreamServerMap))
		for k, _ := range potoq.UpstreamServerMap {
			list = append(list, k)
		}
		potoq.UpstreamLock.Unlock()
		WriteJavaUTF(&output, "GetServers")
		WriteJavaUTF(&output, strings.Join(list, ","))
	case "Message":
		nickname, err := ReadJavaUTF(reader)
		if err != nil {
			return err
		}
		other := potoq.Players.GetByNickname(nickname)
		if other == nil {
			return potoq.ErrDropPacket
		}
		msg, err := ReadJavaUTF(reader)
		if err != nil {
			return err
		}
		other.SendChatMessage(msg)
	case "Broadcast", "Broadcast2":
		msg, err := ReadJavaUTF(reader)
		if err != nil {
			return err
		}
		var perm string
		if subchannel == "Broadcast2" {
			perm, err = ReadJavaUTF(reader)
			if err != nil {
				return err
			}
		}
		potoq.Players.Broadcast(perm, msg)
	default:
		return fmt.Errorf("Unknown BungeeCord subchannel: %s", subchannel)
	}

	if output.Len() > 0 {
		resp := &packets.PluginMessagePacketSB{"bungeecord:main", output.Bytes()}
		err := handler.InjectPackets(packets.ServerBound, nil, resp)
		if err != nil {
			return err
		}
	}

	return potoq.ErrDropPacket
}
