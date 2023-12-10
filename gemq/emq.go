package gemq

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/eclipse/paho.mqtt.golang"

	"github.com/atcharles/glibs/config"
	"github.com/atcharles/glibs/util"
)

var (
	// ErrTokenTimeout declare
	ErrTokenTimeout = errors.New("等待emq服务器Token超时")
	Inc             = new(Emq)
)

// Emq ...
type Emq struct {
	Config *config.Config `inject:""`
	Logger util.ItfLogger `inject:""`

	opt         *mqtt.ClientOptions
	client      mqtt.Client
	qos         byte
	retained    bool
	wait        time.Duration
	topicPrefix string

	pubC chan func()
}

// Client ...
func (e *Emq) Client() mqtt.Client { return e.client }

// Constructor ...
func (e *Emq) Constructor() {
	e.qos = 1
	e.retained = false
	e.wait = time.Second * 2
	e.topicPrefix = fmt.Sprintf("%s/", config.C.AppName())
	e.pubC = make(chan func(), 1000)
}

// Dial ...
func (e *Emq) Dial() {
	if err := e.dial(); err != nil {
		log.Println(err)
		return
	}
	log.Println("EMQ connected")
	e.start()
}

// EmqSuperAuth ...
func (e *Emq) EmqSuperAuth() (username, password string) {
	cfg := e.Config.V()
	sha256P := sha256.Sum256([]byte(cfg.GetString("emq.super_password")))
	return cfg.GetString("emq.super_username"), hex.EncodeToString(sha256P[:])
}

// PublishC ... publish in channel
func (e *Emq) PublishC(topic string, payload interface{}) {
	select {
	case e.pubC <- func() {
		if err := e.publish(topic, payload); err != nil {
			e.Logger.Errorf("[EMQ publish] [E] topic %s error: %s", topic, err.Error())
		}
	}:
	case <-time.After(time.Millisecond * 100):
		e.Logger.Errorf("[EMQ publish] [E] publish topic %s timeout", topic)
	}
}

// Subscribe ...
func (e *Emq) Subscribe(topic string, callback mqtt.MessageHandler) (err error) {
	topic = e.topicPrefix + topic
	t := e.client.Subscribe(topic, e.qos, callback)
	if !t.WaitTimeout(e.wait) {
		return ErrTokenTimeout
	}
	return t.Error()
}

// configBroker ......
func (e *Emq) configBroker() string {
	port := "1883"
	v := e.Config.V()
	if p := v.GetString("emq.port"); p != "" {
		port = p
	}
	host := v.GetString("emq.host")
	if host == "" {
		host = v.GetString("app.host")
	}
	return fmt.Sprintf("%s:%s", host, port)
}

// dial ...
func (e *Emq) dial() (err error) {
	e.prepareOption()
	e.client = mqtt.NewClient(e.opt)
	tk := e.client.Connect()
	if !tk.WaitTimeout(e.wait) {
		return ErrTokenTimeout
	}
	return tk.Error()
}

// prepareOption ...
func (e *Emq) prepareOption() {
	//mqtt.DEBUG = log.New(os.Stdout, "[mqtt]", log.LstdFlags)
	llg := newInnerLogger(e.Logger)
	mqtt.WARN = llg.WarnLogger()
	mqtt.CRITICAL = llg.CriticalLogger()
	mqtt.ERROR = llg.ErrorLogger()

	o1 := mqtt.NewClientOptions()
	o1.AddBroker(e.configBroker())
	o1.SetOrderMatters(false)
	o1.SetStore(mqtt.NewMemoryStore())
	o1.SetConnectRetry(true)
	o1.SetAutoReconnect(true)
	o1.SetMaxReconnectInterval(time.Second * 10)
	o1.SetKeepAlive(60 * time.Second)
	o1.SetPingTimeout(2 * time.Second)
	o1.SetConnectTimeout(3 * time.Second)
	o1.SetClientID(fmt.Sprintf("%s:SYS_%s", config.C.AppName(), util.ShortUUID()))
	username, password := e.EmqSuperAuth()
	o1.SetUsername(username)
	o1.SetPassword(password)
	o1.SetOnConnectHandler(func(client mqtt.Client) {
		// client connected
	})
	e.opt = o1
}

func (e *Emq) publish(topic string, payload interface{}) (err error) {
	var buf bytes.Buffer
	switch v := payload.(type) {
	case []byte:
		buf.Write(v)
	case string:
		buf.WriteString(v)
	default:
		buf.WriteString(util.JSONDump(v))
	}
	topic = e.topicPrefix + topic
	t := e.client.Publish(topic, e.qos, e.retained, buf.Bytes())
	if !t.WaitTimeout(e.wait) {
		return ErrTokenTimeout
	}
	return t.Error()
}

// start ...
func (e *Emq) start() {
	go func() {
		for {
			f := <-e.pubC
			f()
		}
	}()
}
