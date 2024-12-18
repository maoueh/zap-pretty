# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.3.1

- Revamped CLI command description and flags.

- Added support for control flags through environment variable, see `zap-pretty --help` for full details.

- Added support for long form of `-n` which is `--multiline-json-threshold`.

- Added support for `-m, --multiline-json-force` which when set to true, forces every object to be printed on multiple line.

- Added support for `-d, --show-delta` which shows delta between current line and last seen valid log line:

  ```bash
  acme | zap-pretty -d
  [2024-12-18 09:27:49.160 EST, +0] INFO (acme) checking if block available
  [2024-12-18 09:28:39.160 EST, +40s] INFO (acme) optimistically fetching block {"block_num":308267722}
  ```

- Added flag `-n` that can be used to control after how many fields within an JSON object we start to print it multiple line.

### Fixed

- [Fix] Fix formatting when `timestamp` is unix timestamp and not string value.

## v0.3.0

- [Fix] Fix formatting when `caller` is not present
- [Improvement] The flag `-all` (or `--all`) can now be used to show fields that are filtered out by default
- [Improvement] More fields are filtered by defaults for Zapdriver format (`labels`, `serviceContext`, `logging.googleapis.com/labels` & `logging.googleapis.com/sourceLocation` are filtered out by default now).
- [Fix] Stacktrace are now properly printed when your log format is the Zap production standard one.
