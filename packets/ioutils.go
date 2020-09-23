package packets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"

	"github.com/google/uuid"
)

func ReadBool(reader io.Reader) (c bool, err error) {
	var v [1]byte
	_, err = reader.Read(v[0:])
	c = v[0] == 1
	return
}

func ReadByte(reader io.Reader) (c int8, err error) {
	err = binary.Read(reader, binary.BigEndian, &c)
	return
}

func ReadUnsignedByte(reader io.Reader) (c byte, err error) {
	err = binary.Read(reader, binary.BigEndian, &c)
	return
}

func ReadShort(reader io.Reader) (c int16, err error) {
	err = binary.Read(reader, binary.BigEndian, &c)
	return
}

func ReadInt(reader io.Reader) (c int32, err error) {
	err = binary.Read(reader, binary.BigEndian, &c)
	return
}

func ReadLong(reader io.Reader) (c int64, err error) {
	err = binary.Read(reader, binary.BigEndian, &c)
	return
}

func ReadVarInt(reader io.Reader) (c VarInt, err error) { // TODO: wymagac ByteReader
	br, ok := reader.(io.ByteReader)
	if !ok {
		br = &dummyByteReader{reader, [1]byte{}}
	}
	x, err := binary.ReadUvarint(br)
	return VarInt(int32(uint32(x))), err
}

func ReadFloat(reader io.Reader) (c float32, err error) {
	err = binary.Read(reader, binary.BigEndian, &c)
	return
}

func ReadDouble(reader io.Reader) (c float64, err error) {
	err = binary.Read(reader, binary.BigEndian, &c)
	return
}

func ReadMinecraftString(reader io.Reader, maxLength int) (string, error) {
	length, err := ReadVarInt(reader)
	if err != nil {
		return "", err
	}
	if int(length) > maxLength {
		return "", fmt.Errorf("string longer than maxLength: %d > %d", length, maxLength)
	}
	if length < 0 {
		return "", fmt.Errorf("string length smaller than 0: %d", length)
	}

	d := make([]byte, length)
	if _, err := io.ReadFull(reader, d); err != nil {
		return "", err
	}
	return string(d), nil
}

func ReadIdentifier(reader io.Reader) (Identifier, error) {
	val, err := ReadMinecraftString(reader, IdentifierMaxLength)
	return Identifier(val), err
}

func ReadIdentifierArray(reader io.Reader) ([]Identifier, error) {
	count, err := ReadVarInt(reader)
	if err != nil {
		return nil, err
	}
	array := make([]Identifier, count)
	for i := 0; i < int(count); i++ {
		value, err := ReadIdentifier(reader)
		if err != nil {
			return array, err
		}
		array = append(array, value)
	}
	return array, err
}

func WriteBool(writer io.Writer, c bool) error {
	var v [1]byte
	if c {
		v[0] = 1
	}
	_, err := writer.Write(v[0:])
	return err
}

func WriteByte(writer io.Writer, c int8) error {
	return binary.Write(writer, binary.BigEndian, &c)
}

func WriteUnsignedByte(writer io.Writer, c byte) error {
	return binary.Write(writer, binary.BigEndian, &c)
}

func WriteShort(writer io.Writer, c int16) error {
	return binary.Write(writer, binary.BigEndian, &c)
}

func WriteInt(writer io.Writer, c int32) error {
	return binary.Write(writer, binary.BigEndian, &c)
}

func WriteLong(writer io.Writer, c int64) error {
	return binary.Write(writer, binary.BigEndian, &c)
}

func WriteVarInt(writer io.Writer, c VarInt) error {
	var buf [8]byte
	n := binary.PutUvarint(buf[:], uint64(uint32(c)))
	_, err := writer.Write(buf[:n])
	return err
}

func WriteFloat(writer io.Writer, c float32) error {
	return binary.Write(writer, binary.BigEndian, &c)
}

func WriteDouble(writer io.Writer, c float64) error {
	return binary.Write(writer, binary.BigEndian, &c)
}

func WriteMinecraftString(writer io.Writer, s string) (err error) {
	err = WriteVarInt(writer, VarInt(len(s)))
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(s))
	return
}

func WriteIdentifier(writer io.Writer, identifier Identifier) (err error) {
	err = WriteVarInt(writer, VarInt(len(identifier)))
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(identifier))
	return
}

func WriteIdentifierArray(writer io.Writer, identifiers []Identifier) (err error) {
	err = WriteVarInt(writer, VarInt(len(identifiers)))
	if err != nil {
		return err
	}
	for _, identifier := range identifiers {
		err = WriteIdentifier(writer, identifier)
		if err != nil {
			return err
		}
	}
	return
}

// dummyByteReader - used in ReadVarInt

type dummyByteReader struct {
	io.Reader
	buf [1]byte
}

