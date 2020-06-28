package packets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
)

// > 0x3B RespawnPacketCB

type RespawnPacketCB struct {
	Dimension  int32
	GameMode   uint8
	HashedSeed int64
	LevelType  string `max_length:"16"`
}

func (packet *RespawnPacketCB) PacketID() VarInt {
	return 0x3B
}

func (packet *RespawnPacketCB) Direction() Direction {
	return ClientBound
}

func (packet *RespawnPacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *RespawnPacketCB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}

// > 0x1B KickPacketCB

type KickPacketCB struct {
	Message string `max_length:"256"`
}

func (packet *KickPacketCB) PacketID() VarInt {
	return 0x1B
}

func (packet *KickPacketCB) Direction() Direction {
	return ClientBound
}

func (packet *KickPacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *KickPacketCB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *KickPacketCB) SetText(text *ChatMessage) error {
	b, err := json.Marshal(text)
	if err != nil {
		return err
	}
	packet.Message = string(b)
	return nil
}

// Use only with hard-coded messages -> panics on error!
func NewIngameKick(text *ChatMessage) *KickPacketCB {
	packet := &KickPacketCB{}
	err := packet.SetText(text)
	if err != nil {
		panic(err)
	}
	return packet
}

func NewIngameKickTxt(text string) *KickPacketCB {
	return NewIngameKick(&ChatMessage{Text: text})
}

// > 0x26 JoinGamePacketCB

type JoinGamePacketCB struct {
	PlayerEntity        EntityID `datatype:"int32"`
	GameMode            uint8
	Dimension           int32
	MaxPlayers          uint8
	HashedSeed          int64
	LevelType           string `max_length:"16"`
	ViewDistance        VarInt
	ReducedDebugInfo    bool
	EnableRespawnScreen bool
}

func (packet *JoinGamePacketCB) PacketID() VarInt {
	return 0x26
}

func (packet *JoinGamePacketCB) Direction() Direction {
	return ClientBound
}

func (packet *JoinGamePacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *JoinGamePacketCB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}

// > 0x19 PluginMessagePacketCB

type PluginMessagePacketCB struct {
	Channel string `max_length:"64"`
	Payload []byte
}

func (packet *PluginMessagePacketCB) PacketID() VarInt {
	return 0x19
}

func (packet *PluginMessagePacketCB) Direction() Direction {
	return ClientBound
}

func (packet *PluginMessagePacketCB) Parse(reader io.Reader) (err error) {
	packet.Channel, err = ReadMinecraftString(reader, 128)
	if err != nil {
		return
	}
	packet.Payload, err = ioutil.ReadAll(reader)
	return
}

func (packet *PluginMessagePacketCB) Serialize(writer io.Writer) (err error) {
	WriteMinecraftString(writer, packet.Channel)
	_, err = writer.Write(packet.Payload)
	return
}

func (packet *PluginMessagePacketCB) MakePayloadReader() *bytes.Reader {
	return bytes.NewReader(packet.Payload)
}

// > 0x1F GameStateChangePacketCB

type GameStateChangePacketCB struct {
	Reason uint8
	Value  float32
}

func (packet *GameStateChangePacketCB) PacketID() VarInt {
	return 0x1F
}

func (packet *GameStateChangePacketCB) Direction() Direction {
	return ClientBound
}

func (packet *GameStateChangePacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *GameStateChangePacketCB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}

// > 0x0F ChatMessagePacketCB

type ChatMessagePacketCB struct {
	Message  string `max_length:"32767"`
	Position uint8
}

func (packet *ChatMessagePacketCB) PacketID() VarInt {
	return 0x0F
}

func (packet *ChatMessagePacketCB) Direction() Direction {
	return ClientBound
}

func (packet *ChatMessagePacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *ChatMessagePacketCB) Serialize(writer io.Writer) (err error) {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *ChatMessagePacketCB) SetMsg(msg *ChatMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	packet.Message = string(b)
	return nil
}

