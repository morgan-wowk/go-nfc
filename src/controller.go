package main

import (
	"context"
	"fmt"
	"github.com/ebfe/scard"
	"strconv"
	"time"
)

type Controller struct {
	scard *scard.Context
	exit  context.CancelFunc
}

func NewController(scard *scard.Context, exitFunc context.CancelFunc) Controller {
	return Controller{
		scard: scard,
		exit:  exitFunc,
	}
}

func (c *Controller) Init(ctx context.Context) {
	c.SelectDevice(ctx)
}

func (c *Controller) SelectDevice(ctx context.Context) {
	logger(ctx).Infof("Checking for NFC devices...")
	waitingForDevice := false

	for {
		devices, err := c.scard.ListReaders()
		if err != nil {
			logger(ctx).Errorf("error listing devices: %s", err.Error())
			c.exit()
			break
		}

		if len(devices) == 0 {
			if !waitingForDevice {
				waitingForDevice = true
				logger(ctx).Error("No devices found. Waiting for device...")
			}
			time.Sleep(1 * time.Second)
			continue
		}

		fmt.Printf("\nSelect a device by entering the number next to it on the list:\n\n")
		for i, device := range devices {
			fmt.Printf("[%d] %s\n", i+1, device)
		}

		var deviceNumberInput string

		_, err = fmt.Scanln(&deviceNumberInput)
		if err != nil {
			logger(ctx).Errorf("error reading device index: %s", err.Error())
			continue
		}

		deviceIndex, err := strconv.Atoi(deviceNumberInput)
		if err != nil {
			logger(ctx).Errorf("error parsing entered device number: %s", err.Error())
			continue
		}

		if deviceIndex < 1 || deviceIndex > len(devices) {
			logger(ctx).Errorf("invalid device number: %v", deviceNumberInput)
			continue
		}

		device := devices[deviceIndex-1]

		logger(ctx).Infof("Selected device: %s", device)

		break
	}
}
