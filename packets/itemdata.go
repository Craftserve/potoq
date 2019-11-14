package packets

import "io"

type ItemData struct {
	ID     int16
	Count  int8
	Damage int16
	Data   []byte
}

func NewItemData() *ItemData {
	return new(ItemData)
}

func (item *ItemData) Parse(reader io.Reader) (err error) {
	item.ID, err = ReadShort(reader)
	if err == nil && item.ID >= 0 {
		item.Count, _ = ReadByte(reader)
		item.Damage, _ = ReadShort(reader)

		nbtLength, _ := ReadShort(reader)
		if nbtLength >= 0 {
			item.Data = make([]byte, nbtLength)
			_, err = io.ReadFull(reader, item.Data)
		}
	}
	return
}

func (item *ItemData) Serialize(writer io.Writer) (err error) {
	WriteShort(writer, item.ID)
	if item.ID >= 0 {
		WriteByte(writer, item.Count)
		WriteShort(writer, item.Damage)
		if len(item.Data) > 0 {
			WriteShort(writer, int16(len(item.Data)))
			writer.Write(item.Data)
		} else {
			WriteShort(writer, -1)
		}
	}
	return
}
