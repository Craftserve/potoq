package potoq

import (
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Craftserve/potoq/packets"
)

type HandlerCommand interface {
	Execute(handler *Handler) error
}

type HandlerCommandChan chan HandlerCommand

// >> ReconnectCommand -> causes handler to reconnect to another upstream and fake client world change

type ReconnectCommand struct {
	Name string
	Addr string
}

func (cmd *ReconnectCommand) Execute(handler *Handler) (err error) {
	if handler.UpstreamC != nil {
		handler.UpstreamTomb.Kill(nil)
		err = handler.UpstreamC.Close()
		if err != nil {
			return
		}
		handler.UpstreamC = nil
		handler.UpstreamW = nil
	}

	// handler.upstream_tomb.Wait() // wait for read_packets end
	// if handler.ClientSettings == nil { // ???
	// return fmt.Errorf("Reconnect error: ClientSettings is nil! %#v", handler.ClientSettings)
	// }

	err = handler.connectUpstream(cmd.Name, cmd.Addr)
	if err != nil { // TODO: brzydkie bledy beda jak sektor pelny chyba
		handler.Log().
			WithField("name", cmd.Name).
			WithField("address", cmd.Addr).
			WithError(err).
			Errorln("Connect to upstream failed")
		_ = handler.DownstreamW.WritePacket(packets.NewIngameKickTxt(err.Error()), true)
		return
	}

	// hijack Join Game packet
	var p packets.Packet
	select {
	case <-handler.UpstreamTomb.Dead():
		return handler.UpstreamTomb.Err()
	case p = <-handler.UpstreamPackets:
	}

	join, ok := p.(*packets.JoinGamePacketCB)
	if !ok {
		err = fmt.Errorf("Packet other than JoinGamePacketCB while reconnecting upstream: %#v", p)
		return
	}
	handler.Log().WithFields(logrus.Fields{
		"join": join,
	}).Debug("Reconnect Join")

	// used for scoreboard
	if derr := handler.dispatchPacket(join); derr != nil {
		return fmt.Errorf("Error in JoinGamePacketCB packet hook: %s", err)
	}

	// packets.WritePacket(handler.Upstream, handler.ClientSettings, true, handler.compress_threshold)
	// if handler.MCBrand != nil { // mc brand is not always catched
	// 	packets.WritePacket(handler.Upstream, handler.MCBrand, false, handler.compress_threshold)
	// } else {
	// 	handler.Log.Warn("minecraft:brand is nil for %s", handler)
	// }

	return SendDimensionSwitch(handler.DownstreamW, join)

	// drop all waiting packets from client until
	// loop:
	// 	for {
	// 		p = <-handler.downstream_packets
	// 		if p == nil { // client conn closed
	// 			handler.upstream_tomb.Wait()
	// 			return handler.upstream_tomb.Err()
	// 		}
	// 		if handler.PacketTrace != nil {
	// 			_, err = io.WriteString(handler.PacketTrace, "reconnect: "+packets.ToString(p, packets.ServerBound)+"\n")
	// 			if err != nil {
	// 				return
	// 			}
	// 		}
	// 		// TODO: przerobic sprawdzenie ponizej na (&packets.PlayerPositionLookPacket{}).PacketId() - trzeba zaimplementowac tego structa
	// 		switch p.PacketID() {
	// 		case 0x10, 0x11, 0x04: // 0x10 Player Position, 0x11 Player Position And Look, 0x04 Client Settings //, 0x12, 0x1A:
	// 			packets.WritePacket(handler.Upstream, p, true, handler.compress_threshold)
	// 			handler.Log.Info("breaking reconnect")
	// 			break loop
	// 		}
	// 	}
	// return
}

func SendDimensionSwitch(w packets.PacketWriter, join *packets.JoinGamePacketCB) (err error) {
	// TODO: bungee tutaj jeszcze czysci scoreboard, bossbary i chyba tabliste aktualizuje, ale mozliwe ze to jest zwiazane z jego wlasnym api a nie spigot

	// send world change to client
	temp_dim := &packets.RespawnPacketCB{
		Dimension: 1,
		GameMode:  join.GameMode,
		LevelType: join.LevelType,
	}
	if join.Dimension == 1 {
		temp_dim.Dimension = -1
	}
	w.WritePacket(temp_dim, false)
	w.WritePacket(join, false)

	w.WritePacket(&packets.RespawnPacketCB{
		Dimension: join.Dimension,
		HashedSeed: join.HashedSeed,
		GameMode:  join.GameMode,
		LevelType: join.LevelType,
	}, false)

	// send gamemode change ; pokazuje 'your gamemode has been changed' na chacie, ale inaczej cos sie pierdoli z nieznanego powodu i jest wieczny survival; TODO: sprawdzic to
	w.WritePacket(&packets.GameStateChangePacketCB{
		Reason: 3,
		Value:  float32(join.GameMode),
	}, false)

	return w.Flush()
}

// InjectPacketCommand -> injects packet to selected side of connection

type injectPacketCommand struct {
	Direction packets.Direction
	Payload   []packets.Packet
	RaiseErr  error
}

func (cmd *injectPacketCommand) Execute(handler *Handler) (err error) {
	var w packets.PacketWriter
	switch cmd.Direction {
	case packets.ServerBound:
		w = handler.UpstreamW
	case packets.ClientBound:
		w = handler.DownstreamW
	default:
		panic(fmt.Sprintf("Unknown direction: %X", cmd.Direction))
	}
	if cmd.Payload == nil {
		panic("Nil payload!")
	}
	for _, packet := range cmd.Payload {
		if handler.PacketTrace != nil {
			_, err = io.WriteString(handler.PacketTrace, "inject: "+packets.ToString(packet, cmd.Direction)+"\n")
			if err != nil {
				return
			}
		}
		w.WritePacket(packet, false)
		if err != nil {
			return err
		}
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	if cmd.RaiseErr == io.EOF {
		// wait for packet to be received by client... a little bit ugly.
		time.Sleep(time.Second)
	}
	return cmd.RaiseErr
}
