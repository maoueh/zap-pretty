package main

import (
	"golang.org/x/sys/windows"
)

type signaler struct {
}

func NewSignaler() *signaler {
	return &signaler{}
}

func (s *signaler) forwardAllSignalsToProcessGroup() {
	consoleCtrlEventChan := make(chan uint, 1)
	if err := handleConsoleCtrlEvent(consoleCtrlEventChan); err != nil {
		debug.Println("[Warning] unable to listen for console events")
		return
	}

	for {
		ctrlType := <-consoleCtrlEventChan

		// The MSDN documentation for this call states the following:
		//
		// > If this parameter is zero, the signal is generated in all processes
		//   that share the console of the calling process.
		//
		// In our case, it's exactly what we want to do, we want to forward to all process in
		// group, so hence the `0` hard-coded value.
		//
		// @see https://docs.microsoft.com/en-us/windows/console/generateconsolectrlevent
		windows.GenerateConsoleCtrlEvent(uint32(ctrlType), 0)
	}
}

// Code for windows console handler based on https://github.com/golang/go/issues/7479#issuecomment-457669779
func handleConsoleCtrlEvent(events chan<- uint) error {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	setConsoleCtrlHandler := kernel32.NewProc("SetConsoleCtrlHandler")
	callback := func(ctrlType uint) uint {
		events <- ctrlType
		return 0
	}


	n, _, err := setConsoleCtrlHandler.Call(windows.NewCallback(callback), 1)
	if n == 0 {
		return err
	}

	return nil
}
