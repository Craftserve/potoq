package packets

import (
	"io"

	bufpool "github.com/libp2p/go-buffer-pool"
)

var BufferPool bufpool.BufferPool

type RawPacket struct {
	ID         VarInt
	DataLength VarInt
	Payload    []byte // includes uncompressed DataLength
}

func (packet RawPacket) PacketID() VarInt {
	return packet.ID
}

func (packet RawPacket) Direction() Direction {
	panic("RawPacket is just a dummy container type - valid directions are unknown!")
}

func (packet RawPacket) Parse(reader io.Reader) (err error) {
	panic("RawPacket is just a dummy container type - can't parse real data stream!")
}

func (packet RawPacket) Serialize(writer io.Writer) (err error) {
	panic("RawPacket is just a dummy container type - can't serialize!")
}

// func (packet *RawPacket) ReadRawData(reader io.Reader) (err error) {
// 	if packet.Payload == nil {
// 		packet.Payload = make([]byte, MaxPacketSize) // allocate 128k reusable buffer.
// 	}
// 	packet.Payload = packet.Payload[:cap(packet.Payload)]
// 	var done, n int
// 	for err == nil {
// 		n, err = reader.Read(packet.Payload[done:])
// 		done += n
// 	}
// 	packet.Payload = packet.Payload[:done]
// 	if done == cap(packet.Payload) {
// 		err = fmt.Errorf("RawPacket.ReadRawData: packet bigger than max packet size! %d", n)
// 	}
// 	if err == io.EOF {
// 		err = nil
// 	}
// 	return
// }
