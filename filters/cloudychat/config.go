package cloudychat

import (
	"github.com/Craftserve/potoq"
	"io/ioutil"
)
import "gopkg.in/yaml.v2"
import "regexp"
import "strings"
import "encoding/base64"

type chatConfig struct {
	SlowmodeFactor     float32
	Censorship         map[string]string
	Automessage_delay  int
	Automessage_prefix string
	Automessage        []string
	censorshipRegexp   map[*regexp.Regexp]string
	ChatOff            bool
	RateLimitHz        int
	RateLimitBurst     int
	Slots              int
	OnlineMultiplier   float32
	MOTD               string
	faviconData        string // url-encoded favicon data
	Resource_pack_url  string `yaml:"resource_pack_url,omitempty"`
}

func LoadConfig(filename string) (config *chatConfig, err error) {
	var data []byte
	data, err = ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	config = new(chatConfig)
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return
	}

	config.Automessage_prefix = strings.Replace(config.Automessage_prefix, "&", "\u00A7", -1)
	for i, msg := range config.Automessage {
		config.Automessage[i] = strings.Replace(msg, "&", "\u00A7", -1)
	}

	config.censorshipRegexp = make(map[*regexp.Regexp]string, len(config.Censorship))
	for exp, repl := range config.Censorship {
		var compiled *regexp.Regexp
		compiled, err = regexp.Compile(exp)
		if err != nil {
			return
		}
		config.censorshipRegexp[compiled] = repl
	}

	if config.RateLimitHz == 0 {
		config.RateLimitHz = 2
		config.RateLimitBurst = 10
	}

	if config.OnlineMultiplier == 0 {
		config.OnlineMultiplier = 1
	}

	if fv, err := ioutil.ReadFile("favicon.png"); err == nil {
		config.faviconData = "data:image/png;base64," + base64.StdEncoding.EncodeToString(fv)
	} else {
		potoq.Log.WithError(err).Error("cloudyChat: favicon.png load error (ignored)")
	}

	return
}
