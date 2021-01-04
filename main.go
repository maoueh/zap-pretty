package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
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
var debugEnabled = false
var severityToColor map[string]Color

// Provided via ldflags by goreleaser automatically
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var errNonZapLine = errors.New("non-zap line")

func init() {
	if os.Getenv("ZAP_PRETTY_DEBUG") != "" {
		debug = log.New(os.Stderr, "[pretty-debug] ", 0)
		debugEnabled = true
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

type processorOption interface {
	apply(p *processor)
}

type processorOptionFunc func(p *processor)

func (f processorOptionFunc) apply(p *processor) {
	f(p)
}

func withAllFields() processorOption {
	return processorOptionFunc(func(p *processor) {
		p.showAllFields = true
	})
}

type processor struct {
	scanner       *bufio.Scanner
	output        io.Writer
	showAllFields bool
}

var showAllFlag = flag.Bool("all", false, "Show ")
var versionFlag = flag.Bool("version", false, "Prints version information and exit")

var showAll = false

func main() {
	flag.Parse()

	if *versionFlag {
		printVersion()
		os.Exit(0)
	}

	go NewSignaler().forwardAllSignalsToProcessGroup()

	// FIXME: How could we make it more resilient to we simply drop the line instead? Would that mean our own "scanner"?
	// New scanner with a maximum of 250MiB per line, pass that, we panic.
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(nil, 250*1024*1024)

	processor := &processor{
		scanner:       scanner,
		output:        os.Stdout,
		showAllFields: *showAllFlag,
	}

	processor.process()
}

func printVersion() {
	fmt.Printf("zap-pretty %s (commit: %s, date: %v)\n", version, commit, date)
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
		debugPrintln("Scanner terminated with error: %s", err)
	}
}

func (p *processor) processLine(line string) {
	defer func() {
		if err := recover(); err != nil {
			p.unformattedPrintLine(line, "Panic occurred while processing line '%s', ending processing (%s)", line, err)
		}
	}()

	debugPrintln("Processing line: %s", line)
	reader := bytes.NewReader([]byte(line))
	decoder := json.NewDecoder(reader)

	token, err := decoder.Token()
	if err != nil {
		p.unformattedPrintLine(line, "Does not look like a JSON line, ending processing (%s)", err)
		return
	}

	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		p.unformattedPrintLine(line, "Expecting a JSON object delimited, ending processing")
		return
	}

	lineData := map[string]interface{}{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			p.unformattedPrintLine(line, "Invalid JSON key in line, ending processing (%s)", err)
			return
		}

		key := token.(string)

		// if keys[key] {
		// 	// Key duplicated here ...
		// }
		// keys[key] = true

		var value interface{}
		if err := decoder.Decode(&value); err != nil {
			p.unformattedPrintLine(line, "Invalid JSON value in line, ending processing (%s)", err)
			return
		}

		lineData[key] = value
	}

	// Read the ending delimiter of the JSON object
	if _, err := decoder.Token(); err != nil {
		p.unformattedPrintLine(line, "Invalid JSON, misssing object end delimiter in line, ending processing (%s)", err)
		return
	}

	prettyLine, err := p.maybePrettyPrintLine(line, lineData)

	if err != nil {
		fmt.Fprint(p.output, line)

		switch err {
		case errNonZapLine:
			debugPrintln("Not a known zap line format")
		default:
			debugPrintln("Not printing line due to error: %s", err)
		}
	} else {
		fmt.Fprint(p.output, prettyLine)
	}
}

func (p *processor) maybePrettyPrintLine(line string, lineData map[string]interface{}) (string, error) {
	if lineData["level"] != nil && lineData["ts"] != nil && lineData["caller"] != nil && lineData["msg"] != nil {
		return p.maybePrettyPrintZapLine(line, lineData)
	}

	if lineData["severity"] != nil && (lineData["time"] != nil || lineData["timestamp"] != nil) && lineData["caller"] != nil && lineData["message"] != nil {
		return p.maybePrettyPrintZapdriverLine(line, lineData)
	}

	return "", errNonZapLine
}

