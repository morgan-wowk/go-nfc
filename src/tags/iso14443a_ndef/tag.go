package iso14443a_ndef

import (
	"context"
	"fmt"

	"github.com/ebfe/scard"
	"github.com/morgan-wowk/nfc/logger"
)

const (
	successResponse   = 0x90
	userDataFirstPage = 0x04
	bytesPerPage      = 4
)

type Tag struct {
	*scard.Card
}

// ReadUID reads the UID off of the card
func (c Tag) ReadUID() (string, error) {
	resp, err := c.Transmit([]byte{0xFF, 0xCA, 0x00, 0x00, 0x00})
	if err != nil {
		return "", fmt.Errorf("error transmitting command: %s", err.Error())
	}

	if resp[len(resp)-2] != successResponse {
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

func (c Tag) ReadPage(ctx context.Context, pageByte byte) ([]byte, error) {
	resp, err := c.Transmit([]byte{0xFF, 0xB0, 0x00, 0x01, 0x10})
	if err != nil {
		return nil, fmt.Errorf("error transmitting command: %s", err.Error())
	}

	/**
	Knowledge bits:
	- NTAG215 and most ISO-14443 compliant tags have 4 byte pages
	- NXP NTAG215 datasheet: https://www.nxp.com/docs/en/data-sheet/NTAG213_215_216.pdf
	- NDEF record layout: https://freemindtronic.com/wp-content/uploads/2022/02/NFC-Data-Exchange-Format-NDEF.pdf
	*/
	if len(resp) >= bytesPerPage+2 && resp[len(resp)-2] == successResponse && resp[len(resp)-1] == 0x00 {
		data := resp[:bytesPerPage]
		logger.FromContext(ctx).Infof("Binary: %08b", data)
		// TODO: Interpret NDEF flags
	}

	return resp, nil
}
