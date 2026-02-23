#  Gaze

> A beautiful Terminal UI for monitoring and managing local development ports

Gaze is a real-time dashboard for your local development environment. It solves the everyday annoyance of hung ports, ghost processes, and "address already in use" errors by giving you instant visibility and control over your local network sockets—all without leaving your terminal.

##  Features

-  **Auto-Discovery**: Continuously scans your local ports to detect active connections
-  **Process Identification**: Maps each port to its process name and PID
-  **Port History Tracking**: Tracks when ports open/close and shows uptime for each active port
-  **History View**: Browse complete port lifecycle with timestamps and event history
-  **Export Functionality**: Export port snapshots to JSON and CSV for auditing or sharing
-  **Flexible Sorting**: Sort by Port, PID, or Process name with ascending/descending order
-  **Kill Switch**: Terminate hung processes with a single keystroke
-  **Beautiful UI**: Modern terminal interface with colors and smooth interactions
-  **Real-time Updates**: Auto-refreshes every 3 seconds to keep you in sync
-  **Cross-Platform**: Works on macOS, Linux, and Windows

##  Why Gaze?

Ever encountered this?
```
Error: listen EADDRINUSE: address already in use :::3000
```

Instead of hunting through terminal windows or running complex `lsof` commands, just launch Gaze. See all your active ports at a glance and kill the culprit with a single key press.

##  Installation

### Prerequisites

- Go 1.21 or higher

### Build from Source

```bash
# Clone the repository
git clone https://github.com/junj-st/gaze.git
cd gaze

# Install dependencies
make install

# Build the binary
make build

# Run it
./bin/gaze
```

### Quick Install

```bash
go install github.com/junj-st/gaze/cmd/gaze@latest
```

## Usage

Simply run:
```bash
gaze
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate through ports |
| `s` | Cycle sort column (Port → PID → Process) |
| `a` | Toggle sort order (ascending ↔ descending) |
| `e` | Export current snapshot to JSON & CSV |
| `h` | Toggle history view |
| `k` | Kill the selected process |
| `r` | Manual refresh |
| `q` or `Esc` | Quit |

## Architecture

Gaze follows clean architecture principles:

```
gaze/
├── cmd/gaze/          # Entry point
│   └── main.go
├── internal/
│   ├── scanner/       # OS interaction layer (ports & PIDs)
│   │   └── scanner.go
│   └── ui/            # Bubble Tea TUI
│       └── ui.go
├── Makefile           # Build automation
└── go.mod
```

### Tech Stack

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: The Elm Architecture for Go, powering the TUI
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout for terminal UIs
- **[gopsutil](https://github.com/shirou/gopsutil)**: Cross-platform library for system information

##  Development

### Run in Development Mode
```bash
make run
```

### Build for Multiple Platforms
```bash
make build-all
```

This creates binaries for:
- macOS (Intel & Apple Silicon)
- Linux (AMD64)
- Windows (AMD64)

### Run Tests
```bash
make test
```

## Screenshots

### Main View (with Uptime)
```
🔍 GAZE - Local Port Monitor

┌───────────────────────────────────────────────────────────────────────────┐
│ Port       PID        Process              Uptime          Status         │
├───────────────────────────────────────────────────────────────────────────┤
│ 3000       12345      node                 2h 15m 32s      LISTEN         │
│ 5432       23456      postgres             5d 3h 12m       LISTEN         │
│ 6379       34567      redis-server         1d 18h 45m      LISTEN         │
│ 8080       45678      docker-proxy         45m 12s         LISTEN         │
└───────────────────────────────────────────────────────────────────────────┘

Monitoring 4 ports • Last scan: 1s ago
Sorted by: Port ↑

↑/↓: Navigate • s: Sort • a: Order • e: Export • h: History • k: Kill • r: Refresh • q: Quit
```

### History View (press `h`)
```
GAZE - Port History

┌───────────────────────────────────────────────────────────────────────────┐
│ Port   Process              Status   First Seen   Last Seen    Uptime     │
├───────────────────────────────────────────────────────────────────────────┤
│ 3000   node                 ACTIVE   14:23:10     16:38:42     2h 15m 32s │
│ 8000   python3              CLOSED   14:00:00     14:15:30     -          │
│ 5432   postgres             ACTIVE   12:00:00     16:38:42     4h 38m 42s │
│ 9000   java                 CLOSED   13:45:12     15:20:00     -          │
└───────────────────────────────────────────────────────────────────────────┘

Tracked: 15 ports • Active: 4 • Events: 32

↑/↓: Navigate • h: Back to Ports • e: Export • q: Quit
```

### Export Feature (press `e`)
Exports are saved to your home directory:
- `gaze-export-2026-02-22-16-38-42.json` - Full snapshot with statistics
- `gaze-export-2026-02-22-16-38-42.csv` - Spreadsheet-friendly format


## License

MIT License - see [LICENSE](LICENSE) for details

##  Acknowledgments

Built with amazing Go libraries from [Charm](https://charm.sh/) and inspired by modern developer tools like `lazygit` and `k9s`.

---

**made by [jun jiang](https://github.com/junj-st)**
