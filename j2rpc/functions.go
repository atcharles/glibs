package j2rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"unsafe"
)

//------------------------------ [segmentation] --------------------------------

// 5MB
const maxRequestContentLength = 1024 * 1024 * 5

var _ = acceptedContentTypes

// https://www.jsonrpc.org/historical/json-rpc-over-http.html#id13
var acceptedContentTypes = []string{
	"application/json",
	"application/json-rpc",
	"application/jsonrequest",
	"html/text",
	"text/plain",
}

// BytesToString convert []byte type to string type.
func BytesToString(b []byte) string { return *(*string)(unsafe.Pointer(&b)) }

// CamelString converts the accepted string to a camel string (xx_yy to XxYy)
func CamelString(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if !k && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || !k) {
			d = d - 32
			j = false
			k = true
		}
		if k && d == '_' && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return BytesToString(data[:])
}

func HasWriteHeader(w http.ResponseWriter) bool {
	defer w.Header().Del("Status-Written")
	return w.Header().Get("Status-Written") == "1"
}

// JSONDump ...
func JSONDump(val interface{}, args ...interface{}) string {
	var indent bool
	if len(args) > 0 {
		indent, _ = args[0].(bool)
	}
	if indent {
		return string(JsMarshalIndent(val))
	}
	return string(JsMarshal(val))
}

// JSONMarshalToBase64 ...
func JSONMarshalToBase64(val interface{}) ([]byte, error) {
	bts, err := json.Marshal(val)
	if err != nil {
		return bts, err
	}
	enc := base64.StdEncoding
	buf := make([]byte, enc.EncodedLen(len(bts)))
	enc.Encode(buf, bts)
	return buf, err
}

// JSONUnmarshalFromBase64 ...
func JSONUnmarshalFromBase64(data []byte, val interface{}) error {
	enc := base64.StdEncoding
	buf := make([]byte, enc.DecodedLen(len(data)))
	n, err := enc.Decode(buf, data)
	if err != nil {
		return err
	}
	bts := buf[:n]
	return json.Unmarshal(bts, val)
}

// JsMarshal ...
func JsMarshal(val interface{}) (bts []byte) { bts, _ = json.Marshal(val); return }

// JsMarshalIndent ...
func JsMarshalIndent(val interface{}) (bts []byte) {
	bts, _ = json.MarshalIndent(val, "", "\t")
	return
}

// ParsePositionalArguments tries to parse the given args to an array of values with the
// given types. It returns the parsed values or an error when the args could not be
// parsed. Missing optional arguments are returned as reflect.Zero|reflect.New values.
func ParsePositionalArguments(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	var args []reflect.Value
	tok, err := dec.Token()
	switch {
	case err == io.EOF || (err == nil && tok == nil):
		// "params" is optional and may be empty. Also allow "params":null even though it's
		// not in the spec because our own client used to send it.
	case err != nil:
		return nil, err
	case tok == json.Delim('['):
		// Read argument array.
		if args, err = parseArgumentArray(dec, types); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("non-array args")
	}
	// Set any missing args to nil.
	for i := len(args); i < len(types); i++ {
		ctp := types[i]
		addVal := reflect.Zero(ctp)
		if ctp.Kind() == reflect.Ptr {
			//addVal = reflect.New(ctp).Elem()
			addVal = reflect.New(ctp.Elem())
		}
		args = append(args, addVal)
	}
	return args, nil
}

// SetWriteHeader ...
func SetWriteHeader(w http.ResponseWriter) {
	w.Header().Set("Status-Written", "1")
}

func SliceStringContainsPrefix(slice []string, val string) bool {
	for _, item := range slice {
		if strings.HasPrefix(item, val) {
			return true
		}
	}
	return false
}

// SnakeString converts the accepted string to a snake string (XxYy to xx_yy)
func SnakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	for _, d := range StringToBytes(s) {
		func() {
			if d >= 'A' && d <= 'Z' {
				if j {
					data = append(data, '_')
					j = false
				}
				return
			}
			if d != '_' {
				j = true
			}
		}()
		data = append(data, d)
	}
	return strings.ToLower(BytesToString(data))
}

// StringToBytes convert string type to []byte type.
// NOTE: panic if modify the member value of the []byte.
func StringToBytes(s string) []byte {
	sp := *(*[2]uintptr)(unsafe.Pointer(&s))
	bp := [3]uintptr{sp[0], sp[1], sp[1]}
	return *(*[]byte)(unsafe.Pointer(&bp))
}

func isErrorType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Implements(errorType)
}

func parseArgumentArray(dec *json.Decoder, types []reflect.Type) ([]reflect.Value, error) {
	args := make([]reflect.Value, 0, len(types))
	for i := 0; dec.More(); i++ {
		if i >= len(types) {
			return args, fmt.Errorf("too many arguments, want at most %d", len(types))
		}
		agv := reflect.New(types[i])
		if err := dec.Decode(agv.Interface()); err != nil {
			return args, fmt.Errorf("invalid argument %d: %s", i, err.Error())
		}
		if agv.IsNil() && types[i].Kind() != reflect.Ptr {
			return args, fmt.Errorf("missing value for required argument %d", i)
		}
		args = append(args, agv.Elem())
	}
	// Read end of args array.
	_, err := dec.Token()
	return args, err
}

// validateRequest returns a non-zero response code and error message if the
// request is invalid.
func validateRequest(r *http.Request) (int, error) {
	if r.Method == http.MethodPut || r.Method == http.MethodDelete {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	if r.ContentLength > maxRequestContentLength {
		err := fmt.Errorf("content length too large (%d>%d)", r.ContentLength, maxRequestContentLength)
		return http.StatusRequestEntityTooLarge, err
	}
	// Allow OPTIONS (regardless of content-type)
	if r.Method == http.MethodOptions {
		return 0, nil
	}
	return 0, nil
	// Check content-type
	//if mt, _, err := mime.ParseMediaType(r.Header.Get("content-type")); err == nil {
	//	if SliceStringContainsPrefix(acceptedContentTypes, mt) {
	//		return 0, nil
	//	}
	//}
	//// Invalid content-type
	//err := fmt.Errorf("invalid content type")
	//return http.StatusUnsupportedMediaType, err
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	SetWriteHeader(w)
	_, _ = fmt.Fprint(w, msg)
}
