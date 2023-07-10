package j2rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/andeya/goutil"
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

// writeResponse ...
func (r *RPCMessage) writeResponse(w http.ResponseWriter, cryptoKey string) {
	if HasWriteHeader(w) {
		return
	}
	bts, err := json.Marshal(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	func() {
		if len(cryptoKey) >= 16 {
			bts = goutil.AESCBCEncrypt([]byte(cryptoKey), bts)
			bts = bytes.ToUpper(bts)
			w.Header().Set("Content-Type", "text/plain")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}()
	_, _ = w.Write(bts)
	//n, err := w.Write(bts)
	//_, _ = n, err
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

func RPCError(err error) *RPCMessage { return new(RPCMessage).setError(err) }

func RPCResult(result interface{}) *RPCMessage {
	r := new(RPCMessage)
	r.Result, _ = json.Marshal(result)
	return r.output()
}

// setError ...
func (r *RPCMessage) setError(err error) *RPCMessage {
	if err == nil {
		return r
	}
	var e *Error
	switch _e := err.(type) {
	case *Error:
		e = _e
	case TokenError:
		e = NewError(ErrAuthorization, _e.Error())
	case ForbiddenError:
		e = NewError(ErrForbidden, _e.Error())
	case IRPCError:
		e = _e.RPCError()
	default:
		func() {
			var j2rpcError *Error
			if errors.As(err, &j2rpcError) {
				e = NewError(j2rpcError.Code, err.Error())
				return
			}
			e = NewError(ErrServer, err.Error())
		}()
	}
	r.Error = e
	return r
}

// namespace ...returns the service's name
func (r *RPCMessage) methods() ([]string, error) {
	elem := strings.SplitN(r.Method, splitMethodSeparator, 2)
	if len(elem) != 2 {
		return nil, NewError(ErrNoMethod, "wrong method")
	}
	return elem, nil
}

func (r *RPCMessage) hasValidID() bool { return len(r.ID) > 0 && r.ID[0] != '{' && r.ID[0] != '[' }
