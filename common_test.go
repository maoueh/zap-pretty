package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

func executeProcessorTest(lines []string, options ...processorOption) *bytes.Buffer {
	reader := bytes.NewReader([]byte(strings.Join(lines, "\n")))
	writer := &bytes.Buffer{}

	processor := &processor{
		scanner: bufio.NewScanner(reader),
		output:  writer,
	}

	for _, opt := range options {
		opt.apply(processor)
	}

	processor.process()
	return writer
}

func zapdriverLine(severity string, time string) string {
	return fmt.Sprintf(`{"severity":"%s","time":"%s","caller":"c:0","message":"m","folder":"f","labels":{},"logging.googleapis.com/sourceLocation":{"file":"f","line":"1","function":"fn"}}`, severity, time)
}
