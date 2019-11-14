package permissions

import (
	"fmt"
	"strings"
)

type PermSet map[string]bool

func (ps PermSet) HasPermission(perm string) bool {
	for {
		if v, ok := ps[perm]; ok {
			return v
		}
		li := strings.LastIndex(perm, ".")
		if li < 0 {
			return false
		}
		perm = perm[:li]
	}
}

func (ps PermSet) SetPermission(perm string, value bool) {
	ps[perm] = value
}

func (ps PermSet) Apply(perms PermSet, root string, override bool) error {
	for k, v := range perms {
		if !strings.HasPrefix(k, root+".") && k != root && root != "" {
			return fmt.Errorf("Capability %s outside of root %s!", k, root)
		}
		if old, has_key := ps[k]; !override && has_key && old != v {
			return fmt.Errorf("Conflict on %s capability! %t != %t", k, old, v)
		}
		ps[k] = v
	}
	return nil
}
