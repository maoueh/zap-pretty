package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	zapp "github.com/maoueh/zap-pretty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
)

// Provided via ldflags by goreleaser automatically
var (
	version = "dev"
)

func main() {
	Run(
		"zap-pretty",
		"This module provides a basic log prettifier for the [zap](https://github.com/uber-go/zap) logging library",

		ConfigureVersion(version),
		ConfigureViper("ZAP_PRETTY"),
		OnCommandErrorPrintAndExit(),

		Description(`
			zap-pretty is a simple tool that reads Zap rendered logs, usually in JSON form from stdin and pretty
			prints them to stdout back.

			## Supported formats

			The tool supports the following JSON Zap formats.

			### zap.NewProduction

			JSON looks like '{"level":"info","ts":1545445711.144533,"caller":"c","msg":"m"}'
			and we support extra variations like 'timestamp' instead of 'ts', ISO-8601 timestamps, etc.

			### zapdriver.NewProduction

			JSON looks like '{"severity":"INFO","timestamp":"2018-12-21T23:06:49.435919-05:00","caller":"c:0","message":"m"}'
			and we support extra variations like 'time' instead of 'timestamp', etc.

			## Formats

			The tool also has formatting options controlled via flags:

			  - '--all' (ZAP_PRETTY_ALL)
			    Show all fields that would normally be ignored by default like 'serviceContext', 'labels', etc.

			  - '--show-delta, -d' (ZAP_PRETTY_SHOW_DELTA)
			    On the timestamp field, add delta from the last seen log line, if any.

			  - '--multiline-json-threshold, -n' (ZAP_PRETTY_MULTILINE_JSON_THRESHOLD)
			    Format JSON as multiline if got more than n elements in data.

			  - '--multiline-json-force, -m' (ZAP_PRETTY_MULTILINE_JSON_FORCE)
			    Force JSON to be printed as multiline even if it's below the threshold, overrides and ignores 'multiline-json-threshold'.
		`),

		Flags(func(flags *pflag.FlagSet) {
			flags.Bool("all", false, "Show all fields that would normally be ignored by default like 'serviceContext', 'labels', etc.")
			flags.BoolP("show-delta", "d", false, "On the timestamp field, add delta from the last seen log line, if any")
			flags.IntP("multiline-json-threshold", "n", 3, "Format JSON as multiline if got more than n elements in data")
			flags.BoolP("multiline-json-force", "m", false, "Force JSON to be printed as multiline even if it's below the threshold, overrides and ignores 'multiline-json-threshold'")
		}),

		Example(`
			# Print logs from file
			cat logs.json | zap-pretty
			[2024-12-18 09:27:49.160 EST] INFO (acme) block {"block":308267722}
			...

			# Live prettifying of logs
			go run ./cmd/acme | zap-pretty
			[2024-12-18 09:27:49.160 EST] INFO (acme) block {"block":308267722}
			...

			# Show delta from last log line
			go run ./cmd/acme | zap-pretty -d
			[2024-12-18 09:27:49.160 EST, +0] INFO (acme) checking if block available
			[2024-12-18 09:28:39.160 EST, +40s] INFO (acme) optimistically fetching block {"block_num":308267722}
			...
		`),

		Execute(run),
	)
}

func run(cmd *cobra.Command, args []string) error {
	debugEnabled := false
	var debugLogger *log.Logger

	if os.Getenv("ZAP_PRETTY_DEBUG") != "" {
		debugEnabled = true
		debugLogger = log.New(os.Stderr, "[pretty-debug] ", 0)
	}

	go zapp.NewSignaler(debugEnabled, debugLogger).ForwardAllSignalsToProcessGroup()

	// FIXME: How could we make it more resilient to we simply drop the line instead? Would that mean our own "scanner"?
	// New scanner with a maximum of 250MiB per line, pass that, we panic.
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(nil, 250*1024*1024)

	opts := []zapp.ProcessorOption{
		zapp.WithMultilineJSONFieldThreshold(sflags.MustGetInt(cmd, "multiline-json-threshold")),
		zapp.WithMultilineJSONForced(sflags.MustGetBool(cmd, "multiline-json-force")),
		zapp.WithDelta(sflags.MustGetBool(cmd, "show-delta")),
	}

	if os.Getenv("ZAP_PRETTY_DEBUG") != "" {
		opts = append(opts, zapp.WithDebugLogger(log.New(os.Stderr, "[pretty-debug] ", 0)))
	}

	if sflags.MustGetBool(cmd, "all") {
		opts = append(opts, zapp.WithAllFields())
	}

	zapp.NewProcessor(scanner, os.Stdout, opts...).Process()

	return nil
}

func OnCommandErrorPrintAndExit() CommandOption {
	if OnAssertionFailure == nil {
		OnAssertionFailure = func(message string) {
			if message != "" {
				fmt.Fprintln(os.Stderr, message)
			}
		}
	}

	return OnCommandError(func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	})
}
