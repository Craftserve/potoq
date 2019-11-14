package potoq

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"

	l4g "github.com/alecthomas/log4go"
	"gopkg.in/yaml.v2"

	"github.com/Craftserve/potoq/packets"
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
	l4g.LogBufferLength = 4096
	logw := l4g.NewFormatLogWriter(os.Stdout, l4g.FORMAT_SHORT)
	l4g.Global = make(l4g.Logger)
	l4g.Global.AddFilter("stdout", l4g.DEBUG, logw)

	if PingHandler == nil || PreLoginHandler == nil || LoginHandler == nil {
		panic("One of required handlers is not set!")
	}

	for {
		socket, err := listener.AcceptTCP()
		if err != nil {
			l4g.Error("Error while accepting: %s", err)
			continue
		}

		handler := NewHandler(socket)
		handler.Log = l4g.Global
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
