//go:build darwin || linux
// +build darwin linux

package zapp

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func NewSignaler(debugEnabled bool, debugLogger *log.Logger) *signaler {
	s := &signaler{
		debugEnabled: debugEnabled,
		debugLogger:  debugLogger,
	}

	pgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		s.processGroupID = pgid
	} else {
		s.debugPrintln("[Warning] unable to determine process group, signaling will be broken")
	}

	return s
}

func (s *signaler) ForwardAllSignalsToProcessGroup() {
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
