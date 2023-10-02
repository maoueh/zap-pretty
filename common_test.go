package zapp

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
)

func executeProcessorTest(lines []string, options ...ProcessorOption) *bytes.Buffer {
	reader := bytes.NewReader([]byte(strings.Join(lines, "\n")))
	writer := &bytes.Buffer{}

	processor := &Processor{
		scanner:                     bufio.NewScanner(reader),
		output:                      writer,
		multilineJSONFieldThreshold: 3,
	}

	if os.Getenv("DEBUG") != "" {
		options = append(options, WithDebugLogger(log.New(os.Stdout, "[pretty-test] ", 0)))
	}

	for _, opt := range options {
		opt.apply(processor)
	}

	processor.Process()
	return writer
}

func zapdriverLine(severity string, time string) string {
	return fmt.Sprintf(`{"severity":"%s","time":"%s","caller":"c:0","message":"m","folder":"f","labels":{},"logging.googleapis.com/sourceLocation":{"file":"f","line":"1","function":"fn"}}`, severity, time)
}
