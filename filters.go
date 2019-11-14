package potoq

import (
	"errors"

	"github.com/Craftserve/potoq/packets"
	"github.com/Craftserve/potoq/utils"
)

type PacketFilter func(handler *Handler, packet packets.Packet) error

var ErrDropPacket = errors.New("ErrDropPacket") // can be returned from PacketFilter

// first half of both arrays is CB, second SB, see RegisterPacketFilter()
var packetFilters [packets.MaxPacketID * 2][]PacketFilter
var packetsParsed = utils.NewBitarray(packets.MaxPacketID * 2)

func init() {
	RegisterPacketFilter(&packets.PlayerListItemPacketCB{}, authProperties)
	RegisterPacketFilter(&packets.ClientSettingsPacketSB{}, saveClientSettings)
	RegisterPacketFilter(&packets.PluginMessagePacketSB{}, saveClientSettings)
}

func RegisterPacketFilter(packet_type packets.Packet, callback PacketFilter) {
	packet_id := packet_type.PacketID()
	switch packet_type.Direction() {
	case packets.ServerBound:
		var i = packet_id
		packetFilters[i] = append(packetFilters[i], callback)
		packetsParsed.Set(int(i), true)
	case packets.ClientBound:
		var i = packet_id + 256
		packetFilters[i] = append(packetFilters[i], callback)
		packetsParsed.Set(int(i), true)
	default:
		panic("invalid direction")
	}
}

// implements skin support
func authProperties(handler *Handler, packet packets.Packet) error {
	plst := packet.(*packets.PlayerListItemPacketCB)
	if plst.Action != packets.ADD_PLAYER {
		return nil
	}

	Players.Range(func(other *Handler) bool {
		for i := range plst.Items { // []PlayerListItem, nie wskazniki
			item := &plst.Items[i]
			if item.UUID == other.UUID {
				item.Properties = append(item.Properties, other.AuthProperties...)
			}
		}
		return true
	})

	return nil
}

func saveClientSettings(handler *Handler, packet packets.Packet) error {
	switch p := packet.(type) {
	case *packets.ClientSettingsPacketSB:
		handler.ClientSettings = p
	case *packets.PluginMessagePacketSB:
		if p.Channel == "minecraft:brand" {
			handler.MCBrand = p
		}
	default:
		panic("unexpected packet")
	}
	return nil
}
