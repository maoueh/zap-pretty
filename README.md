# zap-pretty

This module provides a basic log prettifier for the [zap](https://github.com/uber-go/zap)
logging library. It reads a standard Zap production log line like:

```
{"severity":"INFO","time":"2018-12-10T17:06:24.10107832Z","caller":"main.go:45","message":"doing some stuff","count":2}
```

And formats it to:

```
[2018-12-10 17:06:24.101 UTC] INFO (main.go:45) doing some stuff {"count":2}
```

## Install

```sh
$ go get -u github.com/maoueh/zap-pretty
```

## Usage

It's recommended to use `zap-pretty` with `zap` by piping output to the CLI tool:

```sh
zap_instrumented | zap-pretty
```

### Current State

This package is a "work in progress". Current version works but it's the initial version, still
much more to do to make it production ready:

- Add tests to ensure all functionalities work
- Ensure date format parsing is correct
- Add more suppressed field for JSON output (with CLI argument to add more)
- Add CLI arguments similar to [pino-pretty](https://github.com/pinojs/pino-pretty#cli-arguments)
- Refactor code to be more "nice"
- Add benchmark to see how performant code is (or not)
- Filtering support of log statements?
- How to write JSON fields at the end of line?
- Other ideas?

### CLI Arguments

None for now.

## License

MIT License