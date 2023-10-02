# zap-pretty

This module provides a basic log prettifier for the [zap](https://github.com/uber-go/zap)
logging library. It reads a standard Zap JSON log line:

```
{"severity":"INFO","time":"2018-12-10T17:06:24.10107832Z","caller":"main.go:45","message":"doing some stuff","count":2}
```

And formats it to:

![](./docs/readme_colored_output.png)


The tool receives JSON lines, determines if they seems like a log line of a supported format
and pretty print it to standard output.

Support Zap logging formats:
- `zap.NewProduction`
- `zapdriver.NewProduction`

## Install

### Homebrew

```sh
brew install maoueh/tap/zap-pretty
```

### Binary

Download the binary package for your platform, the list is available at
https://github.com/maoueh/zap-pretty/releases.

Unpack the binary somewhere on your disk and ensure the binary is in a
directory included in your $PATH variable.

### Source

```sh
go install github.com/maoueh/zap-pretty@latest
```

**Note** Source installation requires Go 1.16+.

## Usage

Simply pipe the output of the CLI tool generating Zap JSON log lines to the `zap-pretty` tool:

```sh
zap_instrumented | zap-pretty
```

The tool supports Zap standard production format as well as the Zapdriver standard format (for
consumption by Google Stackdriver).

### Zapdriver

When using the Zapdriver format, those fields are removed by default from the prettified version
to reduce the clutter of the logs:

- `labels`
- `serviceContext`
- `logging.googleapis.com/labels`
- `logging.googleapis.com/sourceLocation`

If you want to see those fields, you can use `--all` flag:

```sh
zap_instrumented | zap-pretty --all
```

### CLI Arguments

- `--all` - Show all fields of the line, even those filtered out by default for the active logger format (default `false`).
- `--version` - Show version information.
- `-n` - Format JSON as multiline if got more than n elements in data (default 3).

### Troubleshoot

#### No Conversion

By default, when using `zap.NewProductionConfig()`, all log statements are issued on
the `/dev/stderr` stream. That means that if you plainly do `zap_instrumented | zap-pretty`,
your JSON lines will not be prettified.

Why? Simply because you are actually piping only the `stdout` stream to `zap-pretty`
but your actual log statements are written on `stderr` so they never make their way
up to `zap-pretty`.

Ensures that JSON line you are seeing is redirected to standard output, if it works
when doing `zap_instrumented 2>&1 | zap-pretty`, then it means logs are going out
to `stderr`.

You can live like this, but if you want to customize your logs to output to `stdout`
instead, simply perform the following changes:

```
    return zap.NewProduction()
```

To:

```
    config := zap.NewProductionConfig()
    config.OutputPaths = []string{"stdout"}
    config.ErrorOutputPaths = []string{"stdout"}

    return config.Build()
```

## License

MIT License
