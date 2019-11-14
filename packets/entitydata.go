package packets

import (
	"fmt"
	"io"
)

type EntityMetadata map[byte]interface{}

func NewEntityMetadata() EntityMetadata {
	return make(EntityMetadata)
}

//TODO new parser and serializer
func (data EntityMetadata) Parse(reader io.Reader) (err error) {
	var item, index, dataType byte
	for {
		item, err = ReadUnsignedByte(reader)
		if err != nil {
			return
		}
		if item == 127 {
			return
		}

		index = item & 0x1F
		dataType = item >> 5

		switch dataType {
		case 0:
			data[index], err = ReadByte(reader) // !!!!!! reads int8 !!!!!!!!!!
		case 1:
			data[index], err = ReadVarInt(reader)
		case 2:
			data[index], err = ReadFloat(reader)
		case 3:
			data[index], err = ReadMinecraftString(reader, 255)
		case 4:
			data[index], err = ReadMinecraftString(reader, 255) //chat
		case 5:
			//data[index], err = chatoptions
		case 6:
			item := new(ItemData)
			err = item.Parse(reader)
			data[index] = item
		case 7:
			data[index], err = ReadBool(reader)
		case 8:
			vector := new(Vector)
			vector.X, _ = ReadInt(reader)
			vector.Y, _ = ReadInt(reader)
			vector.Z, err = ReadInt(reader)
			data[index] = vector
		case 9:
			//Position
		case 10:
			//optional Position
		case 11:
			data[index], err = ReadVarInt(reader)
		case 12:
			//optuuid
		case 13:
			data[index], err = ReadVarInt(reader)
		default:
			panic(fmt.Sprintf("Unkown dataType in EntityMetadata.Parse: %X %X %X", item, index, dataType))
		}
		if err != nil {
			return
		}
	}
	return
}

func (data EntityMetadata) Serialize(writer io.Writer) (err error) {
	for index, value := range data {
		switch value.(type) {
		case int8:
			WriteUnsignedByte(writer, index|(0<<5))
			err = WriteByte(writer, value.(int8))
		case int16:
			WriteUnsignedByte(writer, index|(1<<5))
			err = WriteShort(writer, value.(int16))
		case int32:
			WriteUnsignedByte(writer, index|(2<<5))
			err = WriteInt(writer, value.(int32))
		case float32:
			WriteUnsignedByte(writer, index|(3<<5))
			err = WriteFloat(writer, value.(float32))
		case string:
			WriteUnsignedByte(writer, index|(4<<5))
			err = WriteMinecraftString(writer, value.(string))
		case *ItemData:
			WriteUnsignedByte(writer, index|(5<<5))
			item := value.(*ItemData)
			err = item.Serialize(writer)
		case *Vector:
			WriteUnsignedByte(writer, index|(6<<5))
			vector := value.(*Vector)
			WriteInt(writer, vector.X)
			WriteInt(writer, vector.Y)
			err = WriteInt(writer, vector.Z)
		default:
			panic(fmt.Sprintf("Unkown type in EntityMetadata.Serialize for %d: %T", index, value))
		}
		if err != nil {
			return
		}
	}
	err = WriteUnsignedByte(writer, 0xFF)
	return
}

type Vector struct {
	X int32
	Y int32
	Z int32
}

type AuthProperty struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Signature string `json:"signature"`
}
