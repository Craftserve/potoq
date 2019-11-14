package packets

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

type PacketReader interface {
	ReadPacket() (RawPacket, error)
	Payload() (io.Reader, error)
}

type readResetter interface {
	io.ReadCloser
	zlib.Resetter
}

type byter interface {
	io.Reader
	io.ByteReader
}

type packetReader struct {
	compthres VarInt
	input     io.Reader
	byter     byter
	zliber    readResetter
	slicer    bytes.Reader
	payload   io.Reader
	err       error
}

func NewPacketReader(input io.Reader, compressThreshold int) PacketReader {
	r := &packetReader{
		compthres: VarInt(compressThreshold),
		input:     input,
	}
	if br, ok := input.(byter); ok {
		r.byter = br
	} else {
		r.byter = dummyByteReader{Reader: input}
	}
	return r
}

func (r *packetReader) ReadPacket() (RawPacket, error) {
	if r.err != nil {
		return RawPacket{}, r.err
	}
	r.slicer.Reset(nil) // free pointers to last packets, we'll probably be locked on network below

	var packet_length VarInt
	packet_length, r.err = ReadVarInt(r.byter)
	if r.err != nil {
		return RawPacket{}, r.err
	}
	if packet_length > MaxPacketSize {
		r.err = fmt.Errorf("Packet is too big! %d > %d", packet_length, MaxPacketSize)
		return RawPacket{}, r.err
	}

	var raw = RawPacket{
		Payload: BufferPool.Get(int(packet_length)),
	}
	_, r.err = io.ReadFull(r.input, raw.Payload)
	if r.err != nil {
		return RawPacket{}, r.err
	}
	r.slicer.Reset(raw.Payload)

	if r.compthres > 0 {
		raw.DataLength, r.err = ReadVarInt(&r.slicer)
		if r.err != nil {
			return RawPacket{}, r.err
		}
		if raw.DataLength > 0 && raw.DataLength < r.compthres {
			r.err = fmt.Errorf("Compressed data below threshold! %d < %d", raw.DataLength, r.compthres)
			return RawPacket{}, r.err
		}
	}

	if raw.DataLength > 0 {
		if r.zliber == nil {
			var readCloser io.ReadCloser
			readCloser, r.err = zlib.NewReader(&r.slicer)
			if r.err == nil {
				r.zliber = readCloser.(readResetter)
			}
		} else {
			r.err = r.zliber.Reset(&r.slicer, nil)
		}
		if r.err != nil {
			return RawPacket{}, r.err
		}
		r.payload = r.zliber
	} else {
		r.payload = &r.slicer
	}

	raw.ID, r.err = ReadVarInt(r.payload) // TODO: to alokuje dummyByteReader za kazdym razem
	if r.err != nil {
		return RawPacket{}, r.err
	}
	if raw.ID > MaxPacketID || raw.ID < 0 {
		r.err = fmt.Errorf("Invalid packet_id %d", raw.ID)
		return RawPacket{}, r.err
	}

	return raw, nil
}

// this io.Reader is valid between calls to ReadPacket
func (r *packetReader) Payload() (io.Reader, error) {
	if r.payload == r.zliber {
		var buf = bytes.NewBuffer(nil)
		_, r.err = io.Copy(buf, r.zliber)
		if r.err != nil {
			return nil, r.err
		}
		r.payload = buf
	}
	return r.payload, nil
}

// func (w *packetWriter) HijackInput() io.Reader {
// 	return w.input
// }

type ErrUnexpectedPacket struct {
	ID VarInt
}

func (e ErrUnexpectedPacket) Error() string {
	return fmt.Sprintf("Unexpected Packet 0x%02X", e.ID)
}

func ParsePackets(reader PacketReader, packets ...Packet) (Packet, error) {
	raw, err := reader.ReadPacket()
	if err != nil {
		return nil, err
	}
	for _, packet := range packets {
		if raw.PacketID() != packet.PacketID() {
			continue
		}
		payload, err := reader.Payload()
		if err != nil {
			return nil, err
		}
		return packet, packet.Parse(payload)
	}
	return raw, ErrUnexpectedPacket{ID: raw.PacketID()}
}
