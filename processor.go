package zapp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	. "github.com/logrusorgru/aurora"
)

var (
	severityToColor map[string]Color
)

var errNonZapLine = errors.New("non-zap line")

func init() {
	severityToColor = make(map[string]Color)
	severityToColor["debug"] = BlueFg
	severityToColor["info"] = GreenFg
	severityToColor["warning"] = BrownFg
	severityToColor["error"] = RedFg
	severityToColor["dpanic"] = RedFg
	severityToColor["panic"] = RedFg
	severityToColor["fatal"] = RedFg
}

type ProcessorOption interface {
	apply(p *Processor)
}

type ProcessorOptionFunc func(p *Processor)

func (f ProcessorOptionFunc) apply(p *Processor) {
	f(p)
}

func WithDelta(show bool) ProcessorOption {
	return ProcessorOptionFunc(func(p *Processor) {
		p.delta = show
	})
}

func WithMultilineJSONForced(forced bool) ProcessorOption {
	return ProcessorOptionFunc(func(p *Processor) {
		p.multilineJSONForced = forced
	})
}

func WithAllFields() ProcessorOption {
	return ProcessorOptionFunc(func(p *Processor) {
		p.showAllFields = true
	})
}

func WithMultilineJSONFieldThreshold(threshold int) ProcessorOption {
	return ProcessorOptionFunc(func(p *Processor) {
		p.multilineJSONFieldThreshold = threshold
	})
}

func WithDebugLogger(logger *log.Logger) ProcessorOption {
	return ProcessorOptionFunc(func(p *Processor) {
		p.debugEnabled = true
		p.debugLogger = logger
	})
}

type KeyValueFilter struct {
	Key   string
	Value any
}

func ParseKeyValueFilter(input string) (*KeyValueFilter, error) {
	key, value, found := strings.Cut(input, "=")
	if !found {
		return nil, fmt.Errorf("invalid filter %q: expected key=value", input)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("invalid filter %q: key cannot be empty", input)
	}

	value = strings.TrimSpace(value)

	var parsedValue any
	if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
		parsedValue = value
	}

	return &KeyValueFilter{Key: key, Value: parsedValue}, nil
}

func (f *KeyValueFilter) Matches(lineAllData map[string][]any) bool {
	if f == nil {
		return true
	}

	values, found := lineAllData[f.Key]
	if !found {
		return false
	}

	if reflect.DeepEqual(getValueOrArray(lineAllData, f.Key), f.Value) {
		return true
	}

	for _, value := range values {
		if reflect.DeepEqual(value, f.Value) {
			return true
		}
	}

	return false
}

func WithKeyValueFilter(filter *KeyValueFilter) ProcessorOption {
	return ProcessorOptionFunc(func(p *Processor) {
		p.filter = filter
	})
}

type Processor struct {
	scanner *bufio.Scanner
	output  io.Writer

	// State
	lastProcessedTimestamp *time.Time

	// Options
	debugEnabled                bool
	debugLogger                 *log.Logger
	multilineJSONFieldThreshold int
	multilineJSONForced         bool
	showAllFields               bool
	delta                       bool
	filter                      *KeyValueFilter
}

func NewProcessor(scanner *bufio.Scanner, output io.Writer, opts ...ProcessorOption) *Processor {
	processor := &Processor{
		scanner: scanner,
		output:  output,

		debugEnabled:                false,
		debugLogger:                 nil,
		multilineJSONFieldThreshold: 3,
		showAllFields:               false,
	}

	for _, opt := range opts {
		opt.apply(processor)
	}

	return processor
}

func (p *Processor) Process() {
	hasPrinted := false
	for p.scanner.Scan() {
		output, printed := p.processLine(p.scanner.Text())
		if !printed {
			continue
		}

		if hasPrinted {
			fmt.Fprintln(p.output)
		}

		fmt.Fprint(p.output, output)
		hasPrinted = true
	}

	if err := p.scanner.Err(); err != nil {
		p.debugPrintln("Scanner terminated with error: %w", err)
	}
}

