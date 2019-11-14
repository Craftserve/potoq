package packets

import (
	"encoding/json"
	"io"
)

type ServerStatus struct {
	Version     ServerStatusVersion `json:"version"`
	Players     ServerStatusPlayers `json:"players"`
	Description string              `json:"description"`
	Favicon     string              `json:"favicon"`
}

type ServerStatusVersion struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

type ServerStatusPlayers struct {
	Online int `json:"online"`
	Max    int `json:"max"`
}

func (status ServerStatus) Serialize() (string, error) {
	bytes, err := json.Marshal(status)
	return string(bytes), err
}

// C->S StatusRequestPacket

type StatusRequestPacketSB struct {
}

func (packet *StatusRequestPacketSB) PacketID() VarInt {
	return 0x00
}

func (packet *StatusRequestPacketSB) Parse(reader io.Reader) (err error) {
	return nil
}

func (packet *StatusRequestPacketSB) Serialize(writer io.Writer) error {
	return nil
}

func (packet *StatusRequestPacketSB) Direction() Direction {
	return ServerBound
}

// S->C StatusRequestPacket

type StatusResponsePacketCB struct {
	Data string `max_length:"32767"`
}

func (packet *StatusResponsePacketCB) PacketID() VarInt {
	return 0x00
}

func (packet *StatusResponsePacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *StatusResponsePacketCB) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *StatusResponsePacketCB) Direction() Direction {
	return ClientBound
}

// Two-way StatusPingPacket

type StatusPingPacketCB struct {
	Time int64
}

func (packet *StatusPingPacketCB) PacketID() VarInt {
	return 0x01
}

func (packet *StatusPingPacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *StatusPingPacketCB) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *StatusPingPacketCB) Direction() Direction {
	return ServerBound
}

type StatusPingPacketSB struct {
	Time int64
}

func (packet *StatusPingPacketSB) PacketID() VarInt {
	return 0x01
}

func (packet *StatusPingPacketSB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *StatusPingPacketSB) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *StatusPingPacketSB) Direction() Direction {
	return ClientBound
}
