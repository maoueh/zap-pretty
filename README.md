# zap-pretty

This module provides a basic log prettifier for the [zap](https://github.com/uber-go/zap)
logging library. It reads a standard Zap JSON log line:

```
{"severity":"INFO","time":"2018-12-10T17:06:24.10107832Z","caller":"main.go:45","message":"doing some stuff","count":2}
```

And formats it to (with coloring):

```
[2018-12-10 17:06:24.101 UTC] INFO (main.go:45) doing some stuff {"count":2}
```

Compatible with `zap.NewProduction` and `zapdriver.NewProduction` formats out of the box.

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
$ go get -u github.com/maoueh/zap-pretty
```

**Note** Source installation requires Go 1.11+.

## Usage

Simply pipe the output of the CLI tool generating Zap JSON log lines to the `zap-pretty` tool:

```sh
zap_instrumented | zap-pretty
```

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

### CLI Arguments

None for now.

## License

MIT License
