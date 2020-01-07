//+build darwin linux

package main

import (
	"os"
	"os/signal"
	"syscall"
)

type signaler struct {
	processGroupID int
}

func NewSignaler() *signaler {
	signaler := &signaler{}

	pgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		signaler.processGroupID = pgid
	} else {
		debug.Println("[Warning] unable to determine process group, signaling will be broken")
	}

	return signaler
}

func (s *signaler) forwardAllSignalsToProcessGroup() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan)

	for {
		signal := <-signalChan

		syscallSignal, isSyscallSignalType := signal.(syscall.Signal)
		if s.processGroupID != 0 && isSyscallSignalType {
			syscall.Kill(s.processGroupID, syscallSignal)
		}
	}
}