func (packet *ChatMessagePacketCB) SetText(text string) error {
	return packet.SetMsg(&ChatMessage{Text: text})
}

// Creates ChatMessagePacket with given text
// Use only with hard-coded messages -> panics on error!
/*func NewChatTextMsgCB(text *ChatMessage) *ChatMessagePacketCB {
	packet := &ChatMessagePacketCB{}
	err := packet.SetMsg(text)
	if err != nil {
		panic(err)
	}
	return packet
}*/

// > 0x11 TabCompletePacketCB

type TabCompleteMatch struct {
	Match   string
	Tooltip string
}

type TabCompletePacketCB struct {
	TransactionId VarInt
	Start         VarInt
	Length        VarInt
	Count         VarInt
	Matches       []TabCompleteMatch `mcignore:"-"`
}

func (packet *TabCompletePacketCB) PacketID() VarInt {
	return 0x11
}

func (packet *TabCompletePacketCB) Direction() Direction {
	return ClientBound
}

func (packet *TabCompletePacketCB) Parse(reader io.Reader) (err error) {
	err = ReadMinecraftStruct(reader, packet)
	if err != nil || packet.Count < 1 {
		return
	}
	if packet.Count > 32767 {
		return fmt.Errorf("invalid match count: %d", packet.Count)
	}

	packet.Matches = make([]TabCompleteMatch, int(packet.Count))
	for i := range packet.Matches {
		packet.Matches[i].Match, err = ReadMinecraftString(reader, 4096)
		if err != nil {
			return
		}

		var has_tooltip bool
		has_tooltip, err = ReadBool(reader)
		if err != nil {
			return
		}
		if has_tooltip {
			packet.Matches[i].Tooltip, err = ReadMinecraftString(reader, 4096)
			if err != nil {
				return
			}
		}
	}
	return
}

func (packet *TabCompletePacketCB) Serialize(writer io.Writer) (err error) {
	packet.Count = VarInt(len(packet.Matches))
	err = WriteMinecraftStruct(writer, packet)
	if err != nil {
		return err
	}

	for _, v := range packet.Matches {
		err = WriteMinecraftString(writer, v.Match)
		if err != nil {
			return
		}

		err = WriteBool(writer, v.Tooltip != "")
		if err != nil {
			return
		}
		if v.Tooltip != "" {
			err = WriteMinecraftString(writer, v.Tooltip)
			if err != nil {
				return
			}
		}
	}
	return
}

// > 0x34 PlayerListItemPacketCB
const (
	ADD_PLAYER VarInt = iota
	UPDATE_GAME_MODE
	UPDATE_LATENCY
	UPDATE_DISPLAY_NAME
	REMOVE_PLAYER
)

type PlayerListItem struct {
	UUID        uuid.UUID      // Players UUID, sent in all actions, including REMOVE
	Name        string         `max_length:"16"` // ADD_PLAYER
	Properties  []AuthProperty // ADD_PLAYER
	GameMode    VarInt         // ADD_PLAYER, UPDATE_GAME_MODE
	Ping        VarInt         // ADD_PLAYER, UPDATE_LATENCY
	DisplayName string         `max_length:"64"` // ADD_PLAYER, UPDATE_DISPLAY_NAME # json string
}

type PlayerListItemPacketCB struct {
	Action VarInt
	Items  []PlayerListItem
}

func (packet *PlayerListItemPacketCB) PacketID() VarInt {
	return 0x34
}

func (packet *PlayerListItemPacketCB) Direction() Direction {
	return ClientBound
}

