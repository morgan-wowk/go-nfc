package main

import (
	"fmt"

	"github.com/ebfe/scard"
)

const (
	transmitResponseStatusSuccessful = 0x90
)

type nfcTag struct {
	*scard.Card
}

// readUID reads the UID off of the card
func (c nfcTag) readUID() (string, error) {
	resp, err := c.Transmit([]byte{0xFF, 0xCA, 0x00, 0x00, 0x00})
	if err != nil {
		return "", fmt.Errorf("error transmitting command: %s", err.Error())
	}

	if resp[len(resp)-2] != transmitResponseStatusSuccessful {
		return "", fmt.Errorf("expected successful response reading UID, received: %+v", resp)
	}

	var uid string
	for i, b := range resp[:len(resp)-2] {
		if i > 0 {
			uid = fmt.Sprintf("%s:%02X", uid, b)
		} else {
			uid = fmt.Sprintf("%02X", b)
		}
	}

	return uid, nil
}
