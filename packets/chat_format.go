package packets

import (
	"encoding/json"
	"fmt"
	"text/template"
)

func Chatf(json_format string, args ...string) string {
	var size int = len(json_format)
	var args_safe = make([]interface{}, len(args))
	for i, v := range args {
		v = template.JSEscapeString(v)
		args_safe[i] = v
		size += len(v)
	}
	data := fmt.Sprintf(json_format, args_safe...)
	if !json.Valid([]byte(data)) {
		return `{"text":"ErrChatJsonInvalid","color":"red"}`
	}
	return data
}

// TODO: nie wiem dlaczego ten plik w ogole istnieje i co go uzywa (sekcja ponizej)
// TODO: to nie jest pelna implementacja chatu, prawdziwa jest w utils/chat.go

type ChatMessage struct {
	Text  string     `json:"text"`
	Click ChatAction `json:"clickEvent"`
	Hover ChatAction `json:"hoverEvent"`
}

type ChatAction struct {
	Action string `json:"action"`
	Value  string `json:"value"`
}

// > Chat actions
const CLICK_OPEN_URL = "open_url"           // opens url
const CLICK_OPEN_FILE = "open_file"         // opens file
const CLICK_RUN_CMD = "run_command"         // runs command
const CLICK_SUGGEST_CMD = "suggest_command" // places text in player's text box

const HOVER_SHOW_TEXT = "show_text"               // displays text (can be json)
const HOVER_SHOW_ACHIEVEMENT = "show_achievement" // displays achievement of given name
const HOVER_SHOW_ITEM = "show_item"               // displays item - must be quoted in json format - {id:35,Damage:5,Count:2,tag:{display:{Name:Testing}}}

// > Colors
const COLOR_BLACK = "\u00A70"
const COLOR_DARK_BLUE = "\u00A71"
const COLOR_DARK_GREEN = "\u00A72"
const COLOR_DARK_AQUA = "\u00A73"
const COLOR_DARK_RED = "\u00A74"
const COLOR_DARK_PURPLE = "\u00A75"
const COLOR_GOLD = "\u00A76"
const COLOR_GRAY = "\u00A77"
const COLOR_DARK_GRAY = "\u00A78"
const COLOR_BLUE = "\u00A79"
const COLOR_GREEN = "\u00A7a"
const COLOR_AQUA = "\u00A7b"
const COLOR_RED = "\u00A7c"
const COLOR_LIGHT_PURPLE = "\u00A7d"
const COLOR_YELLOW = "\u00A7e"
const COLOR_WHITE = "\u00A7f"
const COLOR_MAGIC = "\u00A7k"
const COLOR_BOLD = "\u00A7l"
const COLOR_STRIKETHROUGH = "\u00A7m"
const COLOR_UNDERLINE = "\u00A7n"
const COLOR_ITALIC = "\u00A7o"
const COLOR_RESET = "\u00A7r"