func (b dummyByteReader) ReadByte() (byte, error) {
	_, err := b.Read(b.buf[:])
	return b.buf[0], err
}

// ReadMinecraftStruct, WriteMinecraftStruct

func ReadMinecraftStruct(reader io.Reader, data interface{}) (err error) {
	br, ok := reader.(io.ByteReader)
	if !ok {
		br = &dummyByteReader{reader, [1]byte{}}
	}

	elem := reflect.ValueOf(data).Elem()
	elemType := reflect.TypeOf(data).Elem()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		fieldType := elemType.Field(i)

		if fieldType.Tag.Get("mcignore") != "" {
			continue
		}

		switch field.Interface().(type) {
		case bool:
			var val bool
			if val, err = ReadBool(reader); err != nil {
				return
			}
			field.SetBool(val)
		case uint8:
			var buf [1]byte
			if _, err = io.ReadFull(reader, buf[0:]); err != nil {
				return
			}
			field.SetUint(uint64(buf[0]))
		case int8:
			var buf [1]byte
			if _, err = io.ReadFull(reader, buf[0:]); err != nil {
				return
			}
			field.SetInt(int64(buf[0]))
		case int16:
			var buf [2]byte
			if _, err = io.ReadFull(reader, buf[0:]); err != nil {
				return
			}
			field.SetInt(int64(binary.BigEndian.Uint16(buf[0:])))
		case int32:
			var buf [4]byte
			if _, err = io.ReadFull(reader, buf[0:]); err != nil {
				return
			}
			field.SetInt(int64(binary.BigEndian.Uint32(buf[0:])))
		case int64:
			var buf [8]byte
			if _, err = io.ReadFull(reader, buf[0:]); err != nil {
				return
			}
			field.SetInt(int64(binary.BigEndian.Uint64(buf[0:])))
		case VarInt:
			var v uint64
			v, err = binary.ReadUvarint(br)
			if err != nil {
				return
			}
			field.SetInt(int64(int32(uint32(v))))
		case float32:
			var buf [4]byte
			if _, err = io.ReadFull(reader, buf[0:]); err != nil {
				return
			}
			field.SetFloat(float64(math.Float32frombits(binary.BigEndian.Uint32(buf[0:]))))
		case float64:
			var buf [8]byte
			if _, err = io.ReadFull(reader, buf[0:]); err != nil {
				return
			}
			field.SetFloat(math.Float64frombits(binary.BigEndian.Uint64(buf[0:])))
		case Identifier:
			var str string
			if str, err = ReadMinecraftString(reader, IdentifierMaxLength); err != nil {
				return
			}
			field.SetString(str)
		case string:
			var maxlen int
			if maxlen, err = strconv.Atoi(fieldType.Tag.Get("max_length")); err != nil {
				panic("Invalid max_length tag in " + fieldType.Name + ": " + err.Error())
			}
			var str string
			if str, err = ReadMinecraftString(reader, maxlen); err != nil {
				return
			}
			field.SetString(str)
		case []byte:
			var maxlen, datalen int64
			if maxlen, err = strconv.ParseInt(fieldType.Tag.Get("max_length"), 10, 64); err != nil {
				panic("invalid max_length tag in " + elemType.Name())
			}

			prefix := fieldType.Tag.Get("length_prefix")
			switch prefix {
			case "int16":
				var buf [2]byte
				if _, err = io.ReadFull(reader, buf[0:]); err != nil {
					return
				}
				datalen = int64(binary.BigEndian.Uint16(buf[0:]))
			case "int32":
				var buf [4]byte
				if _, err = io.ReadFull(reader, buf[0:]); err != nil {
					return
				}
				datalen = int64(binary.BigEndian.Uint32(buf[0:]))
			case "VarInt":
				var v uint64
				if v, err = binary.ReadUvarint(br); err != nil {
					return
				}
				datalen = int64(uint32(v))
			case "eof":
				datalen = maxlen
				if elem.NumField() > i+1 {
					panic("max_length eof field is not last in struct")
				}
			default:
				panic("invalid length_prefix tag in " + elemType.Name())
			}

			if datalen < 0 || datalen > maxlen {
				return fmt.Errorf("invalid datalen value: 0 < %d < %d", datalen, maxlen)
			}

			var buf bytes.Buffer
			if prefix != "eof" {
				buf.Grow(int(datalen))
			}
			_, err = io.CopyN(&buf, reader, datalen) // CopyN uses io.ReaderFrom interface
			if err == io.EOF && prefix == "eof" {
				err = nil
			}
			if err != nil {
				return
			}
			field.SetBytes(buf.Bytes())
		case EntityID:
			var buf [8]byte
			switch fieldType.Tag.Get("datatype") {
			case "int32":
				_, err = io.ReadFull(reader, buf[0:4])
				field.SetInt(int64(binary.BigEndian.Uint32(buf[0:4])))
			case "VarInt":
				var v uint64
				v, err = binary.ReadUvarint(br)
				if err != nil {
					return
				}
				field.SetInt(int64(int32(uint32(v))))
			default:
				panic("Invalid EntityID datatype in " + elemType.Name())
			}
			if err != nil {
				return
			}
		case uuid.UUID:
			var buf uuid.UUID
			_, err = io.ReadFull(reader, buf[:])
			if err != nil {
				return
			}
			field.Set(reflect.ValueOf(buf))
		default:
			panic(fmt.Sprintf("Invalid field %d in minecraft struct %s", i, elemType.Name()))
		}
	}
	return
}

