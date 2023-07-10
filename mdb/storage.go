package mdb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/karlseguin/ccache/v3"
	"github.com/redis/go-redis/v9"

	"github.com/atcharles/glibs/util"
)

//------------------------------------------------------------ redisStore -----------------------------------------------------------

func GetRedisStore(c *redis.Client) CacheStore {
	return util.LoadSingleInstance[CacheStore]("GetRedisStore", func() CacheStore {
		return &redisStore{c: c}
	})
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

func HDeleter(c *redis.Client, key string) func(fields ...string) error {
	ctx := context.Background()
	const l = 100
	sl := make([]string, 0, l)
	return func(fields ...string) (err error) {
		sl = append(sl, fields...)
		if len(sl) <= 0 {
			return
		}
		if len(sl) < l && len(fields) > 0 {
			return
		}
		r, e := c.HDel(ctx, key, sl...).Result()
		if e != nil {
			return e
		}
		if r != int64(len(sl)) {
			return fmt.Errorf("deleted %d, but expected %d", r, len(sl))
		}
		sl = sl[:0]
		return
	}
}

type redisValue struct {
	Val json.RawMessage `json:"val"`

	ExpireAt int64 `json:"expire_at"`
}

// isExpired ...
func (r *redisValue) isExpired() bool {
	return r.ExpireAt > 0 && r.ExpireAt < time.Now().Unix()
}

type redisStore struct {
	c *redis.Client
}

// scanKeys ...
func (r *redisStore) scanKeys(p string) []string {
	m := make([]string, 0)
	ctx := context.Background()
	iter := r.c.Scan(ctx, 0, fmt.Sprintf("*%s*", p), 1000).Iterator()
	for iter.Next(ctx) {
		m = append(m, iter.Val())
	}
	return m
}

func (r *redisStore) StoreGC(prefix string) {
	prefix = fmt.Sprintf("%s:%s:", globalPrefix, prefix)
	keys := r.scanKeys(prefix)
	if len(keys) == 0 {
		return
	}
	r.gc(keys)
}

// background run gc
func (r *redisStore) gc(keys []string) {
	for _, key := range keys {
		deleter := HDeleter(r.c, key)
		HScanCallback(r.c, key, "*", func(k, v string) {
			var rv redisValue
			err := json.Unmarshal([]byte(v), &rv)

			if err != nil || rv.isExpired() {
				_ = deleter(k)
			}
		})
		_ = deleter()
	}
}

func (*redisStore) getKF(key string) (k, f string, y bool) {
	s1 := strings.Split(key, separator)
	if len(s1) != 2 {
		return
	}
	k = s1[0]
	f = s1[1]
	y = true
	return
}

func (r *redisStore) Get(key string) (data []byte, ok bool) {
	k, f, y := r.getKF(key)
	if !y {
		return
	}

	v, err := r.c.HGet(context.Background(), k, f).Result()
	if err != nil {
		return
	}
	var rv redisValue
	if err = json.Unmarshal([]byte(v), &rv); err != nil {
		return
	}

	if rv.isExpired() {
		r.Del(key)
		return
	}

	data = rv.Val
	ok = true
	return
}

func (r *redisStore) Set(key string, data []byte, ttl ...int64) {
	k, f, y := r.getKF(key)
	if !y {
		return
	}

	var expireAt int64
	if len(ttl) > 0 && ttl[0] > 0 {
		expireAt = time.Now().Unix() + ttl[0]
	}
	rv := redisValue{
		Val:      data,
		ExpireAt: expireAt,
	}
	v, err := json.Marshal(rv)
	if err != nil {
		return
	}
	r.c.HSet(context.Background(), k, f, string(v))
}

func (r *redisStore) Del(key string) {
	k, f, y := r.getKF(key)
	if !y {
		return
	}

	r.c.HDel(context.Background(), k, f)
}

func (r *redisStore) ClearAll() {
	r.c.FlushDB(context.Background())
}

func (r *redisStore) DropPrefix(prefix ...string) {
	if len(prefix) == 0 {
		return
	}

	for _, s := range prefix {
		keys := r.scanKeys(s)
		if len(keys) == 0 {
			continue
		}
		r.c.Del(context.Background(), keys...)
	}
}

//------------------------------------------------------------ memStore -----------------------------------------------------------

func GetMemStore() CacheStore {
	return util.LoadSingleInstance[CacheStore]("GetMemStore", func() CacheStore {
		return new(memStore).lazyInitialize()
	})
}

type memStore struct {
	once sync.Once

	c *badger.DB
}

func (m *memStore) GetDB() *badger.DB { return m.c }

// lazyInitialize ...
func (m *memStore) lazyInitialize() *memStore {
	m.once.Do(func() {
		opts := badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(badger.ERROR)
		opts.IndexCacheSize = 100 << 20
		var err error
		m.c, err = badger.Open(opts)
		if err != nil {
			panic(err)
		}
	})
	return m
}

func (m *memStore) Get(key string) (data []byte, ok bool) {
	db := m.c
	e := db.View(func(t *badger.Txn) (err error) {
		item, err := t.Get([]byte(key))
		if err != nil {
			return
		}
		data, err = item.ValueCopy(nil)
		return
	})
	ok = e == nil
	return
}

func (m *memStore) Set(key string, data []byte, ttl ...int64) {
	var t time.Duration
	if len(ttl) > 0 && ttl[0] > 0 {
		t = time.Duration(ttl[0]) * time.Second
	}
	db := m.c
	_ = db.Update(func(txn *badger.Txn) (err error) {
		et := badger.NewEntry([]byte(key), data)
		if t > 0 {
			et = et.WithTTL(t)
		}
		return txn.SetEntry(et)
	})
}

func (m *memStore) Del(key string) {
	db := m.c
	_ = db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (m *memStore) DropPrefix(prefix ...string) {
	var pbs [][]byte
	for _, s := range prefix {
		pbs = append(pbs, []byte(s))
	}
	_ = m.c.DropPrefix(pbs...)
}

func (m *memStore) ClearAll() {
	_ = m.c.DropAll()
}

//------------------------------------------------------------ cc -----------------------------------------------------------

func GetCCacheStore() CacheStore {
	return util.LoadSingleInstance[CacheStore]("GetCCacheStore", func() CacheStore {
		return new(ccStore).lazyInitialize()
	})
}

type ccStore struct {
	c    *ccache.Cache[[]byte]
	mu   sync.RWMutex
	once sync.Once
}

// lazyInitialize ...
func (c *ccStore) lazyInitialize() *ccStore {
	c.once.Do(func() {
		c.c = ccache.New(
			ccache.Configure[[]byte]().
				MaxSize(100 * 10000).
				ItemsToPrune(1000),
		)
	})
	return c
}

func (c *ccStore) Get(key string) (data []byte, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item := c.c.Get(key)
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

func (c *ccStore) Set(key string, data []byte, ttl ...int64) {
	c.mu.Lock()
	var t = time.Hour
	if len(ttl) > 0 && ttl[0] > 0 {
		t = time.Duration(ttl[0]) * time.Second
	}
	c.c.Set(key, data, t)
	c.mu.Unlock()
}

func (c *ccStore) Del(key string) {
	c.mu.Lock()
	c.c.Delete(key)
	c.mu.Unlock()
}

func (c *ccStore) ClearAll() {
	c.mu.Lock()
	c.c.Clear()
	c.mu.Unlock()
}

func (c *ccStore) DropPrefix(prefix ...string) {
	c.mu.Lock()
	for _, s := range prefix {
		c.c.DeletePrefix(s)
	}
	c.mu.Unlock()
}
