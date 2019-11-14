package cloudybans

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/filters/bungeecord"
	"github.com/Craftserve/potoq/packets"
	"github.com/Craftserve/potoq/utils"
)

const insuffientPermsMsg = packets.COLOR_RED + "Insufficient permissions!"

func CommandsFilter(handler *potoq.Handler, packet packets.Packet) error {
	msg := packet.(*packets.ChatMessagePacketSB)
	if len(msg.Message) < 1 {
		return nil
	}

	words := strings.Fields(msg.Message)
	sender := CommandSender{handler.Nickname, handler.HasPermission, handler.SendChatMessage}
	drop := HandleCommand(sender, words)
	if drop {
		return potoq.ErrDropPacket
	}
	return nil
}

func HandleCommand(sender CommandSender, words []string) (handled bool) {
	var resp string
	switch strings.ToLower(words[0]) {
	case "/ban":
		resp = banCmd("ban", false, sender, words[1:])
	case "/sban":
		resp = banCmd("ban", true, sender, words[1:])
	case "/unban":
		resp = unbanCmd("ban", sender, words[1:])
	case "/bang":
		resp = bangCmd(sender, words[1:])
	case "/mute":
		resp = banCmd("mute", false, sender, words[1:])
	case "/unmute":
		resp = unbanCmd("mute", sender, words[1:])
	case "/baninfo":
		resp = baninfoCmd(sender, words[1:])
	case "/kick":
		resp = kickCmd(sender, false, words[1:])
	case "/skick":
		resp = kickCmd(sender, true, words[1:])
	case "/track":
		resp = trackCmd(sender, words[1:])
	case "/trackip":
		resp = trackipCmd(sender, words[1:])
	case "/revdns":
		resp = revdnsCmd(sender, words[1:])
	default:
		return false
	}

	if resp != "" {
		sender.SendChatMessage(resp)
	}

	return true
}

type CommandSender struct {
	Nickname        string
	HasPermission   func(string) bool
	SendChatMessage func(string)
}

// /ban nickname time komentarz komentarz...
func banCmd(abuse_kind string, silent bool, handler CommandSender, args []string) string {
	if !handler.HasPermission("cloudybans." + abuse_kind) { // cloudybans.mute or cloudybans.ban
		return insuffientPermsMsg
	}
	if len(args) < 3 {
		return packets.COLOR_RED + "Bad command syntax! /" + abuse_kind + " <nickname> <duration> comment [comment...]"
	}

	last_login, err := GetLastLogin(args[0])
	if err != nil {
		return fmt.Sprintf(packets.COLOR_RED+"Database error: %s", err)
	}

	dur, err := parseDuration(args[1])
	if err != nil {
		return fmt.Sprintf(packets.COLOR_RED + "Error while parsing duration! Valid format: \\d+[smhdM], eg. 5d == 5 days")
	}
	var expires time.Time
	if dur > 0 {
		expires = time.Now().Add(dur)
	}

	ban := &Abuse{
		Kind:      abuse_kind,
		Player:    last_login.Player,
		UUID:      last_login.UUID,
		Moderator: handler.Nickname,
		Ts:        time.Now(),
		Comment:   strings.Join(args[2:], " "),
		Expires:   expires,
		IP:        last_login.IP,
	}

	err = dbmap.Insert(ban)
	if err != nil {
		return fmt.Sprintf(packets.COLOR_RED+"Database error: %s", err)
	}
	invalidateCaches(ban)

	if abuse_kind == "ban" {
		if !silent {
			potoq.Players.Broadcast("", packets.COLOR_RED+fmt.Sprintf("BAN: %s do %s -> %s [%s]", ban.Player, ban.Expires.Format(PLAYER_TIME_FORMAT), ban.Comment, ban.Moderator))
		}
		player := potoq.Players.GetByNickname(ban.Player)
		if player != nil {
			err := player.InjectPackets(packets.ClientBound, io.EOF, &packets.KickPacketCB{ban.BanKickMsg()})
			if err != nil {
				return packets.COLOR_RED + "Error while kicking player: " + err.Error()
			}
		}
	}

	return packets.COLOR_GREEN + ban.String()
}

