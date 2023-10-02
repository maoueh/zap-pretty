# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.4.0

### Added

- Added flag `-n` that can be used to

### Fixed

- [Fix] Fix formatting when `timestamp` is unix timestamp and not string value.

## v0.3.0

- [Fix] Fix formatting when `caller` is not present
- [Improvement] The flag `-all` (or `--all`) can now be used to show fields that are filtered out by default
- [Improvement] More fields are filtered by defaults for Zapdriver format (`labels`, `serviceContext`, `logging.googleapis.com/labels` & `logging.googleapis.com/sourceLocation` are filtered out by default now).
- [Fix] Stacktrace are now properly printed when your log format is the Zap production standard one.
