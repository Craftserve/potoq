package cloudychat

import (
	"strings"
	"sync"
	"time"

	"github.com/Craftserve/potoq"
)

type chatUser struct {
	sync.Mutex
	MsgRecipient   string
	LastChatTime   time.Time
	LastHelpopTime time.Time
	SentPrivMsgs   []string
	Ignored        map[string]bool
}

func (user *chatUser) SetMsgRecipient(nick string) {
	user.Lock()
	defer user.Unlock()
	user.MsgRecipient = nick
}

func (user *chatUser) GetMsgRecipient() string {
	user.Lock()
	defer user.Unlock()
	return user.MsgRecipient
}

func (user *chatUser) AddPrivMsgHistory(recipient, msg string) {
	user.Lock()
	defer user.Unlock()
	if user.SentPrivMsgs == nil {
		user.SentPrivMsgs = make([]string, 20)
	}
	n := copy(user.SentPrivMsgs[:], user.SentPrivMsgs[1:])
	user.SentPrivMsgs[n] = msg // chyba bedzie dzialac z tym n xD
	user.MsgRecipient = recipient
}

func (user *chatUser) GetIgnored(nick string) bool {
	user.Lock()
	defer user.Unlock()
	return user.Ignored != nil && user.Ignored[strings.ToLower(nick)]
}

func (user *chatUser) SetIgnored(nick string, val bool) {
	user.Lock()
	defer user.Unlock()
	if user.Ignored == nil {
		user.Ignored = make(map[string]bool)
	}
	if len(user.Ignored) >= 256 {
		return // rate-limiting for dummies...
	}
	user.Ignored[strings.ToLower(nick)] = val
}

const chatDataKey = "filters/cloudychat.chatUser"

func getChatUser(handler *potoq.Handler) *chatUser {
	value, _ := handler.FilterData.LoadOrStore(chatDataKey, &chatUser{})
	return value.(*chatUser)
}
