package cloudychat

import (
	"fmt"
	"golang.org/x/time/rate"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
	"github.com/Craftserve/potoq/utils"

	l4g "github.com/alecthomas/log4go"
)

var Formatter func(recipient, sender *potoq.Handler, msg string) string = DefaultFormatter
var MessageHook func(handler *potoq.Handler, kind, message string)
var globalChatConfig *chatConfig
var globalChatLock sync.Mutex
var globalChatLimiter *rate.Limiter
var log l4g.Logger
var supervisor_log l4g.Logger
var redis radix.Client
var GameVersion = packets.ServerStatusVersion{"1.15.2", 578}

func RegisterFilters(a radix.Client) {
	redis = a

	potoq.RegisterPacketFilter(&packets.ChatMessagePacketSB{}, ChatFilter)
	potoq.PingHandler = PingHandler

	var err error
	globalChatConfig, err = LoadConfig("chat.yml")
	if err != nil {
		panic("Unable to load chat.yml: " + err.Error())
	}

	log = make(l4g.Logger) // l4g.NewDefaultLogger(l4g.DEBUG)
	writer := l4g.NewFileLogWriter("chat.log", false)
	writer.SetFormat("[%D %T] [%L] %M")
	log.AddFilter("file", l4g.DEBUG, writer)

	supervisor_log = make(l4g.Logger)
	_ = os.Mkdir("supervisor", 0700)
	writer2 := l4g.NewFileLogWriter("supervisor/supervisor.log", true)
	writer2.SetFormat("[%D %T] [%L] %M")
	// writer2.SetRotateDaily(true)
	supervisor_log.AddFilter("file", l4g.DEBUG, writer2)

	globalChatLimiter = rate.NewLimiter(rate.Limit(globalChatConfig.RateLimitHz), globalChatConfig.RateLimitBurst)

	potoq.RegisterPacketFilter(&packets.JoinGamePacketCB{}, resFilter)

	go automessage()
}

func ChatFilter(handler *potoq.Handler, rawPacket packets.Packet) error {
	input := rawPacket.(*packets.ChatMessagePacketSB)
	msg := input.Message

	err := checkMessage(input.Message)
	if err != nil {
		return err
	}

	if strings.HasPrefix(msg, "/") {
		log.Info(handler.Nickname + ": " + msg)
		supervisorMessage(handler, msg)
		words := strings.Split(msg, " ")
		switch strings.ToLower(words[0]) {
		case "/msg", "/m", "/tell", "/t":
			return msgCommand(handler, input, words)
		case "/reply", "/r":
			return replyCommand(handler, input, words)
		case "/ignore":
			return ignoreCommand(handler, input, words)
		case "/helpop":
			return helpopCommand(handler, input, words)
		case "/socialspy":
			return socialspyCommand(handler, input, words)
		case "/chatreload":
			return reloadCommand(handler, input, words)
		case "/chat":
			return chatoffCommand(handler, input, words)
		default:
			return nil
		}
	}

	if strings.HasPrefix(msg, "\\") && handler.HasPermission("cloudychat.command_forward") {
		// for local server command usage, eg. kick
		supervisorMessage(handler, msg)
		input.Message = strings.Replace(msg, "\\", "/", 1)
		return nil
	}

	if !handler.HasPermission("cloudychat.censorship.bypass") {
		msg = censorship(msg)
		if len(msg) < 1 {
			return potoq.ErrDropPacket
		}
	}

	if globalChatConfig.ChatOff && !handler.HasPermission("cloudychat.chatoff.bypass") {
		handler.SendChatMessage(packets.COLOR_RED + "Chat jest zablokowany!")
		return potoq.ErrDropPacket
	}

	if !handler.HasPermission("cloudychat.slowmode.bypass") {
		user := getChatUser(handler)
		if slowmode, ok := checkSlowmode(user.LastChatTime); !ok {
			secs := fmt.Sprintf("%.0f sekund", math.Ceil(slowmode.Seconds()))
			handler.SendChatMessage(packets.COLOR_RED + "Mozesz napisac na chacie za " + secs)
			return potoq.ErrDropPacket
		}
		user.LastChatTime = time.Now()
	}

	if !handler.HasPermission("cloudychat.ratelimiter.bypass") && !globalChatLimiter.Allow() {
		handler.SendChatMessage(packets.COLOR_RED + "Wykryto spam na chacie! Poczekaj chwile!")
		return potoq.ErrDropPacket
	}

	potoq.Players.Range(func(recipient *potoq.Handler) bool {
		fmt_msg := Formatter(recipient, handler, msg)
		if fmt_msg == "" {
			return true
		}
		json_msg, _ := utils.Para2Json(fmt_msg)
		recipient.InjectPackets(packets.ClientBound, nil,
			&packets.ChatMessagePacketCB{
				Message:  string(json_msg),
				Position: 0, // regular chat message
			},
		)
		return true
	})

	if MessageHook != nil {
		MessageHook(handler, "chat", msg)
	}

	log.Info(handler.Nickname + ": " + input.Message)

	return potoq.ErrDropPacket
}

func supervisorMessage(handler *potoq.Handler, msg string) {
	if !handler.HasPermission("cloudychat.supervisor.send") {
		return
	}
	supervisor_log.Info(handler.Nickname + ": " + msg)
	potoq.Players.Broadcast("cloudychat.supervisor.receive", packets.COLOR_GRAY+handler.Nickname+": "+msg)
	MessageHook(handler, "supervisor", msg)
}

func checkMessage(msg string) error {
	if len(msg) == 0 || len(msg) > 256 {
		return fmt.Errorf("Invalid message length!")
	}
	for _, char := range msg {
		if char == 167 || char < 32 {
			return fmt.Errorf("Invalid character in message (" + strconv.Itoa(int(char)) + ")!")
		}
	}
	return nil
}

func censorship(msg string) (censored string) {
	censored = msg
	globalChatLock.Lock()
	defer globalChatLock.Unlock()
	censorship := globalChatConfig.censorshipRegexp
	for exp, repl := range censorship {
		censored = exp.ReplaceAllLiteralString(censored, repl)
	}
	return
}

func checkSlowmode(t time.Time) (time.Duration, bool) {
	globalChatLock.Lock()
	defer globalChatLock.Unlock()
	dur := time.Duration(float32(potoq.Players.Len()) * globalChatConfig.SlowmodeFactor * float32(time.Second))
	if dur < 3*time.Second {
		dur = 3 * time.Second
	}
	return dur, time.Now().Sub(t) > dur
}

func DefaultFormatter(recipient, sender *potoq.Handler, msg string) string {
	nick := sender.Nickname
	if sender.Authenticator == nil {
		nick = "~" + nick
	}
	if sender.HasPermission("cloudychat.color") {
		nick = packets.COLOR_RED + nick
	}
	format := packets.COLOR_GRAY + "%s:" + packets.COLOR_WHITE + " %s"
	return fmt.Sprintf(format, nick, msg)
}
