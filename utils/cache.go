package utils

import "fmt"
import "sync"
import "time"
import "reflect"

type cacheEntry struct {
	value   interface{}
	expires time.Time
}

type ExpiringCache struct {
	t reflect.Type
	s int
	m map[string]*cacheEntry
	l sync.Mutex
}

func NewExpiringCache(valid_type interface{}, size int) (cache *ExpiringCache) {
	return &ExpiringCache{
		t: reflect.TypeOf(valid_type),
		s: size,
		m: make(map[string]*cacheEntry),
	}
}

func (cache *ExpiringCache) Get(key string) (value interface{}, ok bool) {
	cache.l.Lock()
	defer cache.l.Unlock()

	entry := cache.m[key]
	if entry == nil {
		return nil, false
	}
	if entry.expires.Before(time.Now()) {
		delete(cache.m, key)
		return nil, false
	}
	return entry.value, true
}

func (cache *ExpiringCache) SetTime(key string, value interface{}, expire_time time.Time) {
	cache.l.Lock()
	defer cache.l.Unlock()

	value_type := reflect.TypeOf(value)
	if value_type != cache.t {
		panic(fmt.Sprintf("Invalid type in cache Set(): %s != %s", value_type.Name(), cache.t.Name()))
	}

	if expire_time.Before(time.Now()) {
		return
	}

	if len(cache.m)+1 > cache.s {
		cache.m = make(map[string]*cacheEntry)
	}

	entry := &cacheEntry{
		value:   value,
		expires: expire_time,
	}
	cache.m[key] = entry
}

func (cache *ExpiringCache) SetDelay(key string, value interface{}, expire_delay time.Duration) {
	cache.SetTime(key, value, time.Now().Add(expire_delay))
}

func (cache *ExpiringCache) Invalidate(key string) {
	cache.l.Lock()
	defer cache.l.Unlock()

	delete(cache.m, key)
}

func (cache *ExpiringCache) InvalidateAll() {
	cache.l.Lock()
	defer cache.l.Unlock()

	cache.m = make(map[string]*cacheEntry)
}

// ValueCache

type ValueCache struct {
	t reflect.Type
	l sync.Mutex
	e time.Time
	v interface{}
}

func (cache *ValueCache) Get() (value interface{}, found bool) {
	value, found = cache.GetOrLock()
	if !found {
		cache.l.Unlock()
	}
	return
}

// return value if found, locks otherwise
func (cache *ValueCache) GetOrLock() (value interface{}, found bool) {
	cache.l.Lock()
	if cache.e.IsZero() || cache.e.Before(time.Now()) {
		return nil, false
	}
	cache.l.Unlock()
	return cache.v, true
}

func (cache *ValueCache) setv(value interface{}) {
	value_type := reflect.TypeOf(value)
	if cache.t != nil {
		if value_type != cache.t {
			panic(fmt.Sprintf("Invalid type in cache SetDelay(): %s != %s", value_type.Name(), cache.t.Name()))
		}
	} else {
		cache.t = value_type
	}
	cache.v = value
}

func (cache *ValueCache) SetDelay(value interface{}, ttl time.Duration) {
	cache.l.Lock()
	defer cache.l.Unlock()

	cache.e = time.Now().Add(ttl)
	cache.setv(value)
}

// set value without aquiring lock
func (cache *ValueCache) SetUnsafe(value interface{}, ttl time.Duration) {
	cache.e = time.Now().Add(ttl)
	cache.setv(value)
}

func (cache *ValueCache) Unlock() {
	cache.l.Unlock()
}
