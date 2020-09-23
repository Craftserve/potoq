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
	RegisterPacketFilter(&packets.ClientSettingsPacketSB{}, saveClientSettings)
	RegisterPacketFilter(&packets.PluginMessagePacketSB{}, saveClientSettings)
	RegisterPacketFilter(&packets.PlayerListItemPacketCB{}, savePlayerList)
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

func updatePlayerListItem(handler *Handler, items []packets.PlayerListItem,
	changeData func(savedItem, newItem *packets.PlayerListItem)) {

	for _, item := range items {
		savedItem, ok := handler.playerList[item.UUID]
		if !ok {
			continue
		}
		changeData(&savedItem, &item)
		handler.playerList[item.UUID] = savedItem
	}
}

func savePlayerList(handler *Handler, packet packets.Packet) error {
	playerListPacket := packet.(*packets.PlayerListItemPacketCB)
	action := playerListPacket.Action
	switch action {
	case packets.ADD_PLAYER:
		for _, item := range playerListPacket.Items {
			handler.playerList[item.UUID] = item
		}
		break
	case packets.UPDATE_GAME_MODE:
		updatePlayerListItem(handler, playerListPacket.Items, func(savedItem, newItem *packets.PlayerListItem) {
			savedItem.GameMode = newItem.GameMode
		})
		break
	case packets.UPDATE_DISPLAY_NAME:
		updatePlayerListItem(handler, playerListPacket.Items, func(savedItem, newItem *packets.PlayerListItem) {
			savedItem.DisplayName = newItem.DisplayName
		})
		break
	case packets.UPDATE_LATENCY:
		updatePlayerListItem(handler, playerListPacket.Items, func(savedItem, newItem *packets.PlayerListItem) {
			savedItem.Ping = newItem.Ping
		})
		break
	case packets.REMOVE_PLAYER:
		for _, item := range playerListPacket.Items {
			delete(handler.playerList, item.UUID)
		}
		break
	}
	return nil
}