func (packet *PlayerListItemPacketCB) Parse(reader io.Reader) (err error) {
	packet.Action, err = ReadVarInt(reader)
	length, err := ReadVarInt(reader)
	if err != nil {
		return err
	}
	if length < 0 || length > 16384 {
		return fmt.Errorf("PlayerListItemPacketCB bad item count %d", length)
	}
	packet.Items = make([]PlayerListItem, int(length))
	for i := range packet.Items {
		v := PlayerListItem{}
		_, err = io.ReadFull(reader, []byte(v.UUID[:]))
		if err != nil {
			return
		}
		switch packet.Action {
		case ADD_PLAYER:
			v.Name, _ = ReadMinecraftString(reader, 128)
			pnum, err := ReadVarInt(reader)
			if err != nil {
				return err
			}
			if pnum < 0 || pnum > 16 {
				return fmt.Errorf("PlayerListItemPacketCB bad property count %d", pnum)
			}
			v.Properties = make([]AuthProperty, pnum)
			for _, prop := range v.Properties {
				prop.Name, _ = ReadMinecraftString(reader, 32767)
				prop.Value, _ = ReadMinecraftString(reader, 32767)
				is_signed, err := ReadBool(reader)
				if is_signed {
					prop.Signature, err = ReadMinecraftString(reader, 32767)
				}
				if err != nil {
					return err
				}
				v.Properties[i] = prop
			}
			v.GameMode, _ = ReadVarInt(reader)
			v.Ping, _ = ReadVarInt(reader)
			var has_displayname bool
			has_displayname, err = ReadBool(reader)
			if has_displayname {
				v.DisplayName, err = ReadMinecraftString(reader, 128)
			}
		case UPDATE_GAME_MODE:
			v.GameMode, err = ReadVarInt(reader)
		case UPDATE_LATENCY:
			v.Ping, err = ReadVarInt(reader)
		case UPDATE_DISPLAY_NAME:
			var has_displayname bool
			has_displayname, err = ReadBool(reader)
			if has_displayname {
				v.DisplayName, err = ReadMinecraftString(reader, 128)
			}
		case REMOVE_PLAYER:
			// no data after uuid
		}
		packet.Items[i] = v
		if err != nil {
			return err
		}
	}
	return
}

func (packet *PlayerListItemPacketCB) Serialize(writer io.Writer) (err error) {
	WriteVarInt(writer, packet.Action)
	err = WriteVarInt(writer, VarInt(len(packet.Items)))
	if err != nil {
		return err
	}
	for _, v := range packet.Items {
		_, err = writer.Write([]byte(v.UUID[:]))
		if err != nil {
			return
		}
		switch packet.Action {
		case ADD_PLAYER:
			if len(v.Name) > 16 {
				return fmt.Errorf("PlayerListItemPacketCB Name too long: %q", v.Name)
			}
			WriteMinecraftString(writer, v.Name)
			WriteVarInt(writer, VarInt(len(v.Properties)))
			for _, prop := range v.Properties {
				WriteMinecraftString(writer, prop.Name)
				WriteMinecraftString(writer, prop.Value)
				err = WriteBool(writer, prop.Signature != "")
				if prop.Signature != "" {
					err = WriteMinecraftString(writer, prop.Signature)
				}
				if err != nil {
					return
				}
			}
			WriteVarInt(writer, v.GameMode)
			WriteVarInt(writer, v.Ping)
			if len(v.DisplayName) > 64 {
				return fmt.Errorf("PlayerListItemPacketCB DisplayName too long: %q", v.DisplayName)
			}
			err = WriteBool(writer, v.DisplayName != "")
			if v.DisplayName != "" {
				err = WriteMinecraftString(writer, v.DisplayName)
			}
		case UPDATE_GAME_MODE:
			err = WriteVarInt(writer, v.GameMode)
		case UPDATE_LATENCY:
			err = WriteVarInt(writer, v.Ping)
		case UPDATE_DISPLAY_NAME:
			if len(v.DisplayName) > 64 {
				return fmt.Errorf("PlayerListItemPacketCB Name too long: %q", v.DisplayName)
			}
			err = WriteBool(writer, v.DisplayName != "")
			if v.DisplayName != "" {
				err = WriteMinecraftString(writer, v.DisplayName)
			}
		case REMOVE_PLAYER:
			// no data after uuid
		}
		if err != nil {
			return err
		}
	}
	return
}

