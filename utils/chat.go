package utils

import "strings"
import "regexp"
import "encoding/json"

var URL_EXPR = regexp.MustCompile("(https?\\://)?[a-zA-Z0-9\\-\\.]+\\.[a-zA-Z]{2,3}(/\\S*)?")
var http_prefix_expr = regexp.MustCompile(`https?://`)

type ChatPart struct {
	Text  string      `json:"text"`
	Extra []*ChatPart `json:"extra,omitempty"`
	// styles and colors
	Color         string `json:"color,omitempty"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Underlined    bool   `json:"underlined,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	Obfuscated    bool   `json:"obfuscated,omitempty"`
	// Executes action after click: open_url, open_file, rn_command, suggest_command
	ClickEvent *ChatAction `json:"clickEvent,omitempty"`
	// Executes action after hover: show_text, show_achievement, show_item
	HoverEvent *ChatAction `json:"hoverEvent,omitempty"`
}

type ChatAction struct {
	Action string `json:"action"`
	Value  string `json:"value"`
}

// convert old §-based chat to new json format with clickable urls
func Para2Json(text string) (result []byte, err error) {
	rdr := strings.NewReader(text + "§r")
	var parts []*ChatPart
	var bold, italic, underline, strike, obfuscated bool
	var color, buf string = "", ""
	for rdr.Len() > 0 {
		char, _, err := rdr.ReadRune()
		if err != nil {
			return nil, err // invalid rune
		}
		if char != '§' { // non-control characted, add to buffer
			buf += string(char)
			continue
		}

		// make new part with current state and buffered text
		if loc := URL_EXPR.FindStringIndex(buf); loc != nil {
			a, b := loc[0], loc[1]

			if buf[:a] != "" {
				p1 := &ChatPart{Text: buf[:a], Bold: bold, Italic: italic, Underlined: underline, Strikethrough: strike, Obfuscated: obfuscated, Color: color}
				parts = append(parts, p1)
			}

			url := buf[a:b]
			if !http_prefix_expr.MatchString(url) {
				url = `http://` + url
			}

			p2 := &ChatPart{Text: buf[a:b], Bold: bold, Italic: italic, Underlined: underline, Strikethrough: strike, Obfuscated: obfuscated, Color: color}
			p2.ClickEvent = &ChatAction{"open_url", url}
			parts = append(parts, p2)

			buf = buf[b:]
		}

		if buf != "" {
			p := &ChatPart{Text: buf, Bold: bold, Italic: italic, Underlined: underline, Strikethrough: strike, Obfuscated: obfuscated, Color: color}
			parts = append(parts, p)
			buf = ""
		}

		// read mode for new part
		mode, _, err := rdr.ReadRune()
		if err != nil {
			mode = 'r'
		}
		switch mode {
		case '{': // click event

		case 'l', 'L':
			bold = true
		case 'm', 'M':
			strike = true
		case 'n', 'N':
			underline = true
		case 'o', 'O':
			italic = true
		default: // reset all styles on color change
			bold, strike, underline, italic = false, false, false, false
			switch mode {
			case '0':
				color = "black"
			case '1':
				color = "dark_blue"
			case '2':
				color = "dark_green"
			case '3':
				color = "dark_aqua"
			case '4':
				color = "dark_red"
			case '5':
				color = "dark_purple"
			case '6':
				color = "gold"
			case '7':
				color = "gray"
			case '8':
				color = "dark_gray"
			case '9':
				color = "blue"
			case 'a', 'A':
				color = "green"
			case 'b', 'B':
				color = "aqua"
			case 'c', 'C':
				color = "red"
			case 'd', 'D':
				color = "light_purple"
			case 'e', 'E':
				color = "yellow"
			case 'f', 'F':
				color = "white"
			case 'r', 'R': // reset
				if color != "white" && color != "" {
					color = "white"
				}
			}
		}
	}

	if len(parts) == 1 {
		return json.Marshal(parts[0])
	}
	return json.Marshal(parts)
}
