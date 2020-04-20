package packets

import (
	"fmt"
	"io"
)

const ProtocolVersion = 578
const MaxPacketSize = 128 * 1024
const MaxPacketID = 256
const CompressThreshold = 256
var GameVersion = ServerStatusVersion{"1.15.2", ProtocolVersion}

type EntityID int32
type VarInt int32

type Packet interface {
	PacketID() VarInt
	Direction() Direction
	Parse(reader io.Reader) error
	Serialize(writer io.Writer) error
}

type Marshaler interface {
	MarshalPacket(w io.Writer) error
}

type Unmarshaler interface {
	UnmarshalPacket(w io.Writer) error
}

func NewPacket(packet_id VarInt, state ConnState, direction Direction) (packet Packet) {
	if state == HANDSHAKING && direction == ServerBound && packet_id == 0 {
		packet = new(HandshakePacket)
	} else if state == STATUS {
		if direction == ServerBound {
			switch packet_id {
			case 0x00:
				packet = new(StatusRequestPacketSB)
			case 0x01:
				packet = new(StatusPingPacketSB)
			}
		} else {
			switch packet_id {
			case 0x00:
				packet = new(StatusResponsePacketCB)
			case 0x01:
				packet = new(StatusPingPacketCB)
			}
		}
	} else if state == LOGIN {
		if direction == ServerBound {
			switch packet_id {
			case 0x00:
				packet = new(LoginStartPacket)
			case 0x01:
				packet = new(EncryptionResponsePacket)
			}
		} else {
			switch packet_id {
			case 0x00:
				packet = new(LoginKickPacket)
			case 0x01:
				packet = new(EncryptionRequestPacket)
			case 0x02:
				packet = new(LoginSuccessPacket)
			case 0x03:
				packet = new(LoginCompressionPacket)
			}
		}
	} else if state == PLAY {
		packet = newPlayPacket(packet_id, direction)
	}

	if packet != nil {
		if packet.PacketID() != packet_id {
			panic(fmt.Sprintf("Packet id mismatch %02X != %T", packet_id, packet))
		}
		if packet.Direction() != direction {
			panic(fmt.Sprintf("Packet direction mismatch %T", packet))
		}
	}
	return
}

func ToString(packet Packet, direction Direction) string {
	var s string
	if raw, is_raw := packet.(RawPacket); is_raw {
		if len(raw.Payload) < 48 {
			s = fmt.Sprintf("%X", raw.Payload)
		} else {
			s = fmt.Sprintf("RAW TOO_LARGE %d", len(raw.Payload))
		}
	} else { // not raw
		s = fmt.Sprintf("%#v", packet)
		if len(s) > 256 {
			s = fmt.Sprintf("%T", packet)
		}
	}
	return fmt.Sprintf("%s_%02X %s", direction, packet.PacketID(), s)
}

func newPlayPacket(packet_id VarInt, direction Direction) (packet Packet) {
	if direction == ServerBound {
		switch packet_id {
		case 0x03:
			packet = new(ChatMessagePacketSB)
		case 0x04:
			packet = new(ClientStatusPacketSB)
		case 0x05:
			packet = new(ClientSettingsPacketSB)
		case 0x06:
			packet = new(TabCompletePacketSB)
		case 0x0B:
			packet = new(PluginMessagePacketSB)
		}
	} else {
		switch packet_id {
		case 0x3B:
			packet = new(RespawnPacketCB)
		case 0x1B:
			packet = new(KickPacketCB)
		case 0x26:
			packet = new(JoinGamePacketCB)
		case 0x19:
			packet = new(PluginMessagePacketCB)
		case 0x1F:
			packet = new(GameStateChangePacketCB)
		case 0x0F:
			packet = new(ChatMessagePacketCB)
		case 0x11:
			packet = new(TabCompletePacketCB) //TODO Przyjrzeć się temu pakietowi
		case 0x34:
			packet = new(PlayerListItemPacketCB)
		case 0x4A:
			packet = new(ScoreboardObjectivePacketCB)
		case 0x4C:
			packet = new(TeamsPacketCB) //TODO Przyjrzeć się temu pakietowi
		}
	}
	return
}

type Position uint64

// struct {
// 	X int32
// 	Y int32
// 	Z int32
// }
