## Unreleased

## 0.3.0 (April 21th, 2021)

- [Fix] Fix formatting when `caller` is not present
- [Improvement] The flag `-all` (or `--all`) can now be used to show fields that are filtered out by default
- [Improvement] More fields are filtered by defaults for Zapdriver format (`labels`, `serviceContext`, `logging.googleapis.com/labels` & `logging.googleapis.com/sourceLocation` are filtered out by default now).
- [Fix] Stacktrace are now properly printed when your log format is the Zap production standard one.
