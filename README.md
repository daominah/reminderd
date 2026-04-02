# reminderd

A background daemon that monitors mouse/keyboard input
and reminds you to take a break.

The generic service name leaves room for other reminder types in the future.

## How it works

- Polls the OS for the time since the last keyboard/mouse event.
- If you are continuously active for 1 hour
  (with no idle gap longer than 2 minutes), it sends a desktop notification.
- After the reminder, if the user keeps working, remind again with
  exponential backoff: 5m, 10m, 20m, ...
- The timer resets once you actually take a break
  (2 minutes of no input).
- Durations are configurable via Go constants in `pkg/logic/app.go`
  (requires a rebuild after changes).

## Platforms

- macOS 13 Ventura (Core Graphics API)
- Windows 10/11 (`GetLastInputInfo` from user32.dll)
- Linux Mint 22.3 "Zena" (X11, `XScreenSaverQueryInfo`)

## Usage

```bash
# Build
go build -o reminderd ./cmd/reminderd

# Run in background
./reminderd &
```

## Design

### Components

1. **Idle detector** (per-platform driver):
   one method `IdleSeconds() (float64, error)`.
   Three implementations via build tags (darwin, windows, linux).

2. **Notifier** (per-platform driver):
   one method `Notify(title, message string) error`.
   Shells out to `osascript` / PowerShell / `notify-send`.

3. **UserInputTracker** (core logic, platform-independent):
   polls idle detector every 30s.
   If idle < 5min: accumulate active duration.
   If idle >= 5min: reset everything.
   If active >= 1h: send reminder, then exponential backoff (5m, 10m, 20m, ...).

4. **`cmd/reminderd/main.go`**: wires drivers into tracker, starts the loop.

## Checklist

- [x] Step 1: Understand requirements
- [x] Step 2: Clarify vague items
- [x] Step 3: Spike (not needed)
- [x] Step 4: High-level design
- [x] Step 5: Detailed implementation plan
- [x] Step 6: Write failing tests
- [x] Step 7: Commit, push
- [x] Step 8: Implement the feature
- [x] Step 9: Document
- [x] Step 10: Commit and push
- [x] Step 11: Self-review
- [x] Step 12: Tested on macOS and Windows
