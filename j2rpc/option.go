package j2rpc

// SnakeOption ...
var SnakeOption = &Option{SnakeNamespace: true}

// Option ...
type Option struct {
	SnakeNamespace bool
	Service        interface{}
	Logger         LevelLogger
	//CryptoKey  request/response crypto key, default is nil (no crypto)
	CryptoKey string
}

// SetService ...
func (o *Option) SetService(s interface{}) *Option {
	o.Service = s
	return o
}
