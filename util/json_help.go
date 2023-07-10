package util

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
)

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

// JsMarshal ...
func JsMarshal(val interface{}) (bts []byte) { bts, _ = json.Marshal(val); return }

// JsMarshalIndent ...
func JsMarshalIndent(val interface{}) (bts []byte) {
	bts, _ = json.MarshalIndent(val, "", "\t")
	return
}

// JSONUnmarshalFromBase64 ...
func JSONUnmarshalFromBase64(data []byte, val interface{}) error {
	enc := base64.StdEncoding
	dbuf := make([]byte, enc.DecodedLen(len(data)))
	n, err := enc.Decode(dbuf, data)
	if err != nil {
		return err
	}
	bts := dbuf[:n]
	return json.Unmarshal(bts, val)
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

func JsMarshalHex(val interface{}) (dst []byte, err error) {
	b, err := json.Marshal(val)
	if err != nil {
		return
	}
	dst = make([]byte, hex.EncodedLen(len(b)))
	hex.Encode(dst, b)
	return
}

func JsUnmarshalHex(data []byte, val interface{}) (err error) {
	b := make([]byte, len(data))
	copy(b, data)
	n, err := hex.Decode(b, b)
	if err != nil {
		return
	}
	return json.Unmarshal(b[:n], val)
}

type RawMessage = json.RawMessage
type Decoder = json.Decoder
type Encoder = json.Encoder

func NewDecoder(r io.Reader) *Decoder {
	return json.NewDecoder(r)
}

func NewEncoder(w io.Writer) *Encoder {
	return json.NewEncoder(w)
}

func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// MarshalToString ...
func MarshalToString(v interface{}, args ...interface{}) (string, error) {
	_fn := func() ([]byte, error) {
		if len(args) > 0 && args[0] == true {
			return MarshalIndent(v, "", "  ")
		}
		return Marshal(v)
	}
	data, err := _fn()
	return string(data), err
}

// UnmarshalFromString ...
func UnmarshalFromString(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}
