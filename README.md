# reminderd

A background daemon that monitors mouse/keyboard input
and **reminds you to take a break after sitting for too long**.

The generic service name leaves room for other reminder types in the future.

## Usage

```bash
# Build
go build -o reminderd ./cmd/reminderd

# Run in background
./reminderd &
```

## How it works

- Polls the OS for the time since the last keyboard/mouse event.
- If you are continuously active for the configured limit (default 60m),
  it sends a desktop notification.
- After the reminder, if you keep working, it reminds again with
  exponential backoff starting at the configured initial backoff (default 5m),
  then doubling: 10m, 20m, ...
- The timer resets once you take a break
  (idle for the configured threshold, default 2m).
- Records activity history to daily files in `~/.reminderd/`.
- Serves a web UI with an activity chart and settings at <http://localhost:20902>.

## Web UI

Open <http://localhost:20902> in a browser. The web UI has three tabs:

![Activity History tab showing a bar chart of keyboard/mouse activity over the last 4 hours](reminderd20902.jpg)

### Activity History

A bar chart showing when you were active or idle.
You can choose a time range (Last 1h, 4h, 12h, 24h, 2d, 7d, 30d, 6m, 1y, all time).
Example summary: Last 4h | Active: 2h32m (63%) | Reminders: 2
Hover over any bar to see the active/total duration breakdown.

Activity is recorded to daily files in `~/.reminderd/` (e.g. `history-2026-04-03.jsonl`).
At daily rollover, the previous day's file is compacted:
only the first and last record of each consecutive state run are kept.

History is kept forever. Estimated storage:
~300 KB/year (compacted), ~42 MB/year (uncompacted, 10s poll, 8h/day).

### Configuration

View and edit all settings from the browser.
Each field has a tooltip explaining its meaning and recommended values.
Changes take effect within one poll interval (10s), no restart needed.

On first run, the app creates `~/.reminderd/config.json` with defaults:

```json
{
	"ContinuousActiveLimit": "60m",
	"IdleDurationToConsiderBreak": "2m",
	"NotificationInitialBackoff": "5m",
	"WebUIPort": 20902
}
```

### Notification

Send a test notification to verify that desktop alerts are working on your system.

TODO: allow user to customize notification content.

## Log Compaction and Activity State

### Log Compaction

`CompactHistory` keeps only the first and last entry of each consecutive same-state run.

```mermaid
flowchart LR
    A["Raw\nentries"] --> B["For each consecutive\nsame-state run"]
    B --> C{"Run length\n> 1?"}
    C -- yes --> D["Keep first\n+ last"]
    C -- no --> E["Keep the\nsingle entry"]
    D --> F["Compacted\nentries"]
    E --> F
```

### User Activity State

Each entry's state lasts until the next entry (state-boundary model).
Works identically on raw and compacted logs.

```mermaid
flowchart LR
    Q["Query: state\nat time T"] --> S["Find latest entry\nwhere timestamp <= T"]
    S --> R["That entry's\nState is the answer"]
```

## Design

```mermaid
graph TD
    K["main.go"] -->|startup| A
    K -->|startup| F
    A[UserInputTracker] -->|reload config, write history, restore active start| S
    A -->|break reminder| N[Notifier]
    F[HTTP Server] -->|read history, read/write config| S

    subgraph S ["~/.reminderd/"]
        C["config.json"]
        D["history-YYYY-MM-DD.jsonl"]
    end
```

## Platforms

- [x] Windows 10/11 (`GetLastInputInfo` from user32.dll)
- [x] macOS 13 Ventura (Core Graphics API)
- [ ] Linux Mint 22.3 "Zena" (X11, `XScreenSaverQueryInfo`, `notify-send`) — implemented, not tested

## Roadmap

- **v0.0.3**: minimal UI with system tray, install as a service (auto-start on boot).
