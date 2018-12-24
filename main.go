package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	. "github.com/logrusorgru/aurora"
)

var debug = log.New(ioutil.Discard, "", 0)
var severityToColor map[string]Color

var errNonZapLine = errors.New("non-zap line")

func init() {
	if os.Getenv("ZAP_PRETTY_DEBUG") != "" {
		debug = log.New(os.Stderr, "[pretty-debug] ", 0)
	}

	severityToColor = make(map[string]Color)
	severityToColor["debug"] = BlueFg
	severityToColor["info"] = GreenFg
	severityToColor["warning"] = BrownFg
	severityToColor["error"] = RedFg
	severityToColor["dpanic"] = RedFg
	severityToColor["panic"] = RedFg
	severityToColor["fatal"] = RedFg
}

type processor struct {
	scanner *bufio.Scanner
	output  io.Writer
}

func main() {
	processor := &processor{
		scanner: bufio.NewScanner(os.Stdin),
		output:  os.Stdout,
	}

	processor.process()
}

func (p *processor) process() {
	first := true
	for p.scanner.Scan() {
		if !first {
			fmt.Fprintln(p.output)
		}

		p.processLine(p.scanner.Text())
		first = false
	}

	if err := p.scanner.Err(); err != nil {
		debug.Println("Scanner terminated with error", err)
	}
}

func (p *processor) processLine(line string) {
	debug.Println("Processing line", line)
	if !p.mightBeJson(line) {
		fmt.Fprint(p.output, line)
		return
	}

	var lineData map[string]interface{}
	err := json.Unmarshal([]byte(line), &lineData)
	if err != nil {
		fmt.Fprint(p.output, line)
		debug.Println(err)
		return
	}

	prettyLine, err := p.maybePrettyPrintLine(line, lineData)

	if err != nil {
		fmt.Fprint(p.output, line)

		switch err {
		case errNonZapLine:
		default:
			debug.Println(err)
		}
	} else {
		fmt.Fprint(p.output, prettyLine)
	}
}

func (p *processor) mightBeJson(line string) bool {
	// TODO: Improve optimization when some benchmarks are available
	return strings.Contains(line, "{")
}

func (p *processor) maybePrettyPrintLine(line string, lineData map[string]interface{}) (string, error) {
	if lineData["level"] != nil && lineData["ts"] != nil && lineData["caller"] != nil && lineData["msg"] != nil {
		return p.maybePrettyPrintZapLine(line, lineData)
	}

	if lineData["severity"] != nil && lineData["time"] != nil && lineData["caller"] != nil && lineData["message"] != nil {
		return p.maybePrettyPrintZapdriverLine(line, lineData)
	}

	return "", errNonZapLine
}

func (p *processor) maybePrettyPrintZapLine(line string, lineData map[string]interface{}) (string, error) {
	var buffer bytes.Buffer

	nanosSinceEpoch := lineData["ts"].(float64) * time.Second.Seconds()
	secondsPart, nanosPart := math.Modf(nanosSinceEpoch)
	parsedTime := time.Unix(int64(secondsPart), int64(nanosPart/time.Nanosecond.Seconds()))

	buffer.WriteString(fmt.Sprintf("[%s]", parsedTime.Format("2006-01-02 15:04:05.000 MST")))

	buffer.WriteByte(' ')
	buffer.WriteString(p.colorizeSeverity(lineData["level"].(string)).String())

	buffer.WriteByte(' ')
	buffer.WriteString(Gray(fmt.Sprintf("(%s)", lineData["caller"].(string))).String())

	buffer.WriteByte(' ')
	buffer.WriteString(Blue(lineData["msg"].(string)).String())

	// Standard stuff
	delete(lineData, "level")
	delete(lineData, "ts")
	delete(lineData, "caller")
	delete(lineData, "message")

	if len(lineData) > 0 {
		// FIXME: This is poor, we would like to print in a single line stuff that are not too
		//        big. But what represents a too big value exactly? We would need to serialize to
		//        JSON, check lenght, if smaller than threshold, print with space, otherwise
		//        re-serialize with pretty-printing stuff
		var jsonBytes []byte
		var err error

		if len(lineData) <= 2 {
			jsonBytes, err = json.Marshal(lineData)
		} else {
			jsonBytes, err = json.MarshalIndent(lineData, "", "  ")
		}

		if err != nil {
			// FIXME: We could print each line as raw text maybe when it's not working?
			debug.Println(err)
		} else {
			buffer.WriteByte(' ')
			buffer.Write(jsonBytes)
		}
	}

	return buffer.String(), nil
}

func (p *processor) maybePrettyPrintZapdriverLine(line string, lineData map[string]interface{}) (string, error) {
	var buffer bytes.Buffer
	parsedTime, err := time.Parse(time.RFC3339, lineData["time"].(string))
	if err != nil {
		return "", err
	}

	buffer.WriteString(fmt.Sprintf("[%s]", parsedTime.Format("2006-01-02 15:04:01.000 MST")))

	buffer.WriteByte(' ')
	buffer.WriteString(p.colorizeSeverity(lineData["severity"].(string)).String())

	buffer.WriteByte(' ')
	buffer.WriteString(Gray(fmt.Sprintf("(%s)", lineData["caller"].(string))).String())

	buffer.WriteByte(' ')
	buffer.WriteString(Blue(lineData["message"].(string)).String())

	// Standard stuff
	delete(lineData, "time")
	delete(lineData, "severity")
	delete(lineData, "caller")
	delete(lineData, "message")

	// Extra stuff
	delete(lineData, "labels")
	delete(lineData, "logging.googleapis.com/sourceLocation")

	if len(lineData) > 0 {
		// FIXME: This is poor, we would like to print in a single line stuff that are not too
		//        big. But what represents a too big value exactly? We would need to serialize to
		//        JSON, check lenght, if smaller than threshold, print with space, otherwise
		//        re-serialize with pretty-printing stuff
		var jsonBytes []byte
		if len(lineData) <= 2 {
			jsonBytes, err = json.Marshal(lineData)
		} else {
			jsonBytes, err = json.MarshalIndent(lineData, "", "  ")
		}

		if err != nil {
			// FIXME: We could print each line as raw text maybe when it's not working?
			debug.Println(err)
		} else {
			buffer.WriteByte(' ')
			buffer.Write(jsonBytes)
		}
	}

	return buffer.String(), nil
}

func (p *processor) colorizeSeverity(severity string) aurora.Value {
	color := severityToColor[strings.ToLower(severity)]
	if color == 0 {
		color = BlueFg
	}

	return Colorize(severity, color)
}
