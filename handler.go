package potoq

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Craftserve/potoq/packets"
	"github.com/Craftserve/potoq/utils"

	"github.com/google/uuid"
	"gopkg.in/tomb.v1"
)

type Permissible interface {
	HasPermission(name string) bool
}

type Handler struct {
	tomb.Tomb

	DownstreamC net.Conn
	DownstreamR packets.PacketReader
	DownstreamW packets.PacketWriter

	downstream_packets     chan packets.Packet
	downstream_tomb        *tomb.Tomb
	DownstreamAddr         string
	DownstreamPacketsCount uint64

	UpstreamW            packets.PacketWriter
	UpstreamC            io.Closer
	UpstreamPackets      chan packets.Packet
	UpstreamTomb         *tomb.Tomb
	UpstreamName         string
	UpstreamPacketsCount uint64

	// used in full connection
	Handshake      packets.HandshakePacket
	Nickname       string
	UUID           uuid.UUID
	Authenticator  *Authenticator
	AuthProperties []packets.AuthProperty
	Permissions    Permissible
	FilterData     sync.Map

	// packet dispatcher and other hooks
	commandChan       HandlerCommandChan
	CloseHooks        []func()

	// startup packets, public because contents may be needed in filters etc
	ClientSettings packets.Packet
	MCBrand        packets.Packet

	// logging
	PacketTrace io.WriteCloser
	t0          time.Time
}

func NewHandler(downsock *net.TCPConn) *Handler {
	h := &Handler{}

	h.DownstreamC = downsock
	h.DownstreamR = packets.NewPacketReader(h.DownstreamC, 0)
	h.DownstreamW = packets.NewPacketWriter(h.DownstreamC, 0)
	h.DownstreamAddr = downsock.RemoteAddr().(*net.TCPAddr).IP.String()

	h.Authenticator = getRandomAuthenticator()

	h.commandChan = make(HandlerCommandChan, 10)

	h.t0 = time.Now()
	return h
}

func (handler *Handler) Handle() {
	_, err := packets.ParsePackets(handler.DownstreamR, &handler.Handshake)

	if err == nil {
		switch packets.ConnState(handler.Handshake.NextState) {
		case packets.STATUS:
			err = handler.handleStatus()
		case packets.LOGIN:
			err = handler.handleProxy()
		default:
			err = fmt.Errorf("Invalid handshake.NextState! %d", handler.Handshake.NextState)
		}
	}

	handler.Tomb.Kill(err)

	if handler.UpstreamC != nil {
		handler.UpstreamC.Close()
		handler.UpstreamC = nil
		handler.UpstreamW = nil
	}
	if handler.DownstreamC != nil {
		handler.DownstreamC.Close()
		handler.DownstreamR = nil
		if c, ok := handler.DownstreamW.(io.Closer); ok {
			c.Close()
		}
		handler.DownstreamW = nil
		handler.DownstreamC = nil
	}

	handler.Tomb.Done()

	err = handler.Tomb.Err()
	if err != nil {
		handler.Log().WithError(err).Error("Handler last error")
	}

	for _, f := range handler.CloseHooks {
		f()
	}
}

func (handler *Handler) handleStatus() error {
	var request packets.StatusRequestPacketSB
	_, err := packets.ParsePackets(handler.DownstreamR, &request)
	if err != nil {
		return err
	}

	status := PingHandler(&handler.Handshake)
	json, err := status.Serialize()
	if err != nil {
		return err
	}
	err = handler.DownstreamW.WritePacket(&packets.StatusResponsePacketCB{json}, true)
	if err != nil {
		return err
	}

	var ping packets.StatusPingPacketSB
	_, err = packets.ParsePackets(handler.DownstreamR, &ping)
	if err != nil {
		return nil
	}

	return handler.DownstreamW.WritePacket(&packets.StatusPingPacketCB{Time: ping.Time}, true)
}

