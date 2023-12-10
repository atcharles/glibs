package gemq

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"

	"github.com/atcharles/glibs/config"
	"github.com/atcharles/glibs/giris"
	"github.com/atcharles/glibs/util"
)

var API = new(AuthAPI)

type AuthAPI struct {
	AuthFunc func(c iris.Context, d *mqttData) (err error)
}

// ACL ......
func (a *AuthAPI) ACL(d *mqttData) (err error) {
	if !strings.HasPrefix(d.Topic, config.C.AppName()+"/") {
		return fmt.Errorf("[%s]refused", util.JSONDump(d))
	}
	username, _ := Inc.EmqSuperAuth()
	if d.Username == username {
		return
	}
	if d.Access == 1 {
		return
	}
	if strings.HasSuffix(d.Topic, "/pong") {
		return
	}
	return fmt.Errorf("[%s]refused", util.JSONDump(d))
}

// Auth ......
func (a *AuthAPI) Auth(c iris.Context, d *mqttData) (err error) {
	username, password := Inc.EmqSuperAuth()
	if d.Username == username {
		if d.Password != password {
			return errors.New("super password error")
		}
		return
	}
	c.Request().Header.Set("token", d.Password)
	if a.AuthFunc != nil {
		return a.AuthFunc(c, d)
	}
	return
}

// Handle ...
func (a *AuthAPI) Handle(r iris.Party) iris.Party {
	p := r.Party("/mqtt")
	ac := accesslog.New(util.ZapLogger("mqtt").Writer())
	p.UseRouter(ac.Handler)
	p.Post("/auth", giris.Func2Handler(a.Auth))
	p.Post("/acl", giris.Func2Handler(a.ACL))
	return p
}

type mqttData struct {
	//(1 - subscribe, 2 - publish)
	Access   int    `form:"access"`
	Ipaddr   string `form:"ipaddr"`
	Topic    string `form:"topic"`
	Username string `form:"username"`
	Password string `form:"password"`
	ClientID string `form:"clientid"`
}
