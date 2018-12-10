package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/logrusorgru/aurora"
	. "github.com/logrusorgru/aurora"
)

var debug bool
var severityToColor map[string]Color

func init() {
	if os.Getenv("ZAP_PRETTY_DEBUG") != "" {
		debug = true
	}

	severityToColor = make(map[string]Color)
	severityToColor["DEBUG"] = BlueFg
	severityToColor["INFO"] = GreenFg
	severityToColor["WARNING"] = BrownFg
	severityToColor["ERROR"] = RedFg
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		processLine(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

func processLine(line string) {
	if !mightBeJson(line) {
		fmt.Println(line)
	}

	var lineData map[string]interface{}
	err := json.Unmarshal([]byte(line), &lineData)
	if err != nil {
		maybeLogError(err)
		fmt.Println(line)
	}

	prettyLine, err := maybePrettyPrintLine(line, lineData)
	if prettyLine != "" {
		fmt.Println(prettyLine)
	}

	if err != nil {
		maybeLogError(err)
	}
}

func mightBeJson(line string) bool {
	// TODO: Shall we make an optimization to first check if the line might be
	//       a valid JSON object an only process it if it's the case? Let's process
	//       all line for now!
	return true
}

func maybePrettyPrintLine(line string, lineData map[string]interface{}) (string, error) {
	if lineData["time"] == nil ||
		lineData["severity"] == nil ||
		lineData["caller"] == nil ||
		lineData["message"] == nil {
		return line, nil
	}

	var buffer bytes.Buffer
	parsedTime, err := time.Parse(time.RFC3339, lineData["time"].(string))
	if err != nil {
		return line, err
	}

	buffer.WriteString(fmt.Sprintf("[%s]", parsedTime.Format("2006-01-02 15:04:01.000 MST")))

	buffer.WriteByte(' ')
	buffer.WriteString(colorizeSeverity(lineData["severity"].(string)).String())

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
			maybeLogError(err)
		} else {
			buffer.WriteByte(' ')
			buffer.Write(jsonBytes)
		}
	}

	return buffer.String(), nil
}

func colorizeSeverity(severity string) aurora.Value {
	color := severityToColor[severity]
	if color == 0 {
		color = BlueFg
	}

	return Colorize(severity, color)
}

func maybeLogError(err error) {
	if debug {
		fmt.Println(err)
	}
}
