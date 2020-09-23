package packets

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
)

// C->S LoginStartPacket

type LoginStartPacket struct {
	Nickname string `max_length:"16"`
}

func (packet *LoginStartPacket) PacketID() VarInt {
	return 0x00
}

func (packet *LoginStartPacket) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *LoginStartPacket) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *LoginStartPacket) Direction() Direction {
	return ServerBound
}

// S->C LoginSuccessPacket

type LoginSuccessPacket struct {
	UID      uuid.UUID
	Username string `max_length:"16"`
}

func (packet *LoginSuccessPacket) PacketID() VarInt {
	return 0x02
}

func (packet *LoginSuccessPacket) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *LoginSuccessPacket) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *LoginSuccessPacket) Direction() Direction {
	return ClientBound
}

// S->C LoginKickPacket

type LoginKickPacket struct {
	Message string `max_length:"256"`
}

func (packet *LoginKickPacket) PacketID() VarInt {
	return 0x00
}

func (packet *LoginKickPacket) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *LoginKickPacket) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *LoginKickPacket) Direction() Direction {
	return ClientBound
}

func (packet *LoginKickPacket) SetText(text *ChatMessage) error {
	b, err := json.Marshal(text)
	if err != nil {
		return err
	}
	packet.Message = string(b)
	return nil
}

func (packet *LoginKickPacket) Error() string {
	return fmt.Sprintf("%#v", packet)
}

// Use only with hard-coded messages -> panics on error!
func NewLoginKick(text *ChatMessage) *LoginKickPacket {
	packet := &LoginKickPacket{}
	err := packet.SetText(text)
	if err != nil {
		panic(err)
	}
	return packet
}

// S->C EncryptionRequestPacket

type EncryptionRequestPacket struct {
	ServerID  string `max_length:"20"`
	PublicKey []byte `max_length:"2048" length_prefix:"VarInt"`
	Token     []byte `max_length:"256" length_prefix:"VarInt"`
}

func (packet *EncryptionRequestPacket) PacketID() VarInt {
	return 0x01
}

func (packet *EncryptionRequestPacket) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *EncryptionRequestPacket) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *EncryptionRequestPacket) Direction() Direction {
	return ClientBound
}

// C->S EncryptionResponsePacket

type EncryptionResponsePacket struct {
	Secret []byte `max_length:"1024" length_prefix:"VarInt"`
	Token  []byte `max_length:"256" length_prefix:"VarInt"`
}

func (packet *EncryptionResponsePacket) PacketID() VarInt {
	return 0x01
}

func (packet *EncryptionResponsePacket) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *EncryptionResponsePacket) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *EncryptionResponsePacket) Direction() Direction {
	return ServerBound
}

// S->C LoginCompressionPacket

type LoginCompressionPacket struct {
	Threshold VarInt
}

func (packet *LoginCompressionPacket) PacketID() VarInt {
	return 0x03
}

func (packet *LoginCompressionPacket) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *LoginCompressionPacket) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *LoginCompressionPacket) Direction() Direction {
	return ClientBound
}
