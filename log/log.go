package log

type Logger interface {
	Debugf(format string, v ...any)
	Infof(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
	Panicf(format string, v ...any)
	Fatalf(format string, v ...any)
}

func Debugf(format string, v ...any) {}
func Infof(format string, v ...any)  {}
func Warnf(format string, v ...any)  {}
func Errorf(format string, v ...any) {}
func Panicf(format string, v ...any) {}
func Fatalf(format string, v ...any) {}
