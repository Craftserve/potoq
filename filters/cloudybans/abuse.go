package cloudybans

import (
	"fmt"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/Craftserve/potoq/packets" // chat colors
	"github.com/Craftserve/potoq/utils"   // chat converter
)

type Abuse struct {
	Id               int
	Kind             string // ban or mute
	UUID             uuid.UUID
	Player           string
	Moderator        string
	Ts               time.Time
	Comment          string
	IP               string
	Expires          time.Time
	Pardon_moderator *string
	Pardon_comment   *string
	Pardon_ts        *time.Time
}

func (b *Abuse) String() string {
	// #1234 JakisCheater99 13.09.27_13:11 -> 13.12.31_14:22 [Cubixmeister] 'straszny zly cheater'
	head := fmt.Sprintf("#%d%s %s %s->%s [%s]\n> '%s' ", b.Id, upperFirstChar(b.Kind), b.Player, b.Ts.Format(TIME_FORMAT), b.Expires.Format(TIME_FORMAT), b.Moderator, b.Comment)
	if b.Expires.Before(time.Now()) {
		head = head + "EXPIRED"
	}
	if b.Pardon_moderator != nil {
		// PARDONED 13.09.27_20:33 [IROLL23] 'to byla pomylka'
		head = head + fmt.Sprintf("\n> PARDONED %s [%s] '%s'", b.Pardon_ts.Format(TIME_FORMAT), *b.Pardon_moderator, *b.Pardon_comment)
	}
	return head
}

func (b *Abuse) BanKickMsg() string {
	if b.Kind != "ban" {
		panic("getBanKickMsg() is valid only for Abuses with 'ban' kind!")
	}
	msg, _ := utils.Para2Json(fmt.Sprintf(packets.COLOR_RED+"Zostales zbanowany przez %s do %s z powodu: %s", b.Moderator, b.Expires.Format(PLAYER_TIME_FORMAT), b.Comment))
	return string(msg)
}

func (b *Abuse) BanMsg() string {
	if b.Kind != "ban" {
		panic("getBanKickMsg() is valid only for Abuses with 'ban' kind!")
	}
	return fmt.Sprintf(packets.COLOR_RED+"Zostales zbanowany przez %s do %s z powodu: %s", b.Moderator, b.Expires.Format(PLAYER_TIME_FORMAT), b.Comment)
}

func upperFirstChar(s string) string {
	for _, r := range s {
		return string(unicode.ToUpper(r))
	}
	return "?"
}