func (p *processor) maybePrettyPrintZapLine(line string, lineData map[string]interface{}) (string, error) {
	logTimestamp, err := tsFieldToTimestamp(lineData["ts"])
	if err != nil {
		return "", fmt.Errorf("unable to process field 'ts': %s", err)
	}

	var buffer bytes.Buffer
	p.writeHeader(&buffer, logTimestamp, lineData["level"].(string), lineData["caller"].(string), lineData["msg"].(string))

	// Delete standard stuff from data fields
	delete(lineData, "level")
	delete(lineData, "ts")
	delete(lineData, "caller")
	delete(lineData, "msg")

	stacktrace := ""
	if t, ok := lineData["stacktrace"].(string); ok && t != "" {
		delete(lineData, "stacktrace")
		stacktrace = t
	}

	p.writeJSON(&buffer, lineData)

	if stacktrace != "" {
		p.writeErrorDetails(&buffer, "", stacktrace)
	}

	return buffer.String(), nil
}

var zeroTime = time.Time{}

func tsFieldToTimestamp(input interface{}) (*time.Time, error) {
	switch v := input.(type) {
	case float64:
		nanosSinceEpoch := v * time.Second.Seconds()
		secondsPart, nanosPart := math.Modf(nanosSinceEpoch)
		timestamp := time.Unix(int64(secondsPart), int64(nanosPart/time.Nanosecond.Seconds()))

		return &timestamp, nil

	case string:
		timestamp, err := time.Parse(time.RFC3339Nano, v)
		timestamp = timestamp.Local()

		return &timestamp, err
	}

	return &zeroTime, fmt.Errorf("don't know how to turn %T (value %s) into a time.Time object", input, input)
}

func (p *processor) maybePrettyPrintZapdriverLine(line string, lineData map[string]interface{}) (string, error) {
	timeField := "time"
	timeValue := lineData[timeField]
	if lineData[timeField] == nil {
		timeField = "timestamp"
		timeValue = lineData[timeField]
	}

	var buffer bytes.Buffer
	parsedTime, err := time.Parse(time.RFC3339, timeValue.(string))
	if err != nil {
		return "", fmt.Errorf("unable to process field 'time': %s", err)
	}

	p.writeHeader(&buffer, &parsedTime, lineData["severity"].(string), lineData["caller"].(string), lineData["message"].(string))

	// Delete standard stuff from data fields
	delete(lineData, timeField)
	delete(lineData, "severity")
	delete(lineData, "caller")
	delete(lineData, "message")

	if !p.showAllFields {
		delete(lineData, "labels")
		delete(lineData, "serviceContext")
		delete(lineData, "logging.googleapis.com/labels")
		delete(lineData, "logging.googleapis.com/sourceLocation")
	}

	errorVerbose := ""
	if t, ok := lineData["errorVerbose"].(string); ok && t != "" {
		delete(lineData, "errorVerbose")
		errorVerbose = t
	}

	stacktrace := ""
	if t, ok := lineData["stacktrace"].(string); ok && t != "" {
		delete(lineData, "stacktrace")
		stacktrace = t
	}

	p.writeJSON(&buffer, lineData)

	if errorVerbose != "" || stacktrace != "" {
		p.writeErrorDetails(&buffer, errorVerbose, stacktrace)
	}

	return buffer.String(), nil
}

func (p *processor) writeHeader(buffer *bytes.Buffer, timestamp *time.Time, severity string, caller string, message string) {
	buffer.WriteString(fmt.Sprintf("[%s]", timestamp.Format("2006-01-02 15:04:05.000 MST")))

	buffer.WriteByte(' ')
	buffer.WriteString(p.colorizeSeverity(severity).String())

	buffer.WriteByte(' ')
	buffer.WriteString(Gray(12, fmt.Sprintf("(%s)", caller)).String())

	buffer.WriteByte(' ')
	buffer.WriteString(Blue(message).String())
}

var temporaryStackSpacer = "_-@\\!/@-_"

