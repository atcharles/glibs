package giris

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kataras/iris/v12"

	"github.com/atcharles/glibs/config"
	"github.com/atcharles/glibs/j2rpc"
	"github.com/atcharles/glibs/mdb"
	"github.com/atcharles/glibs/util"
)

const (
	JWTToken    = "JWT_TOKEN"
	ContextUser = "CONTEXT_USER"
)

const (
	ErrorTokenExpired     = "令牌已过期"
	ErrorTokenNotValidYet = "令牌不存在"
	ErrorTokenMalformed   = "错误的令牌"
	ErrorTokenInvalid     = "令牌已失效"
	ErrorTokenMissing     = "非法访问"
)

const innerJwtCryptoKey = "FUXqWKtBONoR85Eb"

var crypto = util.NewCrypto(util.TrimKey(innerJwtCryptoKey)[:16])

// SetCrypto ... must replace with your own crypto
func SetCrypto(k string) {
	crypto = util.NewCrypto(util.TrimKey(k))
}

func QueueClearExpiredPattern() string {
	return fmt.Sprintf("%s:jwt:clear_expired", config.C.AppName())
}

var IncJWT = newJWT()

func newJWT() *JWT {
	return &JWT{
		Expire:   3600 * 24 * 30,
		memStore: mdb.GetMemStore(),
	}
}

// JWT is a middleware for the RPC web framework that provides JWT authentication.
type JWT struct {
	// jwt settings
	Expire int64

	memStore mdb.CacheStore
}

// lazyInit ......
func (j *JWT) lazyInit() *JWT {
	if j.Expire == 0 {
		// * days
		j.Expire = 3600 * 24 * 30
	}
	return j
}

// ClearExpiredFunc Clear expired tokens
func (j *JWT) ClearExpiredFunc() func() error {
	rdb := mdb.Rdb
	return func() error {
		rc := rdb.GetClient()
		key := j.key()
		deleter := mdb.HDeleter(rc, key)
		var err error
		mdb.HScanCallback(rc, key, "*", func(k, v string) {
			var t Token
			err = json.Unmarshal([]byte(v), &t)
			if err != nil || t.Expired() {
				err = deleter(k)
			}
		})
		return deleter()
	}
}

// Logout ...
func (j *JWT) Logout(id string) {
	rdb := mdb.Rdb
	rc := rdb.GetClient()
	key := j.key()
	rc.HDel(context.Background(), key, id)
}

// AfterLogin ...
func (j *JWT) AfterLogin(id string) (t *Token, err error) {
	t = &Token{ID: id}
	if err = j.toRedis(t, false); err != nil {
		return
	}
	return
}

// Verify verifies the token and returns the user ID if the token is valid.
func (j *JWT) Verify(c iris.Context, call func(id string) (user interface{}, err error)) (err error) {
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.URLParam("token")
	}
	if token == "" {
		token = c.GetHeader("token")
	}
	token = strings.Replace(token, "Bearer ", "", -1)
	if token == "" {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenMissing)
		return
	}
	// Parse the token
	t, err := j.fromRedis(token)
	if err != nil {
		return
	}
	c.Values().Set(JWTToken, token)
	// Call the user function
	user, err := call(t.ID)
	if err != nil {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, err.Error())
		return
	}
	c.Values().Set(ContextUser, user)
	return
}

func (*JWT) User(ctx iris.Context) interface{} { return ctx.Values().Get(ContextUser) }

func (*JWT) key() string { return fmt.Sprintf("%s:jwt", config.C.AppName()) }

// toRedis ......
func (j *JWT) toRedis(t *Token, isRefresh bool) (err error) {
	j.lazyInit()
	if t.Token == "" || isRefresh {
		t.Token = generateToken(t.ID)
	}
	t.ExpiresAt = util.TimeNow().Unix() + j.Expire

	rdb := mdb.Rdb
	rc := rdb.GetClient()
	key := j.key()
	storedString, err := json.Marshal(t)
	if err != nil {
		return
	}
	err = rc.HSet(context.Background(), key, t.ID, storedString).Err()
	if err != nil {
		return
	}

	j.memStore.Set(j.memeKey(t), util.JsMarshal(t), j.Expire)
	return
}

// memeKey ......
func (j *JWT) memeKey(t *Token) string { return fmt.Sprintf("%s:%s", j.key(), t.ID) }

// fromRedis ...
func (j *JWT) fromRedis(tokenStr string) (t *Token, err error) {
	j.lazyInit()
	t, err = parseToken(tokenStr)
	if err != nil {
		return
	}
	v, ok := j.memStore.Get(j.memeKey(t))
	func() {
		if ok {
			err = json.Unmarshal(v, t)
			return
		}
		rdb := mdb.Rdb
		rc := rdb.GetClient()
		key := j.key()
		storedString := rc.HGet(context.Background(), key, t.ID).Val()
		if storedString == "" {
			err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenNotValidYet)
			return
		}
		err = json.Unmarshal([]byte(storedString), t)
		if err != nil {
			err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenMalformed)
			return
		}
		j.memStore.Set(j.memeKey(t), util.JsMarshal(t), 60)
	}()
	if tokenStr != t.Token {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenInvalid)
		return
	}
	if t.Expired() {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenExpired)
		return
	}
	// UpdateParams the token if now is after 1/2 of the expiry time
	if util.TimeNow().Unix() > t.ExpiresAt-j.Expire/2 {
		if err = j.toRedis(t, true); err != nil {
			return
		}
	}
	return
}

// parseToken ...
func parseToken(str string) (t *Token, err error) {
	id, err := crypto.DecryptCBC(strings.ToLower(str))
	if err != nil {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenMalformed)
		return
	}
	t = &Token{ID: id}
	return
}

// generateToken ...
func generateToken(id string) string {
	return strings.ToUpper(crypto.EncryptCBC(id))
}

type Token struct {
	ID        string `json:"id,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	Token     string `json:"token,omitempty"`
}

func (t *Token) Expired() bool {
	return t.ExpiresAt < util.TimeNow().Unix()
}
