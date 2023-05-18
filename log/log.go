package log

import (
	"context"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

func init() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar = logger.Sugar()
}

func Info(_ context.Context, msg string, args ...interface{}) {
	sugar.Infow(msg, args...)
}

func Error(_ context.Context, msg string, args ...interface{}) {
	sugar.Errorw(msg, args...)
}

func Fatal(_ context.Context, msg string, args ...interface{}) {
	sugar.Fatalw(msg, args...)
}

func Warn(_ context.Context, msg string, args ...interface{}) {
	sugar.Warnw(msg, args...)
}