func (p *processor) writeErrorDetails(buffer *bytes.Buffer, errorVerbose string, stacktrace string) {
	if stacktrace != "" {
		buffer.WriteByte('\n')
		buffer.WriteString("Stacktrace\n")
		buffer.WriteString("    " + strings.ReplaceAll(stacktrace, "\n", "\n    "))
	}

	if stacktrace != "" && errorVerbose != "" {
		// If both are present, stacktrace has print something, so let's add an extra empty line here for spacing
		buffer.WriteByte('\n')
	}

	// The `errorVerbose` seems to contain a stack trace for each error captured. This behavior
	// comes from `github.com/pkg/errors` that create a stack of errors, each of the item having an associate
	// stacktrace.
	if errorVerbose != "" {
		writeErrorVerbose(buffer, errorVerbose)
	}
}

func writeErrorVerbose(buffer *bytes.Buffer, errorVerbose string) {
	joinedErrorVerbose := strings.ReplaceAll(errorVerbose, "\n\t", temporaryStackSpacer)
	scanner := bufio.NewScanner(strings.NewReader("  " + joinedErrorVerbose))

	var linePrevious *string
	var lineCurrent *string
	startedSection := false

	buffer.WriteByte('\n')
	buffer.WriteString("Error Verbose\n")
	for scanner.Scan() {
		if lineCurrent != nil {
			linePrevious = lineCurrent
		}

		line := scanner.Text()
		lineCurrent = &line

		if linePrevious != nil {
			isPreviousStackLine := strings.Contains(*linePrevious, temporaryStackSpacer)
			isStackLine := strings.Contains(line, temporaryStackSpacer)

			if isStackLine && !isPreviousStackLine {
				// This condition means we are at a section boundary, let's add some extra spacing here
				writeStackSectionTitle(buffer, *linePrevious)
				startedSection = true
			} else if isPreviousStackLine {
				writeStackLine(buffer, *linePrevious, startedSection, false)
				startedSection = false
			} else {
				buffer.WriteString(*linePrevious)
				buffer.WriteByte('\n')

				startedSection = false
			}
		}
	}

	if lineCurrent != nil {
		isStackLine := strings.Contains(*lineCurrent, temporaryStackSpacer)

		if isStackLine {
			writeStackLine(buffer, *lineCurrent, startedSection, true)
		} else {
			// It means we have seen more than one line, so we need the extra padding
			if linePrevious != nil {
				buffer.WriteString("  ")
			}

			buffer.WriteString(*lineCurrent)
		}
	}
}

func writeStackSectionTitle(buffer *bytes.Buffer, line string) {
	buffer.WriteByte('\n')
	buffer.WriteString("  ")
	buffer.WriteString(line)
}

func writeStackLine(buffer *bytes.Buffer, line string, isFirstStack, isLastStack bool) {
	if isFirstStack {
		buffer.WriteByte('\n')
	}

	buffer.WriteString("    ")
	buffer.WriteString(strings.Replace(line, temporaryStackSpacer, "\n    \t", 2))

	if !isLastStack {
		buffer.WriteByte('\n')
	}
}

func (p *processor) writeJSON(buffer *bytes.Buffer, data map[string]interface{}) {
	if len(data) <= 0 {
		return
	}

	// FIXME: This is poor, we would like to print in a single line stuff that are not too
	//        big. But what represents a too big value exactly? We would need to serialize to
	//        JSON, check length, if smaller than threshold, print with space, otherwise
	//        re-serialize with pretty-printing stuff
	var jsonBytes []byte
	var err error

	if len(data) <= 3 {
		jsonBytes, err = json.Marshal(data)
	} else {
		jsonBytes, err = json.MarshalIndent(data, "", "  ")
	}

	if err != nil {
		// FIXME: We could print each line as raw text maybe when it's not working?
		debugPrintln("Unable to marshal data as JSON: %s", err)
	} else {
		buffer.WriteByte(' ')
		buffer.Write(jsonBytes)
	}
}

func (p *processor) colorizeSeverity(severity string) aurora.Value {
	color := severityToColor[strings.ToLower(severity)]
	if color == 0 {
		color = BlueFg
	}

	return Colorize(severity, color)
}

func (p *processor) unformattedPrintLine(line string, message string, args ...interface{}) {
	debugPrintln(message, args...)
	fmt.Fprint(p.output, line)
}

func debugPrintln(msg string, args ...interface{}) {
	if debugEnabled {
		debug.Printf(msg+"\n", args...)
	}
}
