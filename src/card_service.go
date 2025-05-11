package main

import (
	"context"
	"errors"
	"fmt"

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
func (s cardService) dispatchContentsToTarget(ctx context.Context, device string) error {
	logger.FromContext(ctx).Info("Preparing to dispatch tag contents to target...")
	logger.FromContext(ctx).Info("Connecting to card...")

	card, err := s.scard.Connect(device, scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		return fmt.Errorf("error connecting to card: %s", err.Error())
	}
	if card == nil {
		return errors.New("expected card after connecting to card, got nil")
	}
	logger.FromContext(ctx).Info("Connected to tag. Dispatching contents to target...")

	tag := nfcTag{card}

	uid, err := tag.readUID()
	if err != nil {
		return fmt.Errorf("error reading UID: %s", err.Error())
	}

	logger.FromContext(ctx).Infof("uid from tag: %s", uid)

	if err := card.Disconnect(scard.LeaveCard); err != nil {
		return fmt.Errorf("error disconnecting card: %s", err.Error())
	}

	logger.FromContext(ctx).Info("Disconnected from card")

	return nil
}
