package bungeecord

import "github.com/Craftserve/potoq"
import "github.com/Craftserve/potoq/packets"

// based on https://github.com/SpigotMC/BungeeCord/blob/d8c92cd3115457164a974d9c8ac091caba5699c3/proxy/src/main/java/net/md_5/bungee/connection/DownstreamBridge.java#L188

func RegisterFilters() {
	potoq.RegisterPacketFilter(&packets.JoinGamePacketCB{}, joinFilter)
	potoq.RegisterPacketFilter(&packets.PluginMessagePacketCB{}, PluginMessage)
	potoq.RegisterPacketFilter(&packets.ChatMessagePacketSB{}, ChatCommands)
}

func joinFilter(handler *potoq.Handler, packet packets.Packet) error {
	return handler.InjectPackets(packets.ServerBound, nil, &packets.PluginMessagePacketSB{"minecraft:register", []byte("bungeecord:main\x00")})
}