func (handler *Handler) connectUpstream(name string, addr string) (err error) {
	handler.Log().WithFields(logrus.Fields{
		"address": addr,
	}).Info("Connecting to upstream")
	upsock, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	handler.UpstreamW = packets.NewPacketWriter(upsock, 0)
	handler.UpstreamC = upsock
	handler.UpstreamName = name

	handshake := handler.Handshake
	// niestety nie da sie tego zaimplementowac w filtrach, bo nie robimy dispatch w state roznym niz PLAY
	handshake.Host = handshake.Host + "\u0000" + handler.DownstreamAddr + "\u0000" + handler.UUID.String()

	handler.UpstreamW.WritePacket(&handshake, false)
	err = handler.UpstreamW.WritePacket(&packets.LoginStartPacket{handler.Nickname}, true)
	if err != nil {
		return
	}

	upstream_r := packets.NewPacketReader(upsock, 0)
	packet, err := packets.ParsePackets(upstream_r, &packets.LoginCompressionPacket{}, &packets.LoginKickPacket{})
	if err != nil {
		return err
	}
	switch p := packet.(type) {
	case *packets.LoginCompressionPacket:
		if int(p.Threshold) != packets.CompressThreshold {
			return fmt.Errorf("connectUpstream: bad compression threshold %d", p.Threshold)
		}
	case *packets.LoginKickPacket:
		handler.Log().WithFields(logrus.Fields{
			"address": addr,
			"message": p.Message,
		}).Info("Upstream connect kick")
		return fmt.Errorf(p.Message)
	default:
		panic("unexpected packet type")
	}

	upstream_r = packets.NewPacketReader(bufio.NewReaderSize(upsock, packets.MaxPacketSize), packets.CompressThreshold)
	handler.UpstreamW = packets.NewPacketWriter(bufio.NewWriterSize(upsock, packets.MaxPacketSize), packets.CompressThreshold)

	var success packets.LoginSuccessPacket
	_, err = packets.ParsePackets(upstream_r, &success)
	if err != nil {
		return
	}

	handler.Log().WithFields(logrus.Fields{
		"name": name,
		"address": addr,
		"success": success,
	}).Info("Upstream connected")

	handler.UpstreamPackets = make(chan packets.Packet)
	handler.UpstreamTomb = &tomb.Tomb{}
	atomic.StoreUint64(&handler.UpstreamPacketsCount, 0)
	go handler.read_packets(upstream_r, handler.UpstreamPackets, handler.UpstreamTomb, packets.ClientBound)

	return nil
}

func (handler *Handler) read_packets(reader packets.PacketReader, packet_chan chan<- packets.Packet, tmb *tomb.Tomb, direction packets.Direction) {
	var err error
	var raw packets.RawPacket

loop:
	for tmb.Err() == tomb.ErrStillAlive {
		raw, err = reader.ReadPacket()
		if err != nil {
			break
		}

		var packet packets.Packet
		should_parse := packetsParsed.Get(int(direction-1)*256 + int(raw.ID))
		if should_parse {
			packet = packets.NewPacket(raw.ID, packets.PLAY, direction)
			if packet == nil {
				err = fmt.Errorf("Not implemented packet -> %v", raw.ID)
				break
			}

			var payload io.Reader
			payload, err = reader.Payload()
			if err != nil {
				break
			}

			err = packet.Parse(payload)
			if err != nil { // dump packet to file
				var fname string = "WTF"
				f, ferr := ioutil.TempFile("trace/", fmt.Sprintf("%s_%02X_*.packet", handler.Nickname, raw.ID))
				if ferr != nil {
					fname = "FERR: " + ferr.Error()
					handler.Log().WithError(ferr).Error("read_packets: packet dump tempfile error")
				} else {
					fname = f.Name()
					packets.NewPacketWriter(f, 0).WritePacket(raw, true)
					f.Close()
				}
				err = fmt.Errorf("Error parsing packet %02X: %w, dump: %s", raw.ID, err, fname)
				// err = fmt.Errorf("Error parsing packet %02X: %s, dump: %s, n: %d/%d %d", raw.ID, err, fname, payload_lim.N, len(raw.Payload), raw.DataLength)
				break
			}

			packets.BufferPool.Put(raw.Payload)
			raw.Payload = nil
		} else {
			packet = raw
		}

		select {
		case packet_chan <- packet:
		case <-tmb.Dying():
			break loop
		}
	}
	close(packet_chan)
	if err == io.EOF {
		tmb.Kill(err)
	} else {
		tmb.Killf("%s %w", direction, err)
	}
	tmb.Done()
}

func (handler *Handler) handlePacket(packet packets.Packet, direction packets.Direction, writer packets.PacketWriter, flush bool) error {
	var drop bool
	raw, is_raw := packet.(packets.RawPacket)

	if !is_raw {
		// run packet handlers
		err := handler.dispatchPacket(packet)
		switch err {
		case nil:
		case ErrDropPacket:
			drop = true
		default:
			return fmt.Errorf("[%s] dispatch error: %w", direction, err)
		}
	}

	// pass it to other side if needed
	if !drop {
		err := writer.WritePacket(packet, flush)
		if err != nil {
			return fmt.Errorf("[%s] write packet error %w", direction, err)
		}
	} else {
		handler.Log().WithFields(logrus.Fields{
			"direction": direction,
			"packet": packet,
		}).Debug("dropping")
	}
	if is_raw {
		packets.BufferPool.Put(raw.Payload)
	}
	return nil
}