func WriteMinecraftStruct(writer io.Writer, data interface{}) (err error) {
	elem := reflect.ValueOf(data).Elem()
	elemType := reflect.TypeOf(data).Elem()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i).Interface()
		fieldType := elemType.Field(i)

		if fieldType.Tag.Get("mcignore") != "" {
			continue
		}

		switch field.(type) { // TODO: tutaj powinno byc przypisanie, to bysmy nie rzutowali w kazdym case
		case bool:
			if err = WriteBool(writer, field.(bool)); err != nil {
				return
			}
		case uint8:
			buf := []byte{field.(uint8)}
			if _, err = writer.Write(buf); err != nil {
				return
			}
		case int8:
			buf := []uint8{byte(field.(int8))}
			if _, err = writer.Write(buf); err != nil {
				return
			}
		case int16:
			var buf [2]byte
			binary.BigEndian.PutUint16(buf[0:], uint16(field.(int16)))
			if _, err = writer.Write(buf[0:]); err != nil {
				return
			}
		case int32:
			var buf [4]byte
			binary.BigEndian.PutUint32(buf[0:], uint32(field.(int32)))
			if _, err = writer.Write(buf[0:]); err != nil {
				return
			}
		case int64:
			var buf [8]byte
			binary.BigEndian.PutUint64(buf[0:], uint64(field.(int64)))
			if _, err = writer.Write(buf[0:]); err != nil {
				return
			}
		case VarInt:
			var buf [8]byte
			n := binary.PutUvarint(buf[:], uint64(uint32(field.(VarInt))))
			if _, err = writer.Write(buf[:n]); err != nil {
				return
			}
		case float32:
			var buf [4]byte
			binary.BigEndian.PutUint32(buf[0:], math.Float32bits(field.(float32)))
			if _, err = writer.Write(buf[0:]); err != nil {
				return
			}
		case float64:
			var buf [8]byte
			binary.BigEndian.PutUint64(buf[0:], math.Float64bits(field.(float64)))
			if _, err = writer.Write(buf[0:]); err != nil {
				return
			}
		case Identifier:
			if err = WriteIdentifier(writer, field.(Identifier)); err != nil {
				return
			}
		case string:
			if err = WriteMinecraftString(writer, field.(string)); err != nil {
				return
			}
		case []byte:
			bytesArr := field.([]byte)
			bytesLen := len(bytesArr)
			prefix := fieldType.Tag.Get("length_prefix")
			switch prefix {
			case "int16":
				var buf [2]byte
				binary.BigEndian.PutUint16(buf[0:], uint16(bytesLen))
				if _, err = writer.Write(buf[0:]); err != nil {
					return
				}
			case "int32":
				var buf [4]byte
				binary.BigEndian.PutUint32(buf[0:], uint32(bytesLen))
				if _, err = writer.Write(buf[0:]); err != nil {
					return
				}
			case "VarInt":
				var buf [8]byte
				n := binary.PutUvarint(buf[:], uint64(uint32(bytesLen)))
				if _, err = writer.Write(buf[:n]); err != nil {
					return
				}
			case "eof":
				if elem.NumField() > i+1 {
					panic("max_length eof field is not last in struct")
				}
			default:
				panic("Invalid length_prefix tag in " + elemType.Name())
			}

			if _, err = writer.Write(bytesArr); err != nil {
				return
			}
		case EntityID:
			var buf [8]byte
			switch fieldType.Tag.Get("datatype") {
			case "int32":
				binary.BigEndian.PutUint32(buf[:4], uint32(int32(field.(EntityID))))
				_, err = writer.Write(buf[:4])
			case "VarInt":
				n := binary.PutUvarint(buf[:], uint64(uint32(int32(field.(EntityID)))))
				_, err = writer.Write(buf[:n])
			default:
				panic("Invalid EntityID datatype in " + elemType.Name())
			}
			if err != nil {
				return
			}
		case uuid.UUID:
			value := field.(uuid.UUID)
			_, err = writer.Write(value[:])
			if err != nil {
				return
			}
		default:
			panic("Invalid field in minecraft struct - " + fieldType.Name)
		}
	}
	return
}
