package log

import (
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	_globalMu        = sync.RWMutex{}
	_globalS  Logger = zap.NewNop().Sugar()
)

type Logger interface {
	Debug(v ...any)
	Info(v ...any)
	Warn(v ...any)
	Error(v ...any)
	Panic(v ...any)
	Fatal(v ...any)

	Debugf(format string, v ...any)
	Infof(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
	Panicf(format string, v ...any)
	Fatalf(format string, v ...any)

	Sync() error
}

func Debug(v ...any) { _globalS.Debug(v...) }
func Info(v ...any)  { _globalS.Info(v...) }
func Warn(v ...any)  { _globalS.Warn(v...) }
func Error(v ...any) { _globalS.Error(v...) }
func Panic(v ...any) { _globalS.Panic(v...) }
func Fatal(v ...any) { _globalS.Fatal(v...) }

func Debugf(format string, v ...any) { _globalS.Debugf(format, v...) }
func Infof(format string, v ...any)  { _globalS.Infof(format, v...) }
func Warnf(format string, v ...any)  { _globalS.Warnf(format, v...) }
func Errorf(format string, v ...any) { _globalS.Errorf(format, v...) }
func Panicf(format string, v ...any) { _globalS.Panicf(format, v...) }
func Fatalf(format string, v ...any) { _globalS.Fatalf(format, v...) }

func Sync() error { return _globalS.Sync() }

type FileConfig struct {
	Filepath   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}
type Config struct {
	Level   string
	Console bool

	File FileConfig
}

type lumberjackSink struct{ *lumberjack.Logger }

func (lumberjackSink) Sync() error { return nil }

func ReplaceGlobals(l Logger) func() {
	_globalMu.Lock()
	prev := _globalS
	_globalS = l
	_globalMu.Unlock()

	return func() { ReplaceGlobals(prev) }
}

func NewLogger(cfg Config) (Logger, error) {
	level, err := zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "ts",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",

		LineEnding:  zapcore.DefaultLineEnding,
		EncodeLevel: zapcore.CapitalLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			const layout = "2006/01/02 15:04:05.000"

			type appendTimeEncoder interface {
				AppendTimeLayout(time.Time, string)
			}

			if enc, ok := enc.(appendTimeEncoder); ok {
				enc.AppendTimeLayout(t, layout)

				return
			}

			enc.AppendString(t.Format(layout))
		},
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	outputPaths := make([]string, 0)
	errorOutputPaths := make([]string, 0)

	if cfg.File.Filepath != "" {
		ll := lumberjack.Logger{
			Filename:   cfg.File.Filepath,
			MaxSize:    cfg.File.MaxSize,
			MaxBackups: cfg.File.MaxBackups,
			MaxAge:     cfg.File.MaxAge,
			Compress:   cfg.File.Compress,

			LocalTime: true,
		}

		zap.RegisterSink("lumberjack", func(u *url.URL) (zap.Sink, error) { return lumberjackSink{Logger: &ll}, nil })

		outputPaths = append(outputPaths, fmt.Sprintf("lumberjack:%s", cfg.File.Filepath))
		errorOutputPaths = append(errorOutputPaths, fmt.Sprintf("lumberjack:%s", cfg.File.Filepath))
	}

	if cfg.Console {
		outputPaths = append(outputPaths, os.Stdout.Name())
		errorOutputPaths = append(errorOutputPaths, os.Stderr.Name())
	}

	if len(outputPaths) == 0 {
		outputPaths = []string{os.DevNull}
	}
	if len(errorOutputPaths) == 0 {
		errorOutputPaths = []string{os.DevNull}
	}

	zapConfig := zap.Config{
		Level:            level,
		Development:      false,
		Encoding:         "console", // TODO custom encoder
		EncoderConfig:    encoderConfig,
		OutputPaths:      outputPaths,
		ErrorOutputPaths: errorOutputPaths,
	}

	logger, err := zapConfig.Build(zap.AddCallerSkip(0), zap.AddStacktrace(zapcore.PanicLevel))
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}