func (handler *Handler) handleProxy() (err error) {
	if handler.Handshake.Protocol != packets.ProtocolVersion {
		kick := packets.NewLoginKick(&packets.ChatMessage{Text: "Server version:" + packets.GameVersion.Name})
		return handler.DownstreamW.WritePacket(kick, true)
	}

	var login_start packets.LoginStartPacket
	_, err = packets.ParsePackets(handler.DownstreamR, &login_start)
	if err != nil {
		return
	}
	if !ValidateNickname(login_start.Nickname) {
		kick := packets.NewLoginKick(&packets.ChatMessage{Text: "Invalid nickname"})
		return handler.DownstreamW.WritePacket(kick, true)
	}
	handler.Nickname = login_start.Nickname

	compression := &packets.LoginCompressionPacket{packets.VarInt(packets.CompressThreshold)}
	err = handler.DownstreamW.WritePacket(compression, true)
	if err != nil {
		return
	}
	handler.DownstreamW = packets.NewPacketWriter(handler.DownstreamC, packets.CompressThreshold)
	handler.DownstreamR = packets.NewPacketReader(handler.DownstreamC, packets.CompressThreshold)

	// key exchange is quite costly so we look for upstream here, kick non-whitelisted users, set capabilities etc
	err = PreLoginHandler(handler)
	if err != nil {
		var kick *packets.LoginKickPacket
		if kick, _ = err.(*packets.LoginKickPacket); kick == nil {
			kick = packets.NewLoginKick(&packets.ChatMessage{Text: err.Error()})
		}
		return handler.DownstreamW.WritePacket(kick, true)
	}

	if handler.Authenticator == nil {
		handler.Log().Debug("Non-premium login")
		handler.UUID = OfflinePlayerUUID(handler.Nickname)
		err = ErrUnauthenticated
	} else {
		handler.Log().Debug("Checking minecraft account")
		err = handler.establishEncryptionAsServer()
	}

	err = LoginHandler(handler, err)
	if err != nil {
		var kick *packets.LoginKickPacket
		if kick, _ = err.(*packets.LoginKickPacket); kick == nil {
			kick = packets.NewLoginKick(&packets.ChatMessage{Text: err.Error()})
		}
		return handler.DownstreamW.WritePacket(kick, true)
	}

	err = handler.DownstreamW.WritePacket(&packets.LoginSuccessPacket{
		UID:      handler.UUID.String(),
		Username: handler.Nickname,
	}, true)
	if err != nil {
		return
	}

	// STATE IS PLAY FROM NOW

	UpstreamLock.Lock()
	target, ok := UpstreamServerMap[handler.UpstreamName]
	UpstreamLock.Unlock()
	if !ok {
		kick := packets.NewIngameKickTxt("Login error: unknown target " + handler.UpstreamName)
		return handler.DownstreamW.WritePacket(kick, true)
	}

	err = handler.connectUpstream(target.Name, target.Addr)
	if err != nil {
		kick := packets.NewIngameKickTxt("Login error: upstream error: " + err.Error())
		return handler.DownstreamW.WritePacket(kick, true)
	}

	registered := Players.Register(handler)
	if !registered {
		kick := packets.NewIngameKickTxt("Login error: register error")
		return handler.DownstreamW.WritePacket(kick, true)
	}
	defer Players.Unregister(handler)

	handler.downstream_packets = make(chan packets.Packet)
	handler.downstream_tomb = &tomb.Tomb{}
	go handler.read_packets(handler.DownstreamR, handler.downstream_packets, handler.downstream_tomb, packets.ServerBound)

	var last_packet_ts int64 = time.Now().Unix()
	idle_timeout := make(chan struct{})

	go func() {
		const max_idle = 30 * time.Second
		for {
			time.Sleep(max_idle / 10)
			if time.Now().Unix()-atomic.LoadInt64(&last_packet_ts) > int64(max_idle/time.Second) {
				close(idle_timeout)
				break
			}
		}
	}()

	var upflush, downflush = time.Now(), time.Now()

MainLoop:
	for {
		atomic.StoreInt64(&last_packet_ts, time.Now().Unix())
		select {
		case command := <-handler.commandChan:
			t1 := time.Now()
			err = command.Execute(handler)
			handler.Log().WithFields(logrus.Fields{
				"command": command,
				"time": time.Since(t1),
			}).WithError(err).Debug("Executed command")
		case packet, ok := <-handler.downstream_packets:
			if ok {
				if handler.PacketTrace != nil {
					err = handler.packetTrace(packet, packets.ServerBound)
					if err != nil {
						break MainLoop
					}
				}
				atomic.AddUint64(&handler.DownstreamPacketsCount, 1)
				// TODO: zamienic ten flushing na atomic licznik ile pakietow jest juz zparsowanych
				flush := len(handler.downstream_packets) == 0 || time.Since(downflush) > 50*time.Millisecond
				if flush {
					downflush = time.Now()
				}
				err = handler.handlePacket(packet, packets.ServerBound, handler.UpstreamW, flush)
			} else {
				handler.downstream_tomb.Wait()
				err = handler.downstream_tomb.Err()
			}
		case packet, ok := <-handler.UpstreamPackets:
			if ok {
				if handler.PacketTrace != nil {
					err = handler.packetTrace(packet, packets.ClientBound)
					if err != nil {
						break MainLoop
					}
				}
				atomic.AddUint64(&handler.UpstreamPacketsCount, 1)
				flush := len(handler.UpstreamPackets) == 0 || time.Since(upflush) > 50*time.Millisecond
				if flush {
					upflush = time.Now()
				}
				err = handler.handlePacket(packet, packets.ClientBound, handler.DownstreamW, flush)
			} else {
				handler.UpstreamTomb.Wait()
				err = handler.UpstreamTomb.Err()
			}
		case <-idle_timeout:
			handler.Log().Error("Idle timeout in handleProxy MainLoop")
			err = io.EOF
		}
		if err != nil {
			break MainLoop
		}
	}

	if err != nil && err != io.EOF {
		err = fmt.Errorf("Error in handler's MainLoop: %w", err)
		if handler.PacketTrace != nil {
			err_pt := handler.PacketTrace.Close()
			if err_pt != nil {
				handler.Log().WithError(err).Error("PacketTrace close error")
			}
		}
	}

	// end both read_packets goroutines
	handler.UpstreamTomb.Kill(nil)
	handler.downstream_tomb.Kill(nil)

	handler.Log().Info("Closing play handler")
	return err
}

