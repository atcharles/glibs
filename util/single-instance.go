package util

import (
	"runtime"
	"sync"

	"github.com/spf13/cast"
)

type ItfSingleInstanceType[T any] interface {
	Load() (val T)
}

type singleInstanceType[T any] struct {
	once    sync.Once
	newFunc func() T
}

func (s *singleInstanceType[T]) Load() (val T) {
	s.once.Do(func() {
		val = s.newFunc()
	})
	return
}

func NewSingleInstanceType[T any](newFunc func() T) ItfSingleInstanceType[T] {
	return &singleInstanceType[T]{
		newFunc: newFunc,
	}
}

type singleInstance struct {
	m map[string]interface{}
	sync.RWMutex
}

// load the single instance
func (s *singleInstance) load(key string, newFunc func() interface{}) interface{} {
	s.Lock()
	if s.m == nil {
		s.m = make(map[string]interface{})
	}
	if v, ok := s.m[key]; ok {
		s.Unlock()
		return v
	}
	v := newFunc()
	s.m[key] = v
	s.Unlock()
	return v
}

// SingleInstance is a singleton instance
var localSingleInstance = &singleInstance{}

// LoadSingleInstance load the single instance
func LoadSingleInstance[T any](key string, newFunc func() T) T {
	return localSingleInstance.load(key, func() interface{} { return newFunc() }).(T)
}

func LoadSingle[T any](newFunc func() T) T {
	return LoadSingleInstance[T](getRuntimePosition(), newFunc)
}

func getRuntimePosition() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return ""
	}
	return file + ":" + cast.ToString(line)
}
