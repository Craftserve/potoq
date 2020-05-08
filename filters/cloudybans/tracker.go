package cloudybans

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

type Login struct {
	Id             int
	Player         string
	UUID           uuid.UUID
	Premium        bool
	IP             string
	Ts             time.Time
	End            *time.Time
	AuthProperties *[]byte
}

func TrackPlayerLogin(handler *potoq.Handler) (login Login, err error) {
	login = Login{
		Player:  handler.Nickname,
		UUID:    handler.UUID,
		Premium: handler.Authenticator != nil,
		IP:      handler.DownstreamAddr,
		Ts:      time.Now(),
	}

	if handler.Authenticator != nil && handler.AuthProperties != nil {
		authprops, err := json.Marshal(handler.AuthProperties)
		if err != nil {
			return login, err
		}
		login.AuthProperties = &authprops
	}

	err = dbmap.Insert(&login)
	if err != nil {
		return login, fmt.Errorf("cloudyBans: error in LoginInfo insert: %+v, %w", login, err)
	}

	handler.FilterData.Store("cloudyBans.Login", login)

	handler.CloseHooks = append(handler.CloseHooks, func() {
		_, err := dbmap.Exec("UPDATE cloudyBans_logins SET end = NOW() WHERE id = ?", login.Id)
		if err != nil {
			handler.Log().WithError(err).WithFields(logrus.Fields{
				"loginID": login.Id,
			}).Error("cloudyBans: logins end update error")
		}
	})

	logins, err2 := dbmap.SelectInt("SELECT COUNT(DISTINCT uuid) AS `logins` FROM `cloudyBans_logins` WHERE `ip` = ? AND `ts` > NOW() - INTERVAL 1 MONTH;", handler.DownstreamAddr)
	if err2 != nil {
		handler.Log().WithError(err).Error("cloudyBans: error in LoginInfo select")
	}
	if logins > 1 {
		potoq.Players.Broadcast("cloudybans.track", fmt.Sprint(packets.COLOR_RED, "Gracz ", handler.Nickname, " logowal sie w tym miesiacu na IP: ", formatIp(handler.DownstreamAddr, !handler.HasPermission("cloudybans.showfullip")), " na ", logins, " roznych kont!"))
	}

	return login, nil
}

func GetLastLogin(nickname_or_uuid string) (login Login, err error) {
	err = dbmap.SelectOne(&login,
		"SELECT * FROM cloudyBans_logins WHERE (player = ? OR uuid = ?) ORDER BY id DESC LIMIT 1",
		nickname_or_uuid, nickname_or_uuid)
	return
}

func GetFirstLogin(nickname_or_uuid string) (login Login, err error) {
	err = dbmap.SelectOne(&login,
		"SELECT * FROM cloudyBans_logins WHERE (player = ? OR uuid = ?) ORDER BY id ASC LIMIT 1",
		nickname_or_uuid, nickname_or_uuid)
	return
}

func formatIp(ip string, shadowed bool) string {
	if !shadowed {
		return ip
	}
	parts := strings.Split(ip, ".")
	if len(parts) < 4 {
		return ip
	}
	parts[3] = "X"
	return strings.Join(parts, ".")
}

func trackCmd(handler CommandSender, args []string) string {
	if !handler.HasPermission("cloudybans.track") {
		return insuffientPermsMsg
	}
	if len(args) != 1 {
		return packets.COLOR_RED + "Bad command syntax! /track <nickname or uuid>"
	}

	nick := args[0]
	var logins []struct {
		IP string
		C  int
	}
	_, err := dbmap.Select(&logins, "SELECT ip, count(*) as c FROM cloudyBans_logins WHERE (player=? or uuid=?) GROUP BY ip ORDER BY c DESC", nick)
	if err != nil {
		return packets.COLOR_RED + "Error while trying to track player"
	}
	if len(logins) == 0 {
		return packets.COLOR_RED + "No data about this player!"
	}

	showfullip := handler.HasPermission("cloudybans.showfullip")

	for _, login := range logins {
		var others []struct {
			Player string
			C      int
		}
		_, err = dbmap.Select(&others, "SELECT player, count(*) as c FROM cloudyBans_logins WHERE ip=? AND player!=? GROUP BY player ORDER BY c DESC", login.IP, nick)
		if err != nil {
			return packets.COLOR_RED + "Error while trying to track player"
		}

		var resp strings.Builder
		ip := formatIp(login.IP, !showfullip)
		fmt.Fprintf(&resp, packets.COLOR_GREEN+"%s: %d logins, others (%d): %s", ip, login.C, len(others))
		for _, other := range others {
			fmt.Fprintf(&resp, "%s(%d) ", other.Player, other.C)
		}
		handler.SendChatMessage(resp.String())
	}

	return ""
}

func trackipCmd(handler CommandSender, args []string) string {
	if !handler.HasPermission("cloudybans.trackip") {
		return insuffientPermsMsg
	}
	if len(args) != 1 {
		return packets.COLOR_RED + "Bad command syntax! /trackip <ip>"
	}

	ip := args[0]
	var logins []struct {
		Player string
		C      int
	}
	_, err := dbmap.Select(&logins, "SELECT player, count(*) as c FROM cloudyBans_logins WHERE ip=? GROUP BY player ORDER BY c DESC", ip)
	if err != nil {
		return packets.COLOR_RED + "Error while trying to track player"
	}
	if len(logins) == 0 {
		return packets.COLOR_RED + "No data about this ip!"
	}

	showfullip := handler.HasPermission("cloudybans.showfullip")

	for _, login := range logins {
		var others []struct {
			IP string
			C  int
		}
		_, err = dbmap.Select(&others, "SELECT ip, count(*) as c FROM cloudyBans_logins WHERE player=? AND ip!=? GROUP BY ip ORDER BY c DESC", login.Player, ip)
		if err != nil {
			return packets.COLOR_RED + "Error while trying to track player"
		}

		var resp strings.Builder
		fmt.Fprintf(&resp, packets.COLOR_GREEN+"%s: %d logins, others (%d): ", login.Player, login.C, len(others))
		for _, other := range others {
			shadowed := formatIp(other.IP, !showfullip)
			fmt.Fprintf(&resp, "%s(%d) ", shadowed, other.C)
		}
		handler.SendChatMessage(resp.String())
	}

	return ""
}

func revdnsCmd(handler CommandSender, args []string) string {
	if !handler.HasPermission("cloudybans.revdns") {
		return insuffientPermsMsg
	}
	if len(args) != 1 {
		return packets.COLOR_RED + "Bad command syntax! /revdns <ip>"
	}

	ip := args[0]
	lookup, err := net.LookupAddr(ip)

	if err != nil {
		return packets.COLOR_RED + "Lookup error"
	}

	return packets.COLOR_GREEN + "Host: " + strings.Join(lookup, ", ")
}