func (handler *Handler) dispatchPacket(packet packets.Packet) error {
	var bucket = packetFilters[int(packet.Direction()-1)*256+int(packet.PacketID())]
	for _, proc := range bucket {
		err := proc(handler, packet)
		if err != nil {
			return err
		}
	}
	return nil
}

func (handler *Handler) HasPermission(perm string) bool {
	if handler.Permissions == nil {
		return false
	}
	return handler.Permissions.HasPermission(perm)
}

func (handler *Handler) String() string {
	var uuid_str string = "nil"
	if handler.UUID != uuid.Nil {
		uuid_str = handler.UUID.String()
	}
	return fmt.Sprintf("Handler{%s, %s, %s, %s}", handler.Nickname, uuid_str, handler.DownstreamAddr, handler.UpstreamName)
}

func (handler *Handler) Log() logrus.FieldLogger {
	return logrus.WithFields(logrus.Fields{
		"uuid": handler.UUID,
		"nickname": handler.Nickname,
		"upstream": handler.UpstreamName,
		"downstreamAddr": handler.DownstreamAddr,
	})
}

func (handler *Handler) PushCommand(cmd HandlerCommand, block bool) bool {
	if handler.commandChan == nil {
		return false
	}
	if block {
		handler.commandChan <- cmd
		return true
	}
	select {
	case handler.commandChan <- cmd:
		return true
	default:
		return false
	}
}

// Thread-safely send §-formatted chat message
func (handler *Handler) SendChatMessage(msg string) { // TODO: zamienic to na jakies chat.Sendf(handler, msg, args...)
	data, err := utils.Para2Json(msg)
	if err != nil {
		data, err = utils.Para2Json("CHAT FORMATTING ERROR: " + err.Error())
		if err != nil {
			panic(err)
		}
	}
	handler.InjectPackets(packets.ClientBound, nil, &packets.ChatMessagePacketCB{
		Message:  string(data),
		Position: 1, // system chat message
	})
}

// Thread-safely inject packets into given direction. Raises error in main handler goroutine if raise != nil. Useful for kicking player etc.
// Error is returned if packets cannot be queued. Data sending is asynchronous, connection error are not returned here.
func (handler *Handler) InjectPackets(direction packets.Direction, raise error, payload ...packets.Packet) error {
	cmd := &injectPacketCommand{
		Direction: direction,
		Payload:   payload,
		RaiseErr:  raise,
	}
	select {
	case handler.commandChan <- cmd:
		return nil
	default:
		return fmt.Errorf("commandChan is full")
	}
}

func (handler *Handler) packetTrace(packet packets.Packet, direction packets.Direction) error {
	_, err := fmt.Fprintf(handler.PacketTrace, "%4.4f %s\n", time.Since(handler.t0).Seconds(), packets.ToString(packet, direction))
	return err
}
