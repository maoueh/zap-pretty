## Unreleased

- [Improvement] The flag `-all` (or `--all`) can now be used to show fields that are filtered out by default
- [Improvement] More fields are filtered by defaults for Zapdriver format (`labels`, `serviceContext`, `logging.googleapis.com/labels` & `logging.googleapis.com/sourceLocation` are filtered out by default now).
- [Fix] Stacktrace are now properly printed when your log format is the Zap production standard one.
