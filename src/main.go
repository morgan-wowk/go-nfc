package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ebfe/scard"
)

func main() {
	ctx, err := registerLoggerInContext(context.Background())
	cctx, cancelCtx := context.WithCancel(ctx)

	sctx, err := scard.EstablishContext()
	if err != nil {
		logger(ctx).Errorf("error establishing connection to system: %s", err.Error())
		return
	}
	logger(ctx).Info("established scard context")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger(ctx).Warnf("received shutdown signal: %v", sig)
		cancelCtx()
	}()

	defer func() {
		if sctx != nil {
			logger(ctx).Info("releasing scard context...")
			if err := sctx.Release(); err != nil {
				logger(ctx).Errorf("error closing card context: %s", err.Error())
			}
			logger(ctx).Info("scard context released")
		}
		if err := releaseLogger(ctx); err != nil {
			logger(ctx).Warnf("error releasing logger: %s", err.Error())
		}
		if r := recover(); r != nil {
			if err != nil {
				err = fmt.Errorf("panic occurred after previous error: %+v, error: %s", r, err.Error())
			} else {
				err = fmt.Errorf("panic: %+v", r)
			}
		}
		if err != nil {
			fmt.Printf("program exiting with error: %s\n", err.Error())
			os.Exit(1)
		}
	}()

	controller := NewController(sctx, cancelCtx)
	controller.Init(ctx)

	<-cctx.Done()
}