// 0x54 Player List Title (Header/Footer)

type PlayerListTitlePacketCB struct {
	Header string `max_length:"1024"`
	Footer string `max_length:"1024"`
}

func (packet *PlayerListTitlePacketCB) PacketID() VarInt {
	return 0x54
}

func (packet *PlayerListTitlePacketCB) Direction() Direction {
	return ClientBound
}

func (packet *PlayerListTitlePacketCB) Parse(reader io.Reader) (err error) {
	err = ReadMinecraftStruct(reader, packet)
	return
}

func (packet *PlayerListTitlePacketCB) Serialize(writer io.Writer) (err error) {
	err = WriteMinecraftStruct(writer, packet)
	return
}

// > 0x4A ScoreboardObjectivePacketCB

type ScoreboardObjectivePacketCB struct {
	Name  string `max_length:"16"`
	Value string `max_length:"16"`
	Mode  byte
}

func (packet *ScoreboardObjectivePacketCB) PacketID() VarInt {
	return 0x4A
}

func (packet *ScoreboardObjectivePacketCB) Direction() Direction {
	return ClientBound
}

func (packet *ScoreboardObjectivePacketCB) Parse(reader io.Reader) (err error) {
	err = ReadMinecraftStruct(reader, packet)
	return
}

func (packet *ScoreboardObjectivePacketCB) Serialize(writer io.Writer) (err error) {
	err = WriteMinecraftStruct(writer, packet)
	return
}

// > 0x4C TeamsPacketCB // TODO aktualizacja 1.13

type TeamsPacketCB struct {
	Name         string `max_length:"16"`
	Mode         byte
	DisplayName  string
	Prefix       string
	Suffix       string
	FriendlyFire int8
	Players      []string
}

func (packet *TeamsPacketCB) PacketID() VarInt {
	return 0x4C
}

func (packet *TeamsPacketCB) Direction() Direction {
	return ClientBound
}

func (packet *TeamsPacketCB) Parse(reader io.Reader) (err error) {
	packet.Name, err = ReadMinecraftString(reader, 64)
	packet.Mode, err = ReadUnsignedByte(reader)

	if (packet.Mode == 0) || (packet.Mode == 2) { // create and update
		packet.DisplayName, err = ReadMinecraftString(reader, 64)
		packet.Prefix, err = ReadMinecraftString(reader, 16)
		packet.Suffix, err = ReadMinecraftString(reader, 16)
		packet.FriendlyFire, err = ReadByte(reader)
	}

	if err != nil {
		return
	}

	if (packet.Mode == 0) || (packet.Mode == 3) || (packet.Mode == 4) { // create, add player, remove player
		var player_count int16
		player_count, err = ReadShort(reader)
		if err != nil {
			return
		}
		if (player_count < 0) || (player_count > 1024) {
			return fmt.Errorf("Unexpected player_count in TeamsPacket: %d", player_count)
		}
		packet.Players = make([]string, player_count)
		for i := 0; i < int(player_count); i++ {
			packet.Players[int(i)], err = ReadMinecraftString(reader, 16)
			if err != nil {
				return
			}
		}
	}

	return
}

func (packet *TeamsPacketCB) Serialize(writer io.Writer) (err error) {
	WriteMinecraftString(writer, packet.Name)
	WriteUnsignedByte(writer, packet.Mode)

	if (packet.Mode == 0) || (packet.Mode == 2) { // create and update
		WriteMinecraftString(writer, packet.DisplayName)
		WriteMinecraftString(writer, packet.Prefix)
		WriteMinecraftString(writer, packet.Suffix)
		err = WriteByte(writer, packet.FriendlyFire)
	}

	if err != nil {
		return
	}

	if (packet.Mode == 0) || (packet.Mode == 3) || (packet.Mode == 4) { // create, add player, remove player
		if len(packet.Players) > 1024 {
			return fmt.Errorf("TeamsPacket.Players too long! %d elements", len(packet.Players))
		}

		WriteShort(writer, int16(len(packet.Players)))
		for _, v := range packet.Players {
			err = WriteMinecraftString(writer, v)
			if err != nil {
				return
			}
		}
	}
	return
}

