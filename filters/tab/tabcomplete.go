package tab

import (
	"strings"
	"unicode"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

func RegisterFilters() {
	potoq.RegisterPacketFilter(&packets.TabCompletePacketSB{}, tabFilter)
}

func tabFilter(handler *potoq.Handler, packet packets.Packet) error {
	query := packet.(*packets.TabCompletePacketSB)
	if !handler.HasPermission("tabcomplete.enable") {
		return potoq.ErrDropPacket
	}

	i := strings.LastIndexFunc(query.Text, unicode.IsSpace)
	if i < 1 {
		return handler.Log.Error("tabFilter: i: %d len: %d", i, len(query.Text))
	}
	word := query.Text[i+1:]

	response := packets.TabCompletePacketCB{
		TransactionId: query.TransactionId,
		Start:         packets.VarInt(i + 1),
		Length:        packets.VarInt(len(query.Text) - (i + 1)),
	}

	ignore_exempt := handler.HasPermission("tabcomplete.ignore_exempt")

	potoq.Players.Range(func(h *potoq.Handler) bool {
		if !hasPrefixFold(h.Nickname, word) {
			return true
		}
		if h.HasPermission("tabcomplete.exempt") && !ignore_exempt {
			return true
		}
		response.Matches = append(response.Matches, packets.TabCompleteMatch{
			Match: h.Nickname,
			//Tooltip:
		})
		return true
	})

	if len(response.Matches) > 0 {
		err := handler.InjectPackets(packets.ClientBound, nil, &response)
		if err != nil {
			return err
		}
	}
	return potoq.ErrDropPacket
}

// case insensitive strings.HasPrefix
func hasPrefixFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return strings.EqualFold(s[:len(prefix)], prefix)
}
