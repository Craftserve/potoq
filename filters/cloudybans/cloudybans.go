package cloudybans

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/gorp.v2"
	"github.com/sirupsen/logrus"
	"github.com/mediocregopher/radix"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
	"github.com/Craftserve/potoq/utils"
)

const TIME_FORMAT = "06.01.02_15:04"
const PLAYER_TIME_FORMAT = "02.01.2006 15:04"

var redis radix.Client
var dbmap *gorp.DbMap

var abuseCache = utils.NewExpiringCache(&Abuse{}, 50000)

var GroupFinder func(string) ([]uuid.UUID, error)

func RegisterFilters(dbmap_ *gorp.DbMap, redis_ radix.Client) {
	dbmap = dbmap_
	dbmap.AddTableWithName(Abuse{}, "cloudyBans_list").SetKeys(true, "Id")
	dbmap.AddTableWithName(Login{}, "cloudyBans_logins").SetKeys(true, "Id")
	redis = redis_

	potoq.RegisterPacketFilter(&packets.ChatMessagePacketSB{}, MuteFilter)
	potoq.RegisterPacketFilter(&packets.ChatMessagePacketSB{}, CommandsFilter)
	//potoq.RegisterPacketFilter(&packets.HandshakePacket{}, HandshakeFilter)
}

func GetAbuse(kind string, nickname_or_uuid string, cache bool) (abuse *Abuse, err error) {
	cache_key := strings.ToLower(kind + ":" + nickname_or_uuid)
	if cache {
		if v, ok := abuseCache.Get(cache_key); ok {
			return v.(*Abuse), nil
		}
	}

	abuses, err := ListAbuses(kind, nickname_or_uuid)
	if err != nil {
		return nil, err
	}
	if len(abuses) > 0 {
		abuse = abuses[0]
	}

	cache_ttl := time.Now().Add(time.Minute * 5)
	if abuse != nil && cache_ttl.After(abuse.Expires) {
		cache_ttl = abuse.Expires
	}
	abuseCache.SetTime(cache_key, abuse, cache_ttl)

	return
}

func invalidateCaches(abuse *Abuse) {
	abuseCache.Invalidate(strings.ToLower(abuse.Kind + ":" + abuse.Player))
	abuseCache.Invalidate(strings.ToLower(abuse.Kind + ":" + abuse.UUID.String()))
	err := redis.Do(radix.Cmd(nil, "PUBLISH", "cloudyBans.invalidate",
		strings.ToLower(abuse.Kind+":"+abuse.UUID.String()+":"+abuse.Player),
	))
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"abuse": abuse,
		}).Error("cloudyBans: invalidateCaches redis error")
	}
}

func ListAbuses(kind string, nickname_or_uuid string) (abuses []*Abuse, err error) {
	_, err = dbmap.Select(&abuses,
		`SELECT * FROM cloudyBans_list
			WHERE kind=?
				AND ( player=? OR uuid=? )
				AND IFNULL(expires, NOW()) >= NOW()
				AND pardon_ts IS NULL
		ORDER BY expires ASC`, kind, nickname_or_uuid, nickname_or_uuid)
	return
}

func GetFirstKnownNickname(handler *potoq.Handler) (string, error) {
	return dbmap.SelectStr("SELECT player FROM cloudyBans_logins WHERE uuid=? ORDER BY id ASC LIMIT 1", handler.UUID.String())
}
