package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ebfe/scard"
	"strconv"
	"sync"
	"time"
)

type Controller struct {
	scard *scard.Context
}

func NewController(scard *scard.Context) Controller {
	return Controller{
		scard: scard,
	}
}

func (c *Controller) Init(ctx context.Context) {
	booted := false

	for {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}

		if booted {
			logger(ctx).Info("Rebooting NFC parser...")
		} else {
			logger(ctx).Info("Booting NFC parser...")
		}
		booted = true

		device, err := c.SelectDevice(ctx)
		if err != nil {
			logger(ctx).Errorf("Error selecting device: %s, rebooting in 3 seconds", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}

		select {
		case <-ctx.Done():
			return
		default:
			break
		}

		logger(ctx).Infof("Device selected: %s", device)

		if err := c.ScanCards(ctx, device); err != nil {
			logger(ctx).Errorf("Error scanning cards: %s", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}

		break
	}
}

// SelectDevice reads NFC devices and prompts the user to select a device
func (c *Controller) SelectDevice(ctx context.Context) (string, error) {
	logger(ctx).Infof("Checking for NFC devices...")
	waitingForDevice := false

	for {
		select {
		case <-ctx.Done():
			return "", nil
		default:
			break
		}

		devices, err := c.scard.ListReaders()
		if err != nil {
			return "", fmt.Errorf("error listing devices: %w", err)
		}

		if len(devices) == 0 {
			if !waitingForDevice {
				waitingForDevice = true
				logger(ctx).Info("No devices found. Waiting for device...")
			}
			time.Sleep(1 * time.Second)
			continue
		}

		fmt.Printf("\nSelect a device by entering the number next to it on the list:\n\n")
		for i, device := range devices {
			fmt.Printf("[%d] %s\n", i+1, device)
		}

		var deviceNumberInput string

		deviceNumberInputChan := make(chan string)
		deviceNumberInputErrChan := make(chan error)

		// Use go routine to gather user input to prevent blocking
		// shutdown when context is cancelled.
		go func() {
			var i string
			_, err = fmt.Scanln(&i)
			if err != nil {
				logger(ctx).Errorf("error reading device index: %s", err.Error())
				deviceNumberInputErrChan <- err
			}

			deviceNumberInputChan <- i
		}()

		select {
		case <-ctx.Done():
			return "", nil
		case deviceNumberInput = <-deviceNumberInputChan:
			break
		case err = <-deviceNumberInputErrChan:
			break
		}
		if err != nil {
			continue
		}

		deviceIndex, err := strconv.Atoi(deviceNumberInput)
		if err != nil {
			logger(ctx).Errorf("unable to parse device number: %s", err.Error())
			continue
		}

		if deviceIndex < 1 || deviceIndex > len(devices) {
			logger(ctx).Errorf("invalid device number: %v", deviceNumberInput)
			continue
		}

		device := devices[deviceIndex-1]

		return device, nil
	}
}

func (c *Controller) ScanCards(ctx context.Context, device string) (err error) {
	logger(ctx).Infof("Scanning for tags on device: %s...", device)

	errChan := make(chan error, 1)
	readerStateChan := make(chan scard.ReaderState)

	scanCtx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	wg := sync.WaitGroup{}

	wg.Add(1)

	go func(pctx context.Context, device string) {
		readerStates := []scard.ReaderState{
			{
				Reader:       device,
				CurrentState: scard.StateUnaware,
			},
		}

		for {
			select {
			case <-pctx.Done():
				break
			default:
				// This controls how long the scard library will be blocking until it detects
				// a change in state. It should not be too long in order to facilitate graceful
				// shutdown (e.g. when a kill signal is sent to the application).
				stateChangeTimeout := time.Second * 5

				// Check for change in reader state
				if err := c.scard.GetStatusChange(readerStates, stateChangeTimeout); err != nil {
					if !errors.Is(err, scard.ErrTimeout) {
						errChan <- err
						return
					}

					continue
				}

				// No error above means that the state has changed.
				// Pass the state change through the channel.
				readerStateChan <- readerStates[0]

				// Update the current state so the next state change can be detected
				readerStates[0].CurrentState = readerStates[0].EventState

				continue
			}

			break
		}

		wg.Done()
	}(scanCtx, device)

	for {
		select {
		case <-ctx.Done():
			break
		case cErr := <-errChan:
			err = cErr
			cancelCtx()
			break
		case state := <-readerStateChan:
			// React to change in reader state
			if state.EventState&scard.StatePresent != 0 {
				logger(ctx).Info("Tag detected")
			}

			if state.EventState&scard.StateEmpty != 0 {
				if state.CurrentState&scard.StatePresent != 0 {
					logger(ctx).Info("Tag removed from reader")
				}
			}

			continue
		default:
			time.Sleep(time.Millisecond * 10)
			continue
		}

		break
	}

	wg.Wait()

	return err
}
