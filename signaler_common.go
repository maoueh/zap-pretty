package zapp

import (
	"log"
)

type signaler struct {
	// Common
	debugEnabled bool
	debugLogger  *log.Logger

	// Use only on Darwin and Linux
	processGroupID int
}

func (s *signaler) debugPrintln(msg string, args ...interface{}) {
	if s.debugEnabled {
		s.debugLogger.Printf(msg+"\n", args...)
	}
}
