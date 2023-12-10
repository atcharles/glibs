package j2rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andeya/goutil"
)

const (
	vsn = "2.0"

	splitMethodSeparator = "."
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

type (
	server struct {
		mutex          sync.Mutex
		opt            *Option
		run            int32
		services       map[string]service
		logger         LevelLogger
		excludeMethods []string
		midHandlers    Handlers
		contextPool    *ContextPool
		callBefore     BeforeHandler
		callAfter      AfterHandler
	}
	service struct {
		name      string
		receiver  interface{}
		callbacks map[string]callback
	}
)

// GetContext ......
func (s *server) GetContext() *Context {
	return s.contextPool.Acquire()
}

// Handler ...
func (s *server) Handler(ctx *Context) {
	defer func() { ctx.Release() }()
	// Don't serve if server is stopped.
	if atomic.LoadInt32(&s.run) == 0 {
		return
	}
	w, r := ctx.Writer(), ctx.Request()
	//检测是否已经写入header
	if HasWriteHeader(w) {
		return
	}
	if code, err := validateRequest(r); err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	// Prevents Internet Explorer from MIME-sniffing a response away
	// from the declared content-type
	w.Header().Set("x-content-type-options", "nosniff")
	if err := s.handle(ctx); err != nil {
		ctx.Msg().setError(err)
	}
	ctx.Msg().output().writeResponse(w, s.Opt())
	requestID := w.Header().Get("request-id")
	if len(requestID) > 0 {
		s.logger.Debugf("[Request-ID:%s] %s", requestID, JSONDump(ctx.Msg()))
	}
}

func (s *server) Logger() LevelLogger {
	if s.logger == nil {
		s.logger = NewLevelLogger("[STDOUT]")
	}
	return s.logger
}

func (s *server) NamespaceService(method string) interface{} {
	elem := strings.SplitN(method, splitMethodSeparator, 2)
	svs, ok := s.services[elem[0]]
	if !ok {
		return nil
	}
	return svs.receiver
}

// Opt ...
func (s *server) Opt() *Option { return s.opt }

// Register ...
func (s *server) Register(receiver interface{}, names ...string) {
	var _fnGetServiceName = func(rv interface{}) string {
		rvv := ValueIndirect(reflect.ValueOf(rv))
		var name string
		if len(names) > 0 && len(names[0]) > 0 {
			name = names[0]
		}
		if nsn1, ok := rv.(ItfNamespaceName); ok {
			name = nsn1.J2rpcNamespaceName()
		}
		if len(name) == 0 {
			name = s.formatName(rvv.Type().Name())
		}
		return name
	}
	serviceName := _fnGetServiceName(receiver)

	if consVal, ok := receiver.(ItfConstructor); ok {
		consVal.Constructor()
	}

	callbacks := s.suitableCallbacks(receiver)
	if len(callbacks) == 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.services == nil {
		s.services = make(map[string]service)
	}

	srv, ok := s.services[serviceName]
	if ok {
		panic(fmt.Sprintf("namespace [%s] exists", serviceName))
	}
	srv = service{
		name:      serviceName,
		receiver:  receiver,
		callbacks: make(map[string]callback),
	}
	s.services[serviceName] = srv
	for name, cb := range callbacks {
		srv.callbacks[name] = cb
	}
}

// RegisterForApp ...
func (s *server) RegisterForApp(app interface{}) {
	if _app, ok := app.(ItfExcludeMethod); ok {
		s.excludeMethods = _app.ExcludeMethod()
	}
	if _app, ok := app.(ItfStartup); ok {
		_app.Startup(s)
	}
	namespaces := ObjectTagInstances(app, "j2rpc")
	for _, namespace := range namespaces {
		s.Register(namespace)
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Permit dumb empty requests for remote health-checks (AWS)
	if r.Method == http.MethodGet && r.ContentLength == 0 && r.URL.RawQuery == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	s.Handler(s.GetContext().SetContext(r.Context()).BeginRequest(w, r))
}

func (s *server) SetCallAfter(callAfter AfterHandler) {
	s.callAfter = callAfter
}

func (s *server) SetCallBefore(callBefore BeforeHandler) {
	s.callBefore = callBefore
}

func (s *server) SetLogger(logger LevelLogger) { s.logger = logger }

// Stop stops reading new requests, waits for stopPendingRequestTimeout to allow pending
// requests to finish, then closes all codecs which will cancel pending requests and
// subscriptions.
func (s *server) Stop() {
	if atomic.CompareAndSwapInt32(&s.run, 1, 0) {
		s.logger.Debugf("RPC server shutting down")
	}
}

// Use ......
func (s *server) Use(handlers ...Handler) {
	if len(handlers) == 0 {
		return
	}
	s.midHandlers = UpsertHandlers(s.midHandlers, handlers)
}

// formatName ...
func (s *server) formatName(name string) string {
	if s.opt.SnakeNamespace {
		name = SnakeString(name)
	}
	return name
}

// getCallBack ...
func (s *server) getCallBack(elem []string) (cbk callback, err error) {
	svs, ok := s.services[elem[0]]
	if !ok {
		err = NewError(ErrNoMethod, "no namespace")
		return
	}
	cbk, ok = svs.callbacks[elem[1]]
	if !ok {
		err = NewError(ErrNoMethod, "no method")
		return
	}
	return
}

// handle ...
func (s *server) handle(c *Context) (err error) {
	defer func() { _ = c.Request().Body.Close() }()
	// read request body
	var body []byte
	if c.Request().Body != nil {
		var e error
		body, e = io.ReadAll(c.Request().Body)
		if e != nil && e != io.EOF {
			err = e
			return
		}
	}
	//重新设置body
	//c.Request().Body = io.NopCloser(bytes.NewReader(body))
	body, err = s.Opt().ParseRPCBody(body, true)
	if err != nil {
		return
	}
	c.Body = make([]byte, len(body))
	copy(c.Body, body)
	c.Store.Store(StartTimeKey, time.Now().UnixNano())

	msg := c.Msg()
	err = json.NewDecoder(bytes.NewReader(body)).Decode(msg)
	if err != nil {
		err = NewError(ErrParse, "请求错误")
		return
	}
	if !msg.hasValidID() {
		err = NewError(ErrInvalidRequest, "id is invalid")
		return
	}
	elem, err := msg.methods()
	if err != nil {
		return
	}
	for i, e2 := range elem {
		//elem[i] = s.formatName(CamelString(e2))
		elem[i] = s.formatName(e2)
	}
	msg.Method = strings.Join(elem, splitMethodSeparator)
	cbk, err := s.getCallBack(elem)
	if err != nil {
		return
	}
	cbk.providerMethod = msg.Method
	// context do middleware
	if err = c.Do(s.midHandlers...); err != nil {
		return
	}
	if c.IsStopped() {
		return
	}
	// call method
	callArgs, err := ParsePositionalArguments(msg.Params, cbk.argTypes)
	if err != nil {
		err = NewError(ErrBadParams, err.Error())
		return
	}
	res, err := cbk.call(c, callArgs)
	if err != nil {
		return
	}
	val := reflect.ValueOf(res)
	if !val.IsValid() {
		return
	}
	if val.IsZero() {
		return
	}
	answer, err := json.Marshal(res)
	if err != nil {
		return
	}
	msg.Result = answer
	return
}

// stack ...
func (s *server) stack(recover interface{}, methodName string) error {
	msg := fmt.Sprintf("%s: %v", methodName, recover)

	/*const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]*/

	buf := goutil.PanicTrace(4)
	s.logger.Errorf("RPC handler crashed:\n%s\n%s", msg, string(buf))
	return NewError(ErrInternal, msg)
}

// suitableCallbacks ...
func (s *server) suitableCallbacks(receiver interface{}) (callbacks map[string]callback) {
	callbacks = make(map[string]callback)

	var skipMethods = append(
		[]string{"Constructor", "ExcludeMethod", "BeforeCall", "AfterCall"},
		s.excludeMethods...,
	)
	if exv, ok := receiver.(ItfExcludeMethod); ok {
		skipMethods = append(skipMethods, exv.ExcludeMethod()...)
	}
	var _fn1InSkips = func(m1 string) bool {
		for _, method := range skipMethods {
			if m1 == method {
				return true
			}
		}
		return false
	}

	var _fnAppendMethods = func(method reflect.Method) {
		if method.PkgPath != "" {
			return
		}

		if _fn1InSkips(method.Name) {
			return
		}

		c := callback{
			server:         s,
			providerMethod: "",
			methodName:     method.Name,
			rcv:            reflect.ValueOf(receiver),
			fn:             method.Func,
			argTypes:       nil,
			hasCtx:         false,
			errPos:         -1,
		}
		if ok := c.makeArgTypes(); !ok {
			return
		}
		callbacks[s.formatName(method.Name)] = c
	}

	rp := reflect.TypeOf(receiver)
	for i := 0; i < rp.NumMethod(); i++ {
		_fnAppendMethods(rp.Method(i))
	}
	return
}

// New ...
func New(opts ...*Option) RPCServer {
	s := &server{run: 1}
	if len(opts) > 0 && opts[0] != nil {
		s.opt = opts[0]
	}
	if s.opt == nil {
		s.opt = SnakeOption
	}
	if s.opt.Service != nil {
		s.RegisterForApp(s.opt.Service)
	}
	s.logger = s.opt.Logger
	s.contextPool = newContextPool(func() interface{} { return newContext(s) })
	return s
}
