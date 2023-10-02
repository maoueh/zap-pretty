package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	zapp "github.com/maoueh/zap-pretty"
)

// Provided via ldflags by goreleaser automatically
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	showAllFlag                 = flag.Bool("all", false, "Show ")
	versionFlag                 = flag.Bool("version", false, "Prints version information and exit")
	multilineJSONFieldThreshold = flag.Int("n", 3, "Format JSON as multiline if got more than n elements in data")
)

func main() {
	flag.Parse()

	if *versionFlag {
		printVersion()
		os.Exit(0)
	}

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
		zapp.WithMultilineJSONFieldThreshold(*multilineJSONFieldThreshold),
	}

	if os.Getenv("ZAP_PRETTY_DEBUG") != "" {
		opts = append(opts, zapp.WithDebugLogger(log.New(os.Stderr, "[pretty-debug] ", 0)))
	}

	if *showAllFlag {
		opts = append(opts, zapp.WithAllFields())
	}

	zapp.NewProcessor(scanner, os.Stdout, opts...).Process()
}

func printVersion() {
	fmt.Printf("zap-pretty %s (commit: %s, date: %v)\n", version, commit, date)
}
