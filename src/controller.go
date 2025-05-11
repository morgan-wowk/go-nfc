package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ebfe/scard"
	"github.com/morgan-wowk/nfc/logger"
)

type controller struct {
	scard       *scard.Context
	cardService cardService
}

func newController(scard *scard.Context, cardSvc cardService) controller {
	return controller{
		scard:       scard,
		cardService: cardSvc,
	}
}

func (c controller) init(ctx context.Context) {
	booted := false

	for {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}

		if booted {
			logger.FromContext(ctx).Info("Rebooting NFC parser...")
		} else {
			logger.FromContext(ctx).Info("Booting NFC parser...")
		}
		booted = true

		device, err := c.selectDevice(ctx)
		if err != nil {
			logger.FromContext(ctx).Errorf("Error selecting device: %s, rebooting in 3 seconds", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}

		select {
		case <-ctx.Done():
			return
		default:
			break
		}

		logger.FromContext(ctx).Infof("Device selected: %s", device)
		ctx = logger.AttachArgsToLogger(ctx, "device", device)

		readerCtx := c.maintainReaderConnection(ctx, device)

		if err := c.scanTags(readerCtx, device); err != nil {
			logger.FromContext(ctx).Errorf("Error scanning cards: %s", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}

		select {
		case <-ctx.Done():
			break
		case <-readerCtx.Done():
		default:
		}
	}
}

// selectDevice reads NFC devices and prompts the user to select a device
func (c controller) selectDevice(ctx context.Context) (string, error) {
	logger.FromContext(ctx).Infof("Checking for NFC devices...")
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
				logger.FromContext(ctx).Info("No devices found. Waiting for device...")
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
				logger.FromContext(ctx).Errorf("Error reading device index: %s", err.Error())
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
			logger.FromContext(ctx).Errorf("Unable to parse device number: %s", err.Error())
			continue
		}

		if deviceIndex < 1 || deviceIndex > len(devices) {
			logger.FromContext(ctx).Errorf("Invalid device number: %v", deviceNumberInput)
			continue
		}

		device := devices[deviceIndex-1]

		return device, nil
	}
}

// scanTags listens for changes to the state of the reader to detect tag presence and removal
func (c controller) scanTags(ctx context.Context, device string) (err error) {
	logger.FromContext(ctx).Info("Scanning for tags...")

	errChan := make(chan error, 1)
	readerStateChan := make(chan stateChange)

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
			case <-ctx.Done():
				break
			default:
				// This controls how long the scard library will be blocking until it detects
				// a change in state. It should not be too long in order to facilitate graceful
				// shutdown (e.g. when a kill signal is sent to the application).
				stateChangeTimeout := time.Second * 1

				// Check for change in reader state
				if err := c.scard.GetStatusChange(readerStates, stateChangeTimeout); err != nil {
					if !errors.Is(err, scard.ErrTimeout) {
						errChan <- err
						break
					}

					continue
				}

				// No error above means that the state has changed.
				// Pass the state change through the channel.
				readerStateChan <- stateChange{
					newState:   readerStates[0],
					occurredAt: time.Now(),
				}

				// Update the current state so the next state change can be detected
				readerStates[0].CurrentState = readerStates[0].EventState

				continue
			}

			break
		}

		logger.FromContext(pctx).Info("Tag scanning shut down")
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
		case diff := <-readerStateChan:
			// React to change in reader state
			if diff.newState.EventState&scard.StatePresent != 0 && diff.newState.CurrentState&scard.StateEmpty != 0 {
				logger.FromContext(ctx).Info("Tag detected")
				if err := c.cardService.dispatchContentsToTarget(ctx, device); err != nil {
					logger.FromContext(ctx).Errorf("Error dispatching contents to target: %s", err.Error())
				}
			}

			if diff.newState.EventState&scard.StateEmpty != 0 && diff.newState.CurrentState&scard.StatePresent != 0 {
				logger.FromContext(ctx).Info("Tag removed from reader")
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

// maintainReaderConnection periodically checks that the reader is still available for communication
//
// Returns a context.Context representing a healthy reader connection.
//
// When a reader becomes unavailable, the returned context.Context is cancelled.
func (c controller) maintainReaderConnection(ctx context.Context, device string) context.Context {
	readerCtx, cancelReaderCtx := context.WithCancel(ctx)

	go func(pctx context.Context, device string) {
		for {
			select {
			case <-pctx.Done():
				return
			default:
				break
			}

			devices, err := c.scard.ListReaders()
			if err != nil {
				logger.FromContext(pctx).Errorf("Error maintaining reader connection: %s", err.Error())
			}

			found := false
			for _, d := range devices {
				if d == device {
					found = true
					break
				}
			}

			if !found {
				logger.FromContext(pctx).Error("Reader was disconnected")
				cancelReaderCtx()
				return
			}

			time.Sleep(time.Second * 5)
		}

	}(ctx, device)

	return readerCtx
}
