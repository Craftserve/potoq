package cloudychat

import "github.com/Craftserve/potoq"
import "time"

func automessage() {
	for {
		message, delay := getNextAutoMessage()
		if message != "" {
			potoq.Players.Broadcast("", globalChatConfig.Automessage_prefix+message)
		}
		time.Sleep(delay)
	}
}

var nextAutoMessage int

func getNextAutoMessage() (msg string, delay time.Duration) {
	globalChatLock.Lock()
	defer globalChatLock.Unlock()

	delay = time.Duration(globalChatConfig.Automessage_delay) * time.Second

	if len(globalChatConfig.Automessage) < 1 {
		return
	}

	msg = globalChatConfig.Automessage[nextAutoMessage]
	nextAutoMessage = (nextAutoMessage + 1) % len(globalChatConfig.Automessage)
	return
}
