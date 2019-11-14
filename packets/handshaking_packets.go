package packets

import "io"

type HandshakePacket struct {
	Protocol  VarInt
	Host      string `max_length:"255"`
	Port      int16
	NextState VarInt
}

func (packet *HandshakePacket) PacketID() VarInt {
	return 0x00
}

func (packet *HandshakePacket) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *HandshakePacket) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *HandshakePacket) Direction() Direction {
	return ServerBound
}