func (p *Processor) processLine(line string) (output string, printed bool) {
	defer func() {
		if err := recover(); err != nil {
			output = p.unformattedLine(line, "Panic occurred while processing line '%s', ending processing (%s)", line, err)
			printed = true
		}
	}()

	p.debugPrintln("Processing line: %s", line)
	reader := bytes.NewReader([]byte(line))
	decoder := json.NewDecoder(reader)

	token, err := decoder.Token()
	if err != nil {
		return p.unformattedLine(line, "Does not look like a JSON line, ending processing (%s)", err), true
	}

	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return p.unformattedLine(line, "Expecting a JSON object delimited, ending processing"), true
	}

	lineAllData := map[string][]any{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return p.unformattedLine(line, "Invalid JSON key in line, ending processing (%s)", err), true
		}

		key := token.(string)

		var value any
		if err := decoder.Decode(&value); err != nil {
			return p.unformattedLine(line, "Invalid JSON value in line, ending processing (%s)", err), true
		}

		lineAllData[key] = append(lineAllData[key], value)
	}

	// Read the ending delimiter of the JSON object
	if _, err := decoder.Token(); err != nil {
		return p.unformattedLine(line, "Invalid JSON, missing object end delimiter in line, ending processing (%s)", err), true
	}

	if p.filter != nil && !p.filter.Matches(lineAllData) {
		p.debugPrintln("Skipping line because filter %q did not match", p.filter.Key)
		return "", false
	}

	prettyLine, err := p.prettifyIfMatchKnownFormats(line, lineAllData)

	if err != nil {
		switch err {
		case errNonZapLine:
			p.debugPrintln("Not a known zap line format")
		default:
			p.debugPrintln("Not printing line due to error: %s", err)
		}

		return line, true
	} else {
		return prettyLine, true
	}
}

func getLastValue(lineAllData map[string][]any, key string) any {
	if values, exists := lineAllData[key]; exists && len(values) > 0 {
		return values[len(values)-1]
	}
	return nil
}

func getValueOrArray(lineAllData map[string][]any, key string) any {
	if values, exists := lineAllData[key]; exists && len(values) > 0 {
		if len(values) == 1 {
			return values[0]
		}
		return values
	}
	return nil
}

func (p *Processor) prettifyIfMatchKnownFormats(line string, lineAllData map[string][]any) (string, error) {
	if getLastValue(lineAllData, "level") != nil && getLastValue(lineAllData, "ts") != nil && getLastValue(lineAllData, "msg") != nil {
		return p.prettifyZapLine(line, lineAllData)
	}

	if getLastValue(lineAllData, "severity") != nil && (getLastValue(lineAllData, "time") != nil || getLastValue(lineAllData, "timestamp") != nil) && getLastValue(lineAllData, "message") != nil {
		return p.prettifyZapdriveLine(line, lineAllData)
	}

	if getLastValue(lineAllData, "level") != nil && getLastValue(lineAllData, "message") != nil && getLastValue(lineAllData, "time") != nil {
		return p.prettifyLevelMessageTimeLine(line, lineAllData)
	}

	return "", errNonZapLine
}

