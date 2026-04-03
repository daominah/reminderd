# v0.0.2 PR Review

**Verdict: REQUEST_CHANGES**

**Summary:** This PR adds configurable durations, JSONL activity history with daily
compaction, a localhost web UI (solarized chart + notifications tab), and a JS chart
utility with its own test suite. The design is documented in the README, the Go code
is well-structured with GIVEN/WHEN/THEN test comments, and `go vet` is clean.
Two issues block merging.

## Blockers

### 1. History tests fail on Windows: missing `Close()` on `FileStore`

`pkg/driver/history/history_test.go` — all 6 tests

`FileStore` keeps `currentFile *os.File` open for the lifetime of the store. On
Windows, `t.TempDir()` calls `os.RemoveAll` at the end of each test, which fails
with "file used by another process" because the handle is still open.

The `FileStore` has no `Close()` method, so tests cannot release the handle.

Fix: add `Close() error` to `FileStore` (closes `currentFile` if non-nil), and
call `defer store.Close()` in each history test.

### 2. Changing `KeyboardMouseInputPollInterval` has no effect at runtime

`pkg/logic/app.go:91`, `pkg/logic/app.go:240-255`

The ticker is created once at startup using `t.pollInterval()`.
`reloadConfigIfChanged()` updates `t.config` but never touches the ticker. A user
who edits the config to change the poll interval will see no effect until restart,
contradicting the README's "Changes take effect within one poll interval, no restart
needed."

Fix: track the previous poll interval in `reloadConfigIfChanged`. When it changes,
reset the ticker (the ticker lives in `Run`, so either signal via a channel or move
it into the struct).

## Suggestions

### 3. `vnTimezone` is defined twice

`pkg/driver/history/history.go:16` and `pkg/driver/httpsvr/httpsvr.go:14` both
declare:

```go
var vnTimezone = time.FixedZone("ICT", 7*60*60)
```

Move to `pkg/model/model.go` (or `pkg/driver/base/`) so there is one source of
truth.

## Nitpicks

- `currentFile.Close()` return value is silently discarded — `history.go:51`.
  Low risk but worth logging.
- `HistoryReader` is a field of `UserInputTracker` but is never used by `Tick`
  (`app.go:26`). If it is only needed by the HTTP server, removing it from the
  struct keeps dependencies minimal.
