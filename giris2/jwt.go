package giris2

import (
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/atcharles/glibs/crypto"
	"github.com/atcharles/glibs/j2rpc"
	"github.com/atcharles/glibs/util"
)

const (
	ErrorTokenMissing     = "禁止访问"
	ErrorTokenMalformed   = "令牌错误"
	ErrorTokenNotValidYet = "令牌不存在"
	ErrorTokenExpired     = "令牌过期"
	ErrorTokenInvalid     = "令牌无效"
)
const (
	JWTToken    = "JWT_TOKEN"
	ContextUser = "CONTEXT_USER"
)

var innerJWTKey = util.AppName + ":jwt"
var innerJwtCryptoKey = []byte("FUXqWKtBONoR85Eb")[:16]

// JWT is middleware for the RPC web framework that provides JWT authentication.
type JWT struct {
	Expire int64
	Store  ItfTokenStore
	once   sync.Once
}

// AfterLogin ...
func (j *JWT) AfterLogin(id string) (t *Token) {
	t = &Token{ID: id}
	t.ExpiresAt = util.TimeNow().Unix() + j.Expire
	t.Token = generateToken(id)
	j.Store.SetToken(getJWTKey(id), t)
	return
}

// Logout ...
func (j *JWT) Logout(id string) { j.Store.DelToken(getJWTKey(id)) }

// New ...
func (j *JWT) New() *JWT { return j.lazyInit() }

func (*JWT) User(ctx iris.Context) interface{} { return ctx.Values().Get(ContextUser) }

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
	c.Values().Set(JWTToken, token)
	id := getIDFromToken(token)
	if id == "" {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenMalformed)
		return
	}
	t, has := j.Store.GetToken(getJWTKey(id))
	if !has {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenNotValidYet)
		return
	}
	if t.expired() {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenExpired)
		return
	}
	if t.Token != token {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, ErrorTokenInvalid)
		return
	}
	user, err := call(t.ID)
	if err != nil {
		err = j2rpc.NewError(j2rpc.ErrAuthorization, err.Error())
		return
	}
	j.refreshToken(t)
	c.Values().Set(ContextUser, user)
	return
}

// lazyInit ......
func (j *JWT) lazyInit() *JWT {
	j.once.Do(func() {
		if j.Expire == 0 {
			// * days
			j.Expire = 3600 * 24 * 7
		}
		if j.Store == nil {
			j.Store = newDefaultTokenStore()
		}
		if j.Store != nil {
			j.runClearExpiredToken()
		}
	})
	return j
}

// refreshToken ...
func (j *JWT) refreshToken(t *Token) {
	if t.ExpiresAt-util.TimeNow().Unix() > j.Expire/2 {
		return
	}
	t.ExpiresAt = util.TimeNow().Unix() + j.Expire
	j.Store.SetToken(getJWTKey(t.ID), t)
}

func (j *JWT) runClearExpiredToken() { go util.Ticker(time.Hour*10, j.Store.ClearExpiredToken) }

type Token struct {
	ID        string `json:"id,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	Token     string `json:"token,omitempty"`
}

func (t *Token) expired() bool { return t.ExpiresAt < util.TimeNow().Unix() }

type defaultTokenStore struct {
	mu sync.RWMutex

	tokens map[string]*Token
}

func (d *defaultTokenStore) ClearExpiredToken() {
	d.mu.Lock()
	for k, v := range d.tokens {
		if v.expired() {
			delete(d.tokens, k)
		}
	}
	d.mu.Unlock()
}

func (d *defaultTokenStore) DelToken(key string) {
	d.mu.Lock()
	delete(d.tokens, key)
	d.mu.Unlock()
}

func (d *defaultTokenStore) GetToken(key string) (t *Token, has bool) {
	d.mu.RLock()
	t, has = d.tokens[key]
	d.mu.RUnlock()
	return
}

func (d *defaultTokenStore) SetToken(key string, t *Token) {
	d.mu.Lock()
	d.tokens[key] = t
	d.mu.Unlock()
}

func DefaultJWT() *JWT { return new(JWT).New() }

// generateToken ...
func generateToken(id string) string {
	v, _ := crypto.AESCBCEncrypt(innerJwtCryptoKey, []byte(id), true)
	return string(v)
}

func getIDFromToken(token string) string {
	v, _ := crypto.AESCBCDecrypt(innerJwtCryptoKey, []byte(token), true)
	return string(v)
}

func getJWTKey(id string) string { return innerJWTKey + ":" + id }

func newDefaultTokenStore() *defaultTokenStore {
	return &defaultTokenStore{tokens: make(map[string]*Token)}
}