func (p *Processor) prettifyZapLine(line string, lineAllData map[string][]any) (string, error) {
	logTimestamp, err := tsFieldToTimestamp(getLastValue(lineAllData, "ts"))
	if err != nil {
		return "", fmt.Errorf("unable to process field 'ts': %w", err)
	}

	var caller *string
	if v := getLastValue(lineAllData, "caller"); v != nil {
		callerStr := v.(string)
		caller = &callerStr
	}

	var logger *string
	if v := getLastValue(lineAllData, "logger"); v != nil {
		loggerStr := v.(string)
		logger = &loggerStr
	}

	var buffer bytes.Buffer
	p.writeHeader(&buffer, logTimestamp, getLastValue(lineAllData, "level").(string), caller, logger, getLastValue(lineAllData, "msg").(string))

	// Create filtered data map excluding standard fields
	filteredData := make(map[string]any)
	for key := range lineAllData {
		if key != "level" && key != "ts" && key != "caller" && key != "logger" && key != "msg" && key != "stacktrace" {
			filteredData[key] = getValueOrArray(lineAllData, key)
		}
	}

	stacktrace := ""
	if t := getLastValue(lineAllData, "stacktrace"); t != nil {
		if stacktraceStr, ok := t.(string); ok && stacktraceStr != "" {
			stacktrace = stacktraceStr
		}
	}

	p.writeJSON(&buffer, filteredData)

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

func (p *Processor) prettifyZapdriveLine(line string, lineAllData map[string][]any) (string, error) {
	timeField := "time"
	timeValue := getLastValue(lineAllData, timeField)
	if timeValue == nil {
		timeField = "timestamp"
		timeValue = getLastValue(lineAllData, timeField)
	}

	var buffer bytes.Buffer

	parsedTime, err := tsFieldToTimestamp(timeValue)
	if err != nil {
		return "", fmt.Errorf("unable to process field %q: %w", timeField, err)
	}

	var caller *string
	if v := getLastValue(lineAllData, "caller"); v != nil {
		callerStr := v.(string)
		caller = &callerStr
	}

	var logger *string
	if v := getLastValue(lineAllData, "logger"); v != nil {
		loggerStr := v.(string)
		logger = &loggerStr
	}

	p.writeHeader(&buffer, parsedTime, getLastValue(lineAllData, "severity").(string), caller, logger, getLastValue(lineAllData, "message").(string))

	// Create filtered data map excluding standard fields
	filteredData := make(map[string]any)
	for key := range lineAllData {
		if key != timeField && key != "severity" && key != "caller" && key != "logger" && key != "message" && key != "errorVerbose" && key != "stacktrace" {
			if p.showAllFields || (key != "labels" && key != "serviceContext" && key != "logging.googleapis.com/labels" && key != "logging.googleapis.com/sourceLocation") {
				filteredData[key] = getValueOrArray(lineAllData, key)
			}
		}
	}

	errorVerbose := ""
	if t := getLastValue(lineAllData, "errorVerbose"); t != nil {
		if errorVerboseStr, ok := t.(string); ok && errorVerboseStr != "" {
			errorVerbose = errorVerboseStr
		}
	}

	stacktrace := ""
	if t := getLastValue(lineAllData, "stacktrace"); t != nil {
		if stacktraceStr, ok := t.(string); ok && stacktraceStr != "" {
			stacktrace = stacktraceStr
		}
	}

	p.writeJSON(&buffer, filteredData)

	if errorVerbose != "" || stacktrace != "" {
		p.writeErrorDetails(&buffer, errorVerbose, stacktrace)
	}

	return buffer.String(), nil
}

func (p *Processor) prettifyLevelMessageTimeLine(line string, lineAllData map[string][]any) (string, error) {
	logTimestamp, err := tsFieldToTimestamp(getLastValue(lineAllData, "time"))
	if err != nil {
		return "", fmt.Errorf("unable to process field 'time': %w", err)
	}

	var buffer bytes.Buffer

	var logger *string
	// Handle multiple module keys by merging them with dot separator
	if moduleValues, exists := lineAllData["module"]; exists && len(moduleValues) > 0 {
		var modules []string
		for _, module := range moduleValues {
			if moduleStr, ok := module.(string); ok {
				modules = append(modules, moduleStr)
			}
		}
		if len(modules) > 0 {
			mergedModules := strings.Join(modules, ".")
			logger = &mergedModules
		}
	}

	p.writeHeader(&buffer, logTimestamp, getLastValue(lineAllData, "level").(string), nil, logger, getLastValue(lineAllData, "message").(string))

	// Create filtered data map excluding standard fields
	filteredData := make(map[string]any)
	for key := range lineAllData {
		if key != "level" && key != "time" && key != "message" && key != "module" && key != "stacktrace" {
			filteredData[key] = getValueOrArray(lineAllData, key)
		}
	}

	stacktrace := ""
	if t := getLastValue(lineAllData, "stacktrace"); t != nil {
		if stacktraceStr, ok := t.(string); ok && stacktraceStr != "" {
			stacktrace = stacktraceStr
		}
	}

	p.writeJSON(&buffer, filteredData)

	if stacktrace != "" {
		p.writeErrorDetails(&buffer, "", stacktrace)
	}

	return buffer.String(), nil
}

const timeFormat = "2006-01-02 15:04:05.000 MST"

func (p *Processor) writeHeader(buffer *bytes.Buffer, timestamp *time.Time, severity string, caller *string, logger *string, message string) {
	defer func() {
		if p.lastProcessedTimestamp == nil {
			p.lastProcessedTimestamp = timestamp
			return
		}

		if timestamp != nil {
			p.lastProcessedTimestamp = timestamp
		}
	}()

	if p.delta {
		delta := "-"
		if p.lastProcessedTimestamp != nil && timestamp != nil {
			delta = durationToString(timestamp.Sub(*p.lastProcessedTimestamp))
		}

		buffer.WriteString(fmt.Sprintf("[%s, %s]", timestamp.Format(timeFormat), delta))
	} else {
		buffer.WriteString(fmt.Sprintf("[%s]", timestamp.Format(timeFormat)))
	}

	buffer.WriteByte(' ')
	buffer.WriteString(p.colorizeSeverity(severity).String())

	if logger != nil && caller != nil {
		buffer.WriteByte(' ')
		buffer.WriteString(Gray(12, fmt.Sprintf("(%s, %s)", *logger, *caller)).String())
	} else if logger != nil {
		buffer.WriteByte(' ')
		buffer.WriteString(Gray(12, fmt.Sprintf("(%s)", *logger)).String())
	} else if caller != nil {
		buffer.WriteByte(' ')
		buffer.WriteString(Gray(12, fmt.Sprintf("(%s)", *caller)).String())
	}

	buffer.WriteByte(' ')
	buffer.WriteString(Blue(message).String())
}

var temporaryStackSpacer = "_-@\\!/@-_"

func (p *Processor) writeErrorDetails(buffer *bytes.Buffer, errorVerbose string, stacktrace string) {
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

func (p *Processor) writeJSON(buffer *bytes.Buffer, data map[string]interface{}) {
	if len(data) <= 0 {
		return
	}

	// FIXME: This is poor, we would like to print in a single line stuff that are not too
	//        big. But what represents a too big value exactly? We would need to serialize to
	//        JSON, check length, if smaller than threshold, print with space, otherwise
	//        re-serialize with pretty-printing stuff
	var jsonBytes []byte
	var err error

	if p.multilineJSONForced || len(data) > p.multilineJSONFieldThreshold {
		jsonBytes, err = json.MarshalIndent(data, "", "  ")
	} else {
		jsonBytes, err = json.Marshal(data)
	}

	if err != nil {
		// FIXME: We could print each line as raw text maybe when it's not working?
		p.debugPrintln("Unable to marshal data as JSON: %s", err)
	} else {
		buffer.WriteByte(' ')
		buffer.Write(jsonBytes)
	}
}

func (p *Processor) colorizeSeverity(severity string) aurora.Value {
	color := severityToColor[strings.ToLower(severity)]
	if color == 0 {
		color = BlueFg
	}

	return Colorize(strings.ToUpper(severity), color)
}

func (p *Processor) unformattedLine(line string, message string, args ...interface{}) string {
	p.debugPrintln(message, args...)
	return line
}

func (p *Processor) debugPrintln(msg string, args ...interface{}) {
	if p.debugEnabled {
		p.debugLogger.Printf(msg+"\n", args...)
	}
}
