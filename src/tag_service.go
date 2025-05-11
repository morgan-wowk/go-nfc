package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/ebfe/scard"
	"github.com/morgan-wowk/nfc/logger"
	"github.com/morgan-wowk/nfc/tags/iso14443a_ndef"
)

type NFCTag interface {
	ReadUID() (string, error)
	ReadPage(ctx context.Context, pageByte byte) ([]byte, error)
}

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
	logger.FromContext(ctx).Info("Connecting to tag...")

	card, err := s.scard.Connect(device, scard.ShareShared, scard.ProtocolT1)
	if err != nil {
		return fmt.Errorf("error connecting to tag: %s", err.Error())
	}
	if card == nil {
		return errors.New("expected card after connecting to tag, got nil")
	}
	logger.FromContext(ctx).Info("Connected to tag. Dispatching contents to target...")

	var tag NFCTag
	// TODO: Implement technology detection when support for multiple reader and tag types are desired

	/**
	Tested reader and tag combinations:

	uTrust 3700 F
		\_ NTAG215 with NDEF message using ISO-14443 Type A implementation
	*/
	tag = iso14443a_ndef.Tag{Card: card}

	uid, err := tag.ReadUID()
	if err != nil {
		return fmt.Errorf("error reading UID: %s", err.Error())
	}
	logger.FromContext(ctx).Infof("uid from tag: %s", uid)

	resp, err := tag.ReadPage(ctx, 0x04)
	if err != nil {
		return fmt.Errorf("error reading page: %s", err.Error())
	}
	logger.FromContext(ctx).Infof("resp: %+v", resp)

	if err := card.Disconnect(scard.LeaveCard); err != nil {
		return fmt.Errorf("error disconnecting tag: %s", err.Error())
	}

	logger.FromContext(ctx).Info("Disconnected from tag")

	return nil
}
