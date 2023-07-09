package utils

import (
	"fmt"
)

var logs chan string = make(chan string, 100)

func Debugf(format string, args ...any) {
	if GetDebug() {
		logs <- fmt.Sprintf(format, args...)
	}
}

func FlushLogs() []string {
	flushed := make([]string, 0)

	for {
		select {
		case log, more := <-logs:
			if !more {
				return flushed
			}
			flushed = append(flushed, log)
		default:
			return flushed
		}
	}
}
