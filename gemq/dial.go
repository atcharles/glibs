package gemq

import (
	"fmt"
	"time"

	"github.com/eclipse/paho.mqtt.golang"

	"github.com/atcharles/glibs/util"
)

type DialOptions struct {
	AppName  string
	Broker   string
	Username string
	Password string
}

func DailWithOption(opt DialOptions) (c mqtt.Client, err error) {
	o1 := mqtt.NewClientOptions()
	o1.AddBroker(opt.Broker)
	o1.SetOrderMatters(false)
	o1.SetStore(mqtt.NewMemoryStore())
	o1.SetConnectRetry(true)
	o1.SetAutoReconnect(true)
	o1.SetMaxReconnectInterval(time.Second * 10)
	o1.SetKeepAlive(60 * time.Second)
	o1.SetPingTimeout(2 * time.Second)
	o1.SetConnectTimeout(3 * time.Second)
	o1.SetClientID(fmt.Sprintf("%s:SYS_%s", opt.AppName, util.UUID16md5hex()))
	username, password := opt.Username, opt.Password
	o1.SetUsername(username)
	o1.SetPassword(password)
	o1.SetOnConnectHandler(func(client mqtt.Client) {
		// client connected
	})
	client := mqtt.NewClient(o1)
	tk := client.Connect()
	if !tk.WaitTimeout(3 * time.Second) {
		return nil, ErrTokenTimeout
	}
	if err = tk.Error(); err != nil {
		return nil, err
	}
	return client, nil
}
