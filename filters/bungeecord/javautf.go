package bungeecord

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Implementation of https://docs.oracle.com/javase/6/docs/api/java/io/DataInput.html#modified-utf-8

func ReadJavaUTF(reader io.Reader) (s string, err error) {
	var size_prefix uint16
	err = binary.Read(reader, binary.BigEndian, &size_prefix)
	if err != nil {
		return
	}

	buf := make([]byte, size_prefix)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return
	}

	var chars []rune
	for i := 0; i < int(size_prefix); {
		b := buf[i]
		switch {
		case (b & 0x80) == 0x00: // 0xxxxxxx, 1 byte
			chars = append(chars, rune(b))
			i += 1
		case (b & 0xE0) == 0xC0: // 110xxxxx, 2 bytes
			if i+1 >= int(size_prefix) {
				return string(chars), io.ErrUnexpectedEOF
			}
			r := rune(buf[i]&0x1F)<<6 |
				rune(buf[i+1]&0x3F)
			chars = append(chars, r)
			i += 2
		case (b & 0xF0) == 0xE0: // 1110xxxx, 3 bytes
			if i+2 >= int(size_prefix) {
				return string(chars), io.ErrUnexpectedEOF
			}
			r := rune(buf[i]&0x0F)<<12 |
				rune(buf[i+1]&0x3F)<<6 |
				rune(buf[i+2]&0x3F)
			chars = append(chars, r)
			i += 3
		default:
			return "", fmt.Errorf("Invalid first byte of rune at %d", i)
		}
	}

	return string(chars), nil
}

func WriteJavaUTF(writer io.Writer, s string) (err error) {
	var buf = make([]byte, 0, len(s)*2)
	for _, char := range s { // https://golang.org/ref/spec#Operator_precedence
		switch {
		case char <= 0x007F && char != 0x0000: // 1 byte; dont encode nullchar as 1 byte
			buf = append(buf, byte(char))
		case char <= 0x07FF: // 2 bytes, nullchar is encoded here
			c1 := byte(0xc0 | (char>>6)&0x1f)
			c2 := byte(0x80 | char&0x3f)
			buf = append(buf, c1, c2)
		case char <= 0xFFFF: // 3 bytes
			c1 := byte(0xe0 | (char>>12)&0x0f)
			c2 := byte(0x80 | (char>>6)&0x3f)
			c3 := byte(0x80 | 0x3f&char)
			buf = append(buf, c1, c2, c3)
		default:
			return fmt.Errorf("Char outside range: %X", char)
		}
	}
	if len(buf) > 65535 {
		return fmt.Errorf("Encoded string too long!")
	}
	var size_prefix uint16 = uint16(len(buf))
	err = binary.Write(writer, binary.BigEndian, &size_prefix)
	if err != nil {
		return
	}
	_, err = writer.Write(buf)
	return
}
