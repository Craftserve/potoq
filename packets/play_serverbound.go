package packets

import (
	"bytes"
	"io"
)

// > 0x04 ClientSettingsPacketSB

type ClientSettingsPacketSB struct {
	Locale       string `max_length:"16"`
	ViewDistance byte
	ChatMode     VarInt
	ChatColors   bool
	SkinParts    uint8
	Hand         VarInt
}

func (packet *ClientSettingsPacketSB) PacketID() VarInt {
	return 0x05
}

func (packet *ClientSettingsPacketSB) Direction() Direction {
	return ServerBound
}

func (packet *ClientSettingsPacketSB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *ClientSettingsPacketSB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}

// > 0x04 ClientStatusPacketSB

type ClientStatusPacketSB struct {
	Action VarInt
}

func (packet *ClientStatusPacketSB) PacketID() VarInt {
	return 0x04
}

func (packet *ClientStatusPacketSB) Direction() Direction {
	return ServerBound
}

func (packet *ClientStatusPacketSB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *ClientStatusPacketSB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}

// > 0x0B PluginMessagePacketSB

type PluginMessagePacketSB struct {
	Channel string `max_length:"64"`
	Payload []byte `max_length:"35000" length_prefix:"eof"`
}

func (packet *PluginMessagePacketSB) PacketID() VarInt {
	return 0x0B
}

func (packet *PluginMessagePacketSB) Direction() Direction {
	return ServerBound
}

func (packet *PluginMessagePacketSB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *PluginMessagePacketSB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *PluginMessagePacketSB) MakePayloadReader() *bytes.Reader {
	return bytes.NewReader(packet.Payload)
}

// > 0x03 ChatMessagePacketSB

type ChatMessagePacketSB struct {
	Message string `max_length:"256"`
}

func (packet *ChatMessagePacketSB) PacketID() VarInt {
	return 0x03
}

func (packet *ChatMessagePacketSB) Direction() Direction {
	return ServerBound
}

func (packet *ChatMessagePacketSB) Parse(reader io.Reader) error {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *ChatMessagePacketSB) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

// > 0x06 TabCompletePacketSB

type TabCompletePacketSB struct {
	TransactionId VarInt
	Text          string `max_length:"32500"`
}

func (packet *TabCompletePacketSB) PacketID() VarInt {
	return 0x06
}

func (packet *TabCompletePacketSB) Direction() Direction {
	return ServerBound
}

func (packet *TabCompletePacketSB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *TabCompletePacketSB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}