// > 0x4A TimeUpdatePacketCB

// type TimeUpdatePacketCB struct {
// 	WorldAge int64
// 	Time     int64
// }

// func (packet *TimeUpdatePacketCB) PacketID() VarInt {
// 	return 0x4E
// }

// func (packet *TimeUpdatePacketCB) Direction() Direction {
// 	return ClientBound
// }

// func (packet *TimeUpdatePacketCB) Parse(reader io.Reader) (err error) {
// 	return ReadMinecraftStruct(reader, packet)
// }

// func (packet *TimeUpdatePacketCB) Serialize(writer io.Writer) (err error) {
// 	return WriteMinecraftStruct(writer, packet)
// }

// 0x0E SpawnObjectPacketCB

// type SpawnObjectPacketCB struct {
// 	Entity  EntityID
// 	Type VarInt
// 	X       int32
// 	Y       int32
// 	Z       int32
// 	Pitch   int8
// 	Yaw     int8
// 	Data    EntityID // sometimes
// 	SpeedX  int16
// 	SpeedY  int16
// 	SpeedZ  int16
// }

// func (packet *SpawnObjectPacketCB) PacketID() VarInt {
// 	return 0x0E
// }

// func (packet *SpawnObjectPacketCB) Direction() Direction {
// 	return ClientBound
// }

// func (packet *SpawnObjectPacketCB) Parse(reader io.Reader) (err error) {
// 	entityId, err := ReadVarInt(reader)
// 	if err != nil {
// 		return
// 	}
// 	packet.Entity = EntityID(entityId)
// 	packet.Type, _ = ReadVarInt(reader)
// 	packet.X, _ = ReadInt(reader)
// 	packet.Y, _ = ReadInt(reader)
// 	packet.Z, _ = ReadInt(reader)
// 	packet.Yaw, _ = ReadByte(reader)
// 	packet.Pitch, _ = ReadByte(reader)

// 	data, err := ReadInt(reader)
// 	if err != nil {
// 		return
// 	}
// 	packet.Data = EntityID(data)
// 	if packet.Data > 0 {
// 		packet.SpeedX, _ = ReadShort(reader)
// 		packet.SpeedY, _ = ReadShort(reader)
// 		packet.SpeedZ, err = ReadShort(reader)
// 	}
// 	return
// }

// func (packet *SpawnObjectPacketCB) Serialize(writer io.Writer) (err error) {
// 	WriteVarInt(writer, VarInt(packet.Entity))
// 	WriteByte(writer, packet.ObjType)
// 	WriteInt(writer, packet.X)
// 	WriteInt(writer, packet.Y)
// 	WriteInt(writer, packet.Z)
// 	WriteByte(writer, packet.Yaw)
// 	WriteByte(writer, packet.Pitch)
// 	err = WriteInt(writer, int32(packet.Data))
// 	if packet.Data > 0 {
// 		WriteShort(writer, packet.SpeedX)
// 		WriteShort(writer, packet.SpeedY)
// 		err = WriteShort(writer, packet.SpeedZ)
// 	}
// 	return
// }

// 0x05 Spawn Player

type SpawnPlayer struct {
	EntityID EntityID
	UUID     string
	X        int32
	Y        int32
	Z        int32
	Yaw      int8
	Pitch    int8
}

func (packet *SpawnPlayer) PacketID() VarInt {
	return 0x05
}

func (packet *SpawnPlayer) Direction() Direction {
	return ClientBound
}

