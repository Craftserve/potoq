package permissions

// Filters and helpers for easy runtime permission management

// TODO: chyba czas to na uuid przerobic

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/Craftserve/potoq"
	"github.com/Craftserve/potoq/packets"
)

var GlobalPermLock sync.Mutex
var Groups map[string]PermSet // dlaczego nie prywatne?
var Users map[string]PermSet

func RegisterFilters() {
	potoq.RegisterPacketFilter(&packets.ChatMessagePacketSB{}, onChatMessage)
	load_err := LoadPermissions()
	if load_err != nil {
		panic(load_err)
	}
}

// Loads groups and users from file, thread-safe
// On error doesn't modify global state.
func LoadPermissions() error {
	var tmp_groups map[string]PermSet
	err := loadAndUnmarshalYaml("perm_groups.yml", &tmp_groups)
	if err != nil || len(tmp_groups) < 1 {
		return fmt.Errorf("Error loading perm_groups.yml: %s", err)
	}

	var user_groups map[string][]string
	err = loadAndUnmarshalYaml("perm_users.yml", &user_groups)
	if err != nil || len(user_groups) < 1 {
		return fmt.Errorf("Error loading perm_users.yml: %s", err)
	}

	// calculate PermissionSet for each user
	tmp_users := make(map[string]PermSet)
	for nickname, grps := range user_groups {
		u_perms := make(PermSet)
		for _, g_name := range grps {
			g_perms, found := tmp_groups[g_name]
			if !found {
				return fmt.Errorf("Group %s not found for user %s!", g_name, nickname)
			}
			err = u_perms.Apply(g_perms, "", true)
			if err != nil {
				return fmt.Errorf("Error applying %s for user %s: %s", g_name, nickname, err)
			}
		}
		tmp_users[strings.ToLower(nickname)] = u_perms
	}

	// if successfull save to global variables
	GlobalPermLock.Lock()
	Groups = tmp_groups
	Users = tmp_users
	GlobalPermLock.Unlock()

	return nil
}

func loadAndUnmarshalYaml(filename string, value interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("Error while reading perm_users.yml: %s", err)
	}
	defer f.Close()
	err = yaml.NewDecoder(f).Decode(value)
	if err != nil {
		return fmt.Errorf("Error while parsing perm_users.yml: %s", err)
	}
	return nil
}

// Returns calculated players permissions. Default group is applied first. Thread-safe
func GetPlayerPermissions(handler *potoq.Handler) (ps PermSet) {
	GlobalPermLock.Lock()
	defer GlobalPermLock.Unlock()

	ps = make(PermSet)
	ps.Apply(Groups["default"], "", true)
	ps.Apply(Users[strings.ToLower(handler.Nickname)], "", true)
	return
}

func ApplyGroup(ps PermSet, group string, override bool) error {
	GlobalPermLock.Lock()
	defer GlobalPermLock.Unlock()

	g, ok := Groups[group]
	if !ok {
		return fmt.Errorf("Unkown permissions group: %s", group)
	}
	return ps.Apply(g, "", override)
}
