package j2rpc

import (
	"encoding/json"
	"reflect"
	"runtime"
	"strings"
)

// FuncName ...
func FuncName(fn interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	sl := strings.Split(name, ".")
	return strings.Replace(sl[len(sl)-1], "-fm", "", -1)
}

func FuncNameFull(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

// JSONExchange ...使用json copy对象
func JSONExchange(dst interface{}, src interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(b, dst); err != nil {
		return err
	}
	return nil
}

// NewValue ...
func NewValue(bean interface{}) (val interface{}) {
	v := ValueIndirect(reflect.ValueOf(bean))
	return reflect.New(v.Type()).Interface()
}

//ObjectTagInstances ...
/**
 * @Description:根据标签获取字段实例集合
 * @param obj
 * @param tagName
 * @return []interface{}
 */
func ObjectTagInstances(obj interface{}, tagName string) []interface{} {
	data := make([]interface{}, 0)
	tv1 := ValueIndirect(reflect.ValueOf(obj))
	_f1append := func(vv reflect.Value, vf reflect.StructField) {
		_, has := vf.Tag.Lookup(tagName)
		if !has {
			return
		}
		if !(vv.CanSet() && vv.CanAddr() && vv.Kind() == reflect.Ptr) {
			return
		}
		if vv.IsNil() {
			vv.Set(reflect.New(vf.Type.Elem()))
		}
		data = append(data, vv.Interface())
	}
	for i := 0; i < tv1.NumField(); i++ {
		_f1append(tv1.Field(i), tv1.Type().Field(i))
	}
	return data
}

// ReflectValue ...回调函数
func ReflectValue(bean interface{}, fn func(v reflect.Value)) {
	v := reflect.ValueOf(bean)
	if v.Kind() != reflect.Ptr {
		return
	}
	v = ValueIndirect(v)
	fn(v)
}

// ReflectZeroFields 将多个字段设置为零值
func ReflectZeroFields(v interface{}, fields []string) {
	val := reflect.ValueOf(v)
	if val.IsZero() {
		return
	}
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	for _, field := range fields {
		f := val.FieldByName(field)
		if !f.IsValid() && !f.CanSet() {
			continue
		}
		f.Set(reflect.Zero(f.Type()))
	}
}

// SlicePointerValue ...构建类型切片
func SlicePointerValue(bean interface{}) reflect.Value {
	sv := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(bean)), 0, 0)
	sl := reflect.New(sv.Type())
	sl.Elem().Set(sv)
	return sl
}

// ValueIndirect ...值类型
func ValueIndirect(val reflect.Value) reflect.Value {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val
}