func (packet *SpawnPlayer) Parse(reader io.Reader) (err error) {
	entityID, err := ReadVarInt(reader)
	if err != nil {
		return err
	}
	packet.EntityID = EntityID(entityID)
	packet.UUID, err = ReadMinecraftString(reader, 64)
	if err != nil {
		return err
	}
	dataCount, err := ReadVarInt(reader)
	if err != nil {
		return err
	}

	if dataCount > 1024 {
		return fmt.Errorf("SpawnPlayer.Data too long! %d elements", dataCount)
	}

	packet.X, err = ReadInt(reader)
	if err != nil {
		return err
	}
	packet.Y, err = ReadInt(reader)
	if err != nil {
		return err
	}
	packet.Z, err = ReadInt(reader)
	if err != nil {
		return err
	}
	packet.Yaw, err = ReadByte(reader)
	if err != nil {
		return err
	}
	packet.Pitch, err = ReadByte(reader)

	return
}

func (packet *SpawnPlayer) Serialize(writer io.Writer) (err error) {
	err = WriteVarInt(writer, VarInt(packet.EntityID))
	if err != nil {
		return err
	}
	err = WriteMinecraftString(writer, packet.UUID)
	if err != nil {
		return err
	}

	err = WriteInt(writer, packet.X)
	if err != nil {
		return err
	}
	err = WriteInt(writer, packet.Y)
	if err != nil {
		return err
	}
	err = WriteInt(writer, packet.Z)
	if err != nil {
		return err
	}
	err = WriteByte(writer, packet.Yaw)
	if err != nil {
		return err
	}

	return WriteByte(writer, packet.Pitch)
}

// 0x16 Window Property

// type WindowPropertyPacketCB struct {
// 	WindowID uint8
// 	Property int16
// 	Value    int16
// }

// func (packet *WindowPropertyPacketCB) PacketID() VarInt {
// 	return 0x18
// }

// func (packet *WindowPropertyPacketCB) Parse(reader io.Reader) (err error) {
// 	return ReadMinecraftStruct(reader, packet)
// }

// func (packet *WindowPropertyPacketCB) Serialize(writer io.Writer) error {
// 	return WriteMinecraftStruct(writer, packet)
// }

// func (packet *WindowPropertyPacketCB) Direction() Direction {
// 	return ClientBound
// }

// 0x36 Player Position And Look CB

type PlayerPositionAndLookPacketCB struct {
	X          float64
	Y          float64
	Z          float64
	Yaw        float32
	Pitch      float32
	Flags      uint8
	TeleportID VarInt // returned in Teleport Confirm packet
}

func (packet *PlayerPositionAndLookPacketCB) PacketID() VarInt {
	return 0x36
}

func (packet *PlayerPositionAndLookPacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *PlayerPositionAndLookPacketCB) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *PlayerPositionAndLookPacketCB) Direction() Direction {
	return ClientBound
}

// 0x4E Spawn Position
type SpawnPositionPacketCB struct {
	Position Position
}

func (packet *SpawnPositionPacketCB) PacketID() VarInt {
	return 0x4E
}

func (packet *SpawnPositionPacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *SpawnPositionPacketCB) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *SpawnPositionPacketCB) Direction() Direction {
	return ClientBound
}

// 0x3F Camera

type CameraPacketCB struct { // 1.14
	ID EntityID `datatype:"VarInt"`
}

func (packet *CameraPacketCB) PacketID() VarInt {
	return 0x3F
}

func (packet *CameraPacketCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *CameraPacketCB) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *CameraPacketCB) Direction() Direction {
	return ClientBound
}

type ResourcePackSendCB struct {
	Url  string
	Hash string
}

func (packet *ResourcePackSendCB) PacketID() VarInt {
	return 0x3A
}

func (packet *ResourcePackSendCB) Parse(reader io.Reader) (err error) {
	return ReadMinecraftStruct(reader, packet)
}

func (packet *ResourcePackSendCB) Serialize(writer io.Writer) error {
	return WriteMinecraftStruct(writer, packet)
}

func (packet *ResourcePackSendCB) Direction() Direction {
	return ClientBound
}
