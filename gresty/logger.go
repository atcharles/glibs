package gresty

type emptyLogger struct{}

func (e *emptyLogger) Errorf(string, ...interface{}) {}

func (e *emptyLogger) Warnf(string, ...interface{}) {}

func (e *emptyLogger) Debugf(string, ...interface{}) {}
