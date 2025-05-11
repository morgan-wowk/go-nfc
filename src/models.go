package main

import (
	"time"

	"github.com/ebfe/scard"
)

type stateChange struct {
	newState   scard.ReaderState
	occurredAt time.Time
}
