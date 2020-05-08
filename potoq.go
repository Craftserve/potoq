package potoq

import (
	"fmt"
	"github.com/Craftserve/potoq/packets"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"sync"
)

var Players PlayerManager

var UpstreamServerMap map[string]*ReconnectCommand
var UpstreamLock sync.Mutex

var PingHandler func(packet *packets.HandshakePacket) packets.ServerStatus
var PreLoginHandler func(handler *Handler) error
var LoginHandler func(handler *Handler, login_err error) error

func Serve(listener *net.TCPListener) {
	err := LoadUpstreams("upstreams.yml")
	if err != nil {
		panic(err)
	}

	if PingHandler == nil || PreLoginHandler == nil || LoginHandler == nil {
		panic("One of required handlers is not set!")
	}

	for {
		socket, err := listener.AcceptTCP()
		if err != nil {
			log.WithError(err).Error("Error while accepting connection")
			continue
		}

		handler := NewHandler(socket)
		go handler.Handle()
	}
}

func LoadUpstreams(fName string) (err error) {
	var data []byte
	data, err = ioutil.ReadFile(fName)
	if err != nil {
		return
	}

	var up = make(map[string]string)
	err = yaml.Unmarshal(data, up)
	if err != nil {
		return
	}

	ups := make(map[string]*ReconnectCommand)

	for name, addr := range up {
		if _, err = net.ResolveTCPAddr("tcp4", addr); err != nil {
			err = fmt.Errorf("RegisterUpstream %q %q tcp error %s", name, addr, err)
			return
		}

		ups[name] = &ReconnectCommand{Name: name, Addr: addr}
	}

	if len(ups) <= 0 {
		err = fmt.Errorf("Upstreams map len <= 0!")
		return
	}

	UpstreamLock.Lock()
	UpstreamServerMap = ups
	UpstreamLock.Unlock()

	return
}
