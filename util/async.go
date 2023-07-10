package util

import (
	"sync"
)

func ConcurrentRun(fn func(), num int) {
	if num <= 0 {
		num = 1
	}
	var w sync.WaitGroup
	w.Add(num)
	for i := 0; i < num; i++ {
		go func() {
			fn()
			w.Done()
		}()
	}
	w.Wait()
}
