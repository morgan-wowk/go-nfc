package main

import (
	"context"

	"github.com/ebfe/scard"
	"github.com/morgan-wowk/nfc/logger"
)

type target interface {
	DispatchContents(data []byte) error
}

type cardService struct {
	scard  *scard.Context
	target target
}

// newCardService creates a new cardService
func newCardService(scard *scard.Context, t target) cardService {
	return cardService{
		scard:  scard,
		target: t,
	}
}

// dispatchContentsToTarget reads the contents of the card present on the reader and
// delivers it to the configured target on the service.
func (s cardService) dispatchContentsToTarget(ctx context.Context) error {
	logger.FromContext(ctx).Info("Dispatching contents of tag to target")
	return nil
}
