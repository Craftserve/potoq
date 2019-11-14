package packets

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
)

type PacketWriter interface {
	WritePacket(packet Packet, flush bool) error
	Flush() error
}

type flusher interface {
	Flush() error
}

type packetWriter struct {
	err       error
	out       io.Writer
	flusher   flusher
	buf       bytes.Buffer
	zliber    *zlib.Writer
	compthres int
}

func NewPacketWriter(out io.Writer, compress_threshold int) PacketWriter {
	w := &packetWriter{
		out:       out,
		buf:       *bytes.NewBuffer(BufferPool.Get(MaxPacketSize)),
		compthres: compress_threshold,
	}
	if flusher, ok := out.(flusher); ok {
		w.flusher = flusher
	}
	return w
}

func (w *packetWriter) WritePacket(packet Packet, flush bool) error {
	if w.err != nil {
		return w.err
	}

	if raw, is_raw := packet.(RawPacket); is_raw {
		w.err = WriteVarInt(w.out, VarInt(len(raw.Payload))) // packet length
		if w.err != nil {
			return w.err
		}
		_, w.err = w.out.Write(raw.Payload)
		if flush {
			return w.Flush()
		}
		return w.err
	}

	// serialize packet to buffer
	w.buf.Reset()
	w.err = WriteVarInt(&w.buf, packet.PacketID())
	if w.err != nil {
		return w.err
	}
	w.err = packet.Serialize(&w.buf) // dodac tutaj sprawdzanie czy pakiet implementuje MarshalPacket, jak nie to structy
	if w.err != nil {
		return w.err
	}

	if w.compthres < 1 {
		// compression disabled
		w.err = WriteVarInt(w.out, VarInt(w.buf.Len()))
		if w.err != nil {
			return w.err
		}
		_, w.err = io.Copy(w.out, &w.buf)
		if flush {
			return w.Flush()
		}
		return w.err
	}

	// > compression enabled
	// packet header changes after set compression packet
	var datalen int = 0
	if w.buf.Len() >= w.compthres {
		// only packets above given threshold are compressed
		// smalled packets have datalen=0 which means that data is sent raw
		datalen = w.buf.Len()
		// fmt.Print("datalen", w.buf.Len())

		if w.zliber == nil {
			w.zliber = zlib.NewWriter(&w.buf)
		} else {
			w.zliber.Reset(&w.buf)
		}

		_, w.err = io.CopyN(w.zliber, &w.buf, int64(datalen))
		if w.err != nil {
			return w.err
		}

		w.err = w.zliber.Close()
		if w.err != nil {
			return w.err
		}
	}

	var datalen_buf [binary.MaxVarintLen64]byte
	datalen_size := binary.PutUvarint(datalen_buf[:], uint64(uint32(int64(datalen))))

	w.err = WriteVarInt(w.out, VarInt(datalen_size+w.buf.Len())) // packet length
	if w.err != nil {
		return w.err
	}

	_, w.err = w.out.Write(datalen_buf[:datalen_size])
	if w.err != nil {
		return w.err
	}

	_, w.err = io.Copy(w.out, &w.buf)
	if flush {
		return w.Flush()
	}
	return w.err
}

func (w *packetWriter) Flush() error {
	if w.err != nil {
		return w.err
	}
	if w.flusher != nil {
		w.err = w.flusher.Flush()
	}
	return w.err
}

func (w *packetWriter) HijackOutput() io.Writer {
	return w.out
}
