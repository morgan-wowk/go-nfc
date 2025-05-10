package main

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

func registerLoggerInContext(ctx context.Context) (context.Context, error) {
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

func logger(ctx context.Context) *zap.SugaredLogger {
	s, ok := ctx.Value(sugarLoggerContextKey).(*zap.SugaredLogger)
	if !ok {
		panic("no sugared logger in context")
	}

	return s
}

func releaseLogger(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if err != nil {
				err = fmt.Errorf("panic releasing logger after previous error: %v, previous error: %w", r, err)
			} else {
				err = fmt.Errorf("panic releasing logger: %v", r)
			}
		}
	}()

	l, ok := ctx.Value(loggerContextKey).(*zap.Logger)
	if !ok {
		panic("no logger in context")
	}

	return l.Sync()
}
