package logger

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	loggerContextKey = iota
	sugarLoggerContextKey
)

func RegisterLoggerInContext(ctx context.Context) (context.Context, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.DisableStacktrace = true

	l, err := config.Build()
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, loggerContextKey, l)
	ctx = context.WithValue(ctx, sugarLoggerContextKey, l.Sugar())

	return ctx, nil
}

func AttachArgsToLogger(ctx context.Context, args ...interface{}) context.Context {
	l := FromContext(ctx)
	updated := l.With(args...)

	return context.WithValue(ctx, sugarLoggerContextKey, updated)
}

func FromContext(ctx context.Context) *zap.SugaredLogger {
	s, ok := ctx.Value(sugarLoggerContextKey).(*zap.SugaredLogger)
	if !ok {
		panic("no sugared logger in context")
	}

	return s
}

func ReleaseLogger(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if err != nil {
				err = fmt.Errorf("panic releasing FromContext after previous error: %v, previous error: %w", r, err)
			} else {
				err = fmt.Errorf("panic releasing FromContext: %v", r)
			}
		}
	}()

	l, ok := ctx.Value(loggerContextKey).(*zap.Logger)
	if !ok {
		panic("no FromContext in context")
	}

	return l.Sync()
}
