package tab

import (
	"fmt"
	"reflect"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"

	"github.com/google/uuid"
)

var DefaultTextures = packets.AuthProperty{ // dark grey square
	Name:      "textures",
	Value:     "eyJ0aW1lc3RhbXAiOjE0MTEyNjg3OTI3NjUsInByb2ZpbGVJZCI6IjNmYmVjN2RkMGE1ZjQwYmY5ZDExODg1YTU0NTA3MTEyIiwicHJvZmlsZU5hbWUiOiJsYXN0X3VzZXJuYW1lIiwidGV4dHVyZXMiOnsiU0tJTiI6eyJ1cmwiOiJodHRwOi8vdGV4dHVyZXMubWluZWNyYWZ0Lm5ldC90ZXh0dXJlLzg0N2I1Mjc5OTg0NjUxNTRhZDZjMjM4YTFlM2MyZGQzZTMyOTY1MzUyZTNhNjRmMzZlMTZhOTQwNWFiOCJ9fX0=",
	Signature: "u8sG8tlbmiekrfAdQjy4nXIcCfNdnUZzXSx9BE1X5K27NiUvE1dDNIeBBSPdZzQG1kHGijuokuHPdNi/KXHZkQM7OJ4aCu5JiUoOY28uz3wZhW4D+KG3dH4ei5ww2KwvjcqVL7LFKfr/ONU5Hvi7MIIty1eKpoGDYpWj3WjnbN4ye5Zo88I2ZEkP1wBw2eDDN4P3YEDYTumQndcbXFPuRRTntoGdZq3N5EBKfDZxlw4L3pgkcSLU5rWkd5UH4ZUOHAP/VaJ04mpFLsFXzzdU4xNZ5fthCwxwVBNLtHRWO26k/qcVBzvEXtKGFJmxfLGCzXScET/OjUBak/JEkkRG2m+kpmBMgFRNtjyZgQ1w08U6HHnLTiAiio3JswPlW5v56pGWRHQT5XWSkfnrXDalxtSmPnB5LmacpIImKgL8V9wLnWvBzI7SHjlyQbbgd+kUOkLlu7+717ySDEJwsFJekfuR6N/rpcYgNZYrxDwe4w57uDPlwNL6cJPfNUHV7WEbIU1pMgxsxaXe8WSvV87qLsR7H06xocl2C0JFfe2jZR4Zh3k9xzEnfCeFKBgGb4lrOWBu1eDWYgtKV67M2Y+B3W5pjuAjwAxn0waODtEn/3jKPbc/sxbPvljUCw65X+ok0UUN1eOwXV5l2EGzn05t3Yhwq19/GxARg63ISGE8CKw=",
}

type TabList struct {
	handler      *potoq.Handler
	send_full    bool
	cells        [80]packets.PlayerListItem
	changed_dn   []packets.PlayerListItem
	changed_ping []packets.PlayerListItem
	title        *packets.PlayerListTitlePacketCB
}

func NewTabList(handler *potoq.Handler) *TabList {
	tb := &TabList{}
	tb.handler = handler
	tb.send_full = true

	for i := range tb.cells {
		tb.cells[i] = packets.PlayerListItem{
			UUID:        uuid.MustParse(fmt.Sprintf("00000000-0000-%04d-0000-000000000000", i)),
			Name:        "", //fmt.Sprintf("~cell#%04d", i),
			Properties:  []packets.AuthProperty{DefaultTextures},
			GameMode:    0,
			Ping:        1,
			DisplayName: `{"text":""}`,
		}
	}

	return tb
}

func (tb *TabList) cell(x, y int) *packets.PlayerListItem {
	id := x*20 + y
	if x > 3 || x < 0 || y > 19 || y < 0 || id >= len(tb.cells) {
		panic(fmt.Errorf("Invalid cell: %d, %d", x, y))
	}
	return &tb.cells[id]
}

func (tb *TabList) SetProperties(x, y int, properties []packets.AuthProperty) {
	cell := tb.cell(x, y)
	if properties == nil {
		properties = []packets.AuthProperty{DefaultTextures}
	}
	if !reflect.DeepEqual(cell.Properties, properties) { // optymalizacja?
		cell.Properties = properties
		tb.send_full = true
	}
}

func (tb *TabList) SetPing(x, y int, ping int) {
	cell := tb.cell(x, y)
	if cell.Ping == packets.VarInt(ping) {
		return
	}
	cell.Ping = packets.VarInt(ping)
	tb.changed_ping = append(tb.changed_ping, *cell)
}

func (tb *TabList) SetCell(x, y int, chat_format string, args ...string) {
	cell := tb.cell(x, y)

	display_name := chat_format
	if len(args) > 0 {
		display_name = packets.Chatf(display_name, args...)
	}
	if len(display_name) > 64 {
		display_name = `{"text":"TOO_LONG_STR"}`
	}

	if cell.DisplayName != display_name {
		cell.DisplayName = display_name
		tb.changed_dn = append(tb.changed_dn, *cell)
	}
}

func (tb *TabList) SetTitle(header, footer string) {
	tb.title = &packets.PlayerListTitlePacketCB{
		Header: header,
		Footer: footer,
	}
}

func (tb *TabList) Flush() error {
	var p []packets.Packet
	if tb.send_full {
		p = append(p, &packets.PlayerListItemPacketCB{
			Action: packets.ADD_PLAYER,
			Items:  tb.cells[:],
		})
		tb.send_full = false
		tb.changed_dn = nil
		tb.changed_ping = nil
	}

	if len(tb.changed_dn) > 0 {
		p = append(p, &packets.PlayerListItemPacketCB{
			Action: packets.UPDATE_DISPLAY_NAME,
			Items:  tb.changed_dn,
		})
		tb.changed_dn = nil
	}

	if len(tb.changed_ping) > 0 {
		p = append(p, &packets.PlayerListItemPacketCB{
			Action: packets.UPDATE_LATENCY,
			Items:  tb.changed_ping,
		})
		tb.changed_ping = nil
	}

	if tb.title != nil {
		p = append(p, tb.title)
		tb.title = nil
	}

	if len(p) < 1 {
		return nil
	}
	return tb.handler.InjectPackets(packets.ClientBound, nil, p...)
}
