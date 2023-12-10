package j2rpc

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/andeya/goutil"
)

// SnakeOption ...
var SnakeOption = &Option{SnakeNamespace: true}

type HandlerCrypto struct {
	Key       string
	UseBase64 bool
}

// Option ...
type Option struct {
	SnakeNamespace bool
	Service        interface{}
	Logger         LevelLogger
	//CryptoKey  request/response crypto key, default is nil (no crypto)
	CryptoKey string
}

func (o *Option) ParseRPCBody(body []byte, isDecrypt bool) (_ []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("parse request body error: %v", p)
			return
		}
	}()
	if len(body) == 0 {
		return body, nil
	}
	if len(o.CryptoKey) == 0 {
		return body, nil
	}
	co := ParseHandlerCrypto(o.CryptoKey)
	if co == nil {
		return body, nil
	}
	if len(co.Key) < 16 {
		return body, nil
	}
	var key []byte
	if len(co.Key) >= 16 && len(co.Key) < 24 {
		key = []byte(co.Key)[:16]
	}
	if len(co.Key) >= 24 && len(co.Key) < 32 {
		key = []byte(co.Key)[:24]
	}
	if len(co.Key) >= 32 {
		key = []byte(co.Key)[:32]
	}
	if isDecrypt {
		if !co.UseBase64 {
			body = bytes.ToLower(body)
		}
		return goutil.AESCBCDecrypt(key, body, co.UseBase64)
	}
	body = goutil.AESCBCEncrypt(key, body, co.UseBase64)
	if !co.UseBase64 {
		body = bytes.ToUpper(body)
	}
	return body, nil
}

// SetService ...
func (o *Option) SetService(s interface{}) *Option {
	o.Service = s
	return o
}

// ParseHandlerCrypto ... str format: base64:xxx or xxx
// if str is empty, return nil
// if str is not start with base64:, return &HandlerCrypto{Key: str}
// if str is start with base64:, return &HandlerCrypto{Key: str[7:], UseBase64: true}
// if str[7:] is empty, return nil
func ParseHandlerCrypto(str string) *HandlerCrypto {
	if len(str) == 0 {
		return nil
	}
	if !strings.HasPrefix(str, "base64:") {
		return &HandlerCrypto{Key: str}
	}
	str = str[7:]
	if len(str) == 0 {
		return nil
	}
	return &HandlerCrypto{Key: str, UseBase64: true}
}
