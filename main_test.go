package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type logTest struct {
	name          string
	lines         []string
	expectedLines []string
}

func init() {
	debug = log.New(os.Stdout, "[pretty-test] ", 0)
}

func TestStandardNewProduction(t *testing.T) {
	runLogTests(t, []logTest{
		{
			name: "single_log_line",
			lines: []string{
				`{"level":"info","ts":1545445711.144533,"caller":"c","msg":"m"}`,
			},
			expectedLines: []string{
				// FIXME: Fixed when implementing zap.NewProduction settings
				``,
			},
		},
	})
}

func TestZapDriverNewProduction(t *testing.T) {
	runLogTests(t, []logTest{
		{
			name: "single_log_line",
			lines: []string{
				zapdriverLine("INFO", "2018-12-21T23:06:49.435919-05:00"),
			},
			expectedLines: []string{
				"[2018-12-21 23:06:12.435 EST] \x1b[32mINFO\x1b[0m \x1b[37m(c:0)\x1b[0m \x1b[34mm\x1b[0m {\"folder\":\"f\"}",
			},
		},
		{
			name: "single_non_log_line",
			lines: []string{
				"A non-JSON string line",
			},
			expectedLines: []string{
				"A non-JSON string line",
			},
		},
		{
			name: "multi_log_line",
			lines: []string{
				zapdriverLine("INFO", "2018-12-21T23:06:49.435919-05:00"),
				zapdriverLine("DEBUG", "2018-12-21T23:06:49.436920-05:00"),
			},
			expectedLines: []string{
				"[2018-12-21 23:06:12.435 EST] \x1b[32mINFO\x1b[0m \x1b[37m(c:0)\x1b[0m \x1b[34mm\x1b[0m {\"folder\":\"f\"}",
				"[2018-12-21 23:06:12.436 EST] \x1b[34mDEBUG\x1b[0m \x1b[37m(c:0)\x1b[0m \x1b[34mm\x1b[0m {\"folder\":\"f\"}",
			},
		},
		{
			name: "multi_mixed",
			lines: []string{
				zapdriverLine("INFO", "2018-12-21T23:06:49.435919-05:00"),
				"A non-JSON string line",
				zapdriverLine("DEBUG", "2018-12-21T23:06:49.436920-05:00"),
			},
			expectedLines: []string{
				"[2018-12-21 23:06:12.435 EST] \x1b[32mINFO\x1b[0m \x1b[37m(c:0)\x1b[0m \x1b[34mm\x1b[0m {\"folder\":\"f\"}",
				"A non-JSON string line",
				"[2018-12-21 23:06:12.436 EST] \x1b[34mDEBUG\x1b[0m \x1b[37m(c:0)\x1b[0m \x1b[34mm\x1b[0m {\"folder\":\"f\"}",
			},
		},
	})
}

func zapdriverLine(severity string, time string) string {
	return fmt.Sprintf(`{"severity":"%s","time":"%s","caller":"c:0","message":"m","folder":"f","labels":{},"logging.googleapis.com/sourceLocation":{"file":"f","line":"1","function":"fn"}}`, severity, time)
}

func runLogTests(t *testing.T, tests []logTest) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bytes.NewReader([]byte(strings.Join(test.lines, "\n")))
			writer := &bytes.Buffer{}

			processor := &processor{
				scanner: bufio.NewScanner(reader),
				output:  writer,
			}

			processor.process()

			outputLines := strings.Split(writer.String(), "\n")
			require.Equal(t, test.expectedLines, outputLines)
		})
	}
}