func bangCmd(handler CommandSender, args []string) string {
	if !handler.HasPermission("cloudybans.bang") {
		return insuffientPermsMsg
	}
	if len(args) < 3 {
		return packets.COLOR_RED + "Bad command syntax! /bang <group_id> <duration> comment [comment...]"
	}

	dur, err := parseDuration(args[1])
	if err != nil {
		return fmt.Sprintf(packets.COLOR_RED + "Error while parsing duration! Valid format: \\d+[smhdM], eg. 5d == 5 days")
	}
	var expires time.Time
	if dur > 0 {
		expires = time.Now().Add(dur)
	}

	if GroupFinder == nil {
		return fmt.Sprintf(packets.COLOR_RED+"Error finding group %s: GroupFinder is not implemented!", args[0])
	}

	players, err := GroupFinder(args[0])
	if err != nil {
		return fmt.Sprintf(packets.COLOR_RED+"Error finding group %s: %s", args[0], err)
	}
	var nicknames []string

	for i, p_uuid := range players {
		last_login, err := GetLastLogin(p_uuid.String())
		if err != nil {
			return fmt.Sprintf(packets.COLOR_RED+"Database error: %s, %s", err, p_uuid)
		}
		nicknames = append(nicknames, last_login.Player)

		ban := &Abuse{
			Kind:      "ban",
			Player:    last_login.Player,
			UUID:      last_login.UUID,
			Moderator: handler.Nickname,
			Ts:        time.Now(),
			Comment:   strings.Join(args[2:], " "),
			Expires:   expires,
			IP:        last_login.IP,
		}

		err = dbmap.Insert(ban) // ta seria insertow to w sumie chyba powinna transakcja byc?
		if err != nil {
			return fmt.Sprintf(packets.COLOR_RED+"Database error (%d/%d): %s", i, len(players), err)
		}
		invalidateCaches(ban)

		player := potoq.Players.GetByNickname(ban.Player)
		if player != nil {
			var buffer bytes.Buffer
			bungeecord.WriteJavaUTF(&buffer, "Ban")
			bungeecord.WriteJavaUTF(&buffer, ban.Player)
			bungeecord.WriteJavaUTF(&buffer, ban.BanMsg())

			player.InjectPackets(packets.ServerBound, nil, &packets.PluginMessagePacketSB{"BungeeCord", buffer.Bytes()})
			time.AfterFunc(500*time.Millisecond, func() {
				player.InjectPackets(packets.ServerBound, io.EOF, &packets.KickPacketCB{ban.BanKickMsg()})
			})
		}
	}

	potoq.Players.Broadcast("", fmt.Sprintf(packets.COLOR_RED+"BAN GRUPOWY: %s "+packets.COLOR_GRAY+"(%d:%s)"+packets.COLOR_RED+" do %s -> %s [%s]",
		args[0], len(nicknames), strings.Join(nicknames, ","), expires.Format(PLAYER_TIME_FORMAT), strings.Join(args[2:], " "), handler.Nickname))
	return ""
}

func unbanCmd(abuse_kind string, handler CommandSender, args []string) string {
	if !handler.HasPermission("cloudybans.un" + abuse_kind) {
		return insuffientPermsMsg
	}
	if len(args) < 2 {
		return packets.COLOR_RED + "Bad command syntax! /un" + abuse_kind + " <ban_id> <nickname> <reason...>"
	}

	ban_id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Sprintf(packets.COLOR_RED+"Error while parsing ban_id! %s", err)
	}

	b, err1 := dbmap.Get(Abuse{}, ban_id)
	if b == nil {
		return fmt.Sprintf(packets.COLOR_RED+"Ban #%d not found!", ban_id)
	}
	if err1 != nil {
		return fmt.Sprintf(packets.COLOR_RED+"Database error(1): %s", err)
	}
	ban := b.(*Abuse)

	if !strings.EqualFold(ban.Player, args[1]) {
		return fmt.Sprintf(packets.COLOR_RED+"Ban #%d nickname mismatch! %s != %s", ban_id, ban.Player, args[1])
	}

	if ban.Pardon_moderator != nil {
		return fmt.Sprintf(packets.COLOR_RED+"Ban #%d has been already pardoned by %s on %s with reason: %s", ban_id, ban.Pardon_moderator, ban.Pardon_ts, ban.Pardon_comment)
	}

	if !(strings.EqualFold(ban.Moderator, handler.Nickname) || handler.HasPermission("cloudybans.un"+abuse_kind+".others")) {
		return fmt.Sprintf(packets.COLOR_RED+"Ban #%d has been added by %s on %s. You don't have sufficient permissions to remove other's bans.", ban_id, ban.Pardon_moderator, ban.Pardon_ts)
	}

	n := handler.Nickname
	ban.Pardon_moderator = &n
	c := strings.Join(args[2:], " ")
	ban.Pardon_comment = &c
	t := time.Now()
	ban.Pardon_ts = &t

	_, err2 := dbmap.Update(ban)
	if err2 != nil {
		return fmt.Sprintf(packets.COLOR_RED+"Database error(2): %s", err)
	}
	invalidateCaches(ban)

	return packets.COLOR_GREEN + ban.String()
}

