package potoq

import "strings"
import "sync"
import "sync/atomic"
import "github.com/google/uuid"

type PlayerManager struct {
	m     sync.Map
	count int64
}

func (pm *PlayerManager) Register(handler *Handler) bool {
	key := strings.ToLower(handler.Nickname)
	_, conflict := pm.m.LoadOrStore(key, handler)
	if conflict {
		return false
	}
	atomic.AddInt64(&pm.count, 1)
	return true
}

func (pm *PlayerManager) Unregister(handler *Handler) {
	key := strings.ToLower(handler.Nickname)
	pm.m.Delete(key)
	atomic.AddInt64(&pm.count, -1)
}

func (pm *PlayerManager) GetByNickname(nickname string) *Handler {
	key := strings.ToLower(nickname)
	v, ok := pm.m.Load(key)
	if !ok {
		return nil
	}
	return v.(*Handler)
}

func (pm *PlayerManager) GetByUUID(u uuid.UUID) (result *Handler) {
	pm.Range(func(h *Handler) bool {
		if u == h.UUID {
			result = h
			return false
		}
		return true
	})
	return result
}

func (pm *PlayerManager) Len() int {
	return int(atomic.LoadInt64(&pm.count))
}

func (pm *PlayerManager) Range(f func(handler *Handler) bool) {
	pm.m.Range(func(key, value interface{}) bool {
		return f(value.(*Handler))
	})
}

func (pm *PlayerManager) Broadcast(perm, msg string) {
	pm.Range(func(h *Handler) bool {
		if perm == "" || h.HasPermission(perm) {
			h.SendChatMessage(msg)
		}
		return true
	})
}
