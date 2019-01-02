## In Progress

- Fixed bug where zap driver was still having `msg` field in JSON payload.
- Fixed bug where zap driver format timestamp's minutes was always fixed at `12`.
- Fixed bug where invalid JSON was not printing line at all.
- Added support for `zap.NewProduction` default lines.
- Fixed bug where an extra new line was printed at end of stream.
- Fixed bug where non-log line were printed twice.
- Added formatting of Zap JSON line.