// /baninfo #ban_id or nickname
func baninfoCmd(handler CommandSender, args []string) string {
	if !handler.HasPermission("cloudybans.baninfo") {
		return insuffientPermsMsg
	}
	if len(args) < 1 {
		return packets.COLOR_RED + "Bad command syntax! /baninfo <#banid or nickname or uuid>"
	}

	var bans []*Abuse

	if strings.HasPrefix(args[0], "#") {
		ban_id, err := strconv.Atoi(args[0][1:])
		if err != nil {
			return fmt.Sprintf(packets.COLOR_RED+"Error while parsing ban_id! %s", err)
		}
		_, err = dbmap.Select(&bans, "SELECT * FROM cloudyBans_list WHERE id = ? LIMIT 1", ban_id)
		if err != nil {
			return fmt.Sprintf(packets.COLOR_RED+"Database error(1): %s", err)
		}
	} else {
		_, err := dbmap.Select(&bans, "SELECT * FROM cloudyBans_list WHERE (player = ? OR uuid = ?) ORDER BY id ASC", args[0], args[0])
		if err != nil {
			return fmt.Sprintf(packets.COLOR_RED+"Database error(2): %s", err)
		}
	}

	if len(bans) == 0 {
		return packets.COLOR_GREEN + "No matching bans found!"
	}

	resp := make([]string, 0, len(bans))
	for _, b := range bans {
		c := packets.COLOR_GREEN
		if b.Pardon_moderator != nil {
			c = packets.COLOR_GRAY
		}
		resp = append(resp, c+b.String())
	}
	return strings.Join(resp, "\n")
}

func kickCmd(handler CommandSender, silent bool, args []string) string {
	if !handler.HasPermission("cloudybans.kick") {
		return insuffientPermsMsg
	}
	if len(args) < 2 {
		return packets.COLOR_RED + "Bad command syntax! /kick <nickname> <reason...>"
	}

	player := potoq.Players.GetByNickname(args[0])
	if player == nil {
		return packets.COLOR_RED + "Player not found!"
	}

	ban := &Abuse{
		Kind:      "kick",
		Player:    player.Nickname,
		UUID:      player.UUID,
		Moderator: handler.Nickname,
		Ts:        time.Now(),
		Comment:   strings.Join(args[1:], " "),
		Expires:   time.Now(),
		IP:        player.DownstreamAddr,
	}
	err := dbmap.Insert(ban)
	if err != nil {
		return fmt.Sprintf(packets.COLOR_RED+"Database error: %s", err)
	}

	msg := strings.Join(args[1:], " ")

	kick_msg, _ := utils.Para2Json(fmt.Sprintf(packets.COLOR_RED+"KICKED BY %s\n%s", handler.Nickname, msg))
	kick_packet := packets.KickPacketCB{string(kick_msg)}
	inject_err := player.InjectPackets(packets.ClientBound, io.EOF, &kick_packet)
	if inject_err != nil {
		return packets.COLOR_RED + "Error while trying to kick player: " + inject_err.Error()
	}
	if !silent {
		potoq.Players.Broadcast("", fmt.Sprintf(packets.COLOR_RED+"KICK: %s -> %s [%s]", player.Nickname, msg, handler.Nickname))
	} else {
		return packets.COLOR_GREEN + "Kicked!"
	}
	return ""
}

var durationExpr = regexp.MustCompile("^(\\d+)([smhdM])$")

func parseDuration(s string) (time.Duration, error) {
	m := durationExpr.FindStringSubmatch(s)
	if len(m) == 0 {
		return 0, fmt.Errorf("Bad duration format!")
	}
	n, _ := strconv.Atoi(m[1])
	num := time.Duration(n)
	var d time.Duration
	switch m[2] { // [smhdM]
	case "s":
		d = num * time.Second
	case "m":
		d = num * time.Minute
	case "h":
		d = num * time.Hour
	case "d":
		d = num * time.Hour * 24
	case "M":
		d = num * time.Hour * 24 * 30
	default:
		panic("Unexpected unit in parseDuration: " + m[2])
	}
	return d, nil
}
