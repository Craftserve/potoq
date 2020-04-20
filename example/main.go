package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"

	l4g "github.com/alecthomas/log4go"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

const BIND_ADDR = "0.0.0.0:25565"

func main() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	potoq.PreLoginHandler = PreLoginHandler
	potoq.LoginHandler = LoginHandler
	potoq.PingHandler = PingHandler

	potoq.RegisterPacketFilter(&packets.ChatMessagePacketSB{}, SimpleChatFilter)

	listener, err := net.Listen("tcp", BIND_ADDR)
	if err != nil {
		l4g.Critical("Error listening: %s", err.Error())
		return
	}
	potoq.Serve(listener.(*net.TCPListener))
}

func PingHandler(packet *packets.HandshakePacket) packets.ServerStatus {
	p := potoq.Players.Len()
	return packets.ServerStatus{
		Version:     packets.GameVersion,
		Players:     packets.ServerStatusPlayers{p, p + 1},
		Description: "potoq",
	}
}

func PreLoginHandler(handler *potoq.Handler) error {
	handler.Authenticator = nil // enable offline-mode login
	return nil
}

func LoginHandler(handler *potoq.Handler, login_err error) (err error) {
	if login_err != nil && login_err != potoq.ErrUnauthenticated { // login_err is ErrUnauthenticated for offline-mode players
		return login_err
	}

	if potoq.Players.GetByUUID(handler.UUID) != nil {
		return fmt.Errorf("Gracz jest juz zalogowany.")
	}

	handler.UpstreamName = "local"

	return nil
}

func SimpleChatFilter(handler *potoq.Handler, packet packets.Packet) error {
	p := packet.(*packets.ChatMessagePacketSB)
	p.Message = strings.ReplaceAll(p.Message, "premium", "cukier")

	if strings.Contains(p.Message, "lagi") {
		return potoq.ErrDropPacket
	}

	return nil
}
