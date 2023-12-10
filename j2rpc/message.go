package j2rpc

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RPCMessage A value of this type can a JSON-RPC request, notification, successful response or
// error response. Which one it is depends on the fields.
type RPCMessage struct {
	ID      json.RawMessage `json:"id,omitempty"`
	Version string          `json:"jsonrpc,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// empty ......
func (r *RPCMessage) empty() {
	r.ID = nil
	r.Version = ""
	r.Method = ""
	r.Params = nil
	r.Result = nil
	r.Error = nil
}

func (r *RPCMessage) hasValidID() bool { return len(r.ID) > 0 && r.ID[0] != '{' && r.ID[0] != '[' }

// namespace ...returns the service's name
func (r *RPCMessage) methods() ([]string, error) {
	elem := strings.SplitN(r.Method, splitMethodSeparator, 2)
	if len(elem) != 2 {
		return nil, NewError(ErrNoMethod, "wrong method")
	}
	return elem, nil
}

// output ...
func (r *RPCMessage) output() *RPCMessage {
	if len(r.ID) == 0 {
		r.ID = []byte{'1'}
	}
	r.Version = vsn
	r.Method = ""
	r.Params = nil
	return r
}

// setError ...
func (r *RPCMessage) setError(err error) *RPCMessage {
	if err == nil {
		return r
	}
	var e = NewError(ErrInternal, err.Error())
	switch v := err.(type) {
	case ItfJ2rpcError:
		e.Code = ErrorCode(v.ErrorCode())
		e.Message = v.Error()
		e.Data = v.ErrorData()
	case IRPCError:
		e = v.RPCError()
	case TokenError:
		e.Code = ErrAuthorization
		e.Message = v.Error()
	case ForbiddenError:
		e.Code = ErrForbidden
		e.Message = v.Error()
	}
	r.Error = e
	return r
}

// writeResponse ...
func (r *RPCMessage) writeResponse(w http.ResponseWriter, opt *Option) {
	if HasWriteHeader(w) {
		return
	}
	bts, err := json.Marshal(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	bts, err = opt.ParseRPCBody(bts, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(opt.CryptoKey) >= 16 {
		w.Header().Set("Content-Type", "text/plain")
	}
	_, _ = w.Write(bts)
	//n, err := w.Write(bts)
	//_, _ = n, err
}

func RPCError(err error) *RPCMessage { return new(RPCMessage).setError(err) }

func RPCResult(result interface{}) *RPCMessage {
	r := new(RPCMessage)
	r.Result, _ = json.Marshal(result)
	return r.output()
}
