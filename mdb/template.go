package mdb

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"text/template"
)

// ReflectIndirect ...
func ReflectIndirect(v interface{}) reflect.Value {
	rv := reflect.ValueOf(v)
	for {
		if rv.Kind() != reflect.Ptr {
			break
		}
		rv = rv.Elem()
	}
	return rv
}

type Map map[string]interface{}

// StructToMap ...
func StructToMap(v interface{}) Map {
	val, _ := json.Marshal(v)
	mVal := make(Map)
	_ = json.Unmarshal(val, &mVal)
	return mVal
}

var ErrInvalidType = errors.New("invalid type")

// TextTemplateMustParse ...
func TextTemplateMustParse(text string, data interface{}) (result string) {
	var err error
	defer func() {
		if err != nil {
			panic(err)
		}
	}()
	rv := ReflectIndirect(data)
	var val interface{}
	switch rv.Type().Kind() {
	case reflect.Map:
		val = data
	case reflect.Struct:
		val = StructToMap(data)
	default:
		err = ErrInvalidType
		return
	}
	tp, err := template.New("t").Parse(text)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err = tp.Execute(&buf, val); err != nil {
		return
	}
	return buf.String()
}
