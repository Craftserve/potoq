package cloudychat

import (
	"time"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
	"github.com/Craftserve/potoq/utils"
)

func PingHandler(packet *packets.HandshakePacket) packets.ServerStatus {
	motd, _, omul, favicondata := GetPingData()
	p := int(float32(potoq.Players.Len()) * omul)
	return packets.ServerStatus{
		Version:     packets.GameVersion,
		Players:     packets.ServerStatusPlayers{p, p + 1},
		Description: motd,
		Favicon:     favicondata,
	}
}

var motdCache utils.ValueCache

func GetPingData() (string, int, float32, string) {
	globalChatLock.Lock()
	defer globalChatLock.Unlock()

	var motd string
	v, found := motdCache.Get()
	if found {
		motd = v.(string)
	} else {
		redis.Do(radix.Cmd(&motd, "GET", "cloudyChat_MOTD"))
		if motd == "" {
			motd = globalChatConfig.MOTD
		}
		motdCache.SetDelay(motd, 5*time.Second)
	}

	return motd, globalChatConfig.Slots, globalChatConfig.OnlineMultiplier, globalChatConfig.faviconData
}
