package mdb

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/karlseguin/ccache/v3"
	"github.com/redis/go-redis/v9"

	"github.com/atcharles/glibs/util"
)

type CCStore struct {
	c    *ccache.Cache[[]byte]
	mu   sync.RWMutex
	once sync.Once
}

func (c *CCStore) ClearAll() {
	c.mu.Lock()
	c.c.Clear()
	c.mu.Unlock()
}

func (c *CCStore) Del(key string) {
	c.mu.Lock()
	c.c.Delete(key)
	c.mu.Unlock()
}

func (c *CCStore) DropPrefix(prefix ...string) {
	c.mu.Lock()
	for _, s := range prefix {
		c.c.DeletePrefix(s)
	}
	c.mu.Unlock()
}

func (c *CCStore) Get(key string) (data []byte, ok bool) {
	c.mu.RLock()
	item := c.c.Get(key)
	c.mu.RUnlock()
	if item == nil {
		return
	}
	if item.Expired() {
		c.Del(key)
		return
	}
	data = item.Value()
	ok = true
	return
}

// GetCCache ...
func (c *CCStore) GetCCache() *ccache.Cache[[]byte] { return c.lazyInitialize().c }

func (c *CCStore) Set(key string, data []byte, ttl ...int64) {
	c.mu.Lock()
	var t = time.Hour
	if len(ttl) > 0 && ttl[0] > 0 {
		t = time.Duration(ttl[0]) * time.Second
	}
	c.c.Set(key, data, t)
	c.mu.Unlock()
}

// lazyInitialize ...
func (c *CCStore) lazyInitialize() *CCStore {
	c.once.Do(func() {
		if c.c == nil {
			c.c = ccache.New(ccache.Configure[[]byte]().
				MaxSize(10 * 10000).ItemsToPrune(1000))
		}
	})
	return c
}

type FreeStore struct {
	c    *freecache.Cache
	mu   sync.RWMutex
	once sync.Once
}

func (f *FreeStore) ClearAll() {
	f.mu.Lock()
	f.c.Clear()
	f.mu.Unlock()
}

func (f *FreeStore) Del(key string) {
	f.mu.Lock()
	f.c.Del([]byte(key))
	f.mu.Unlock()
}

func (f *FreeStore) DropPrefix(prefix ...string) {
	it := f.c.NewIterator()
	for {
		entry := it.Next()
		if entry == nil {
			break
		}
		k := string(entry.Key)
		for _, p := range prefix {
			if strings.HasPrefix(k, p) {
				f.Del(k)
			}
		}
	}
}

func (f *FreeStore) Get(key string) (data []byte, ok bool) {
	f.mu.RLock()
	d, err := f.c.Get([]byte(key))
	f.mu.RUnlock()
	if err != nil {
		return
	}
	data = d
	ok = true
	return
}

func (f *FreeStore) GetFC() *freecache.Cache { return f.lazyInitialize().c }

func (f *FreeStore) Set(key string, data []byte, ttl ...int64) {
	exp := 3600
	if len(ttl) > 0 {
		exp = int(ttl[0])
	}
	f.mu.Lock()
	_ = f.c.Set([]byte(key), data, exp)
	f.mu.Unlock()
}

// lazyInitialize ...
func (f *FreeStore) lazyInitialize() *FreeStore {
	f.once.Do(func() {
		if f.c == nil {
			//100M
			f.c = freecache.NewCache(100 << 20)
		}
	})
	return f
}

func GetCCacheStore() *CCStore { return util.LoadSingle(NewCCacheStore) }

func GetFreeCacheStore() *FreeStore { return util.LoadSingle(NewFreeCacheStore) }

func HDeleter(c *redis.Client, key string) func(fields ...string) {
	ctx := context.Background()
	const l = 100
	sl := make([]string, 0, l)
	return func(fields ...string) {
		sl = append(sl, fields...)
		if len(sl) <= 0 {
			return
		}
		if len(sl) < l && len(fields) > 0 {
			return
		}
		c.HDel(ctx, key, sl...)
		sl = sl[:0]
	}
}

func HScanCallback(c *redis.Client, key, match string, fn func(k, v string)) {
	ctx := context.Background()
	iter := c.HScan(ctx, key, 0, match, 1000).Iterator()
	s := make([]string, 0)
	for iter.Next(ctx) {
		s = append(s, iter.Val())
		n := len(s)
		if n > 1 && n%2 == 0 {
			fn(s[n-2], s[n-1])
		}
	}
}

func NewCCacheStore() *CCStore { return new(CCStore).lazyInitialize() }

func NewFreeCacheStore() *FreeStore { return new(FreeStore).lazyInitialize() }