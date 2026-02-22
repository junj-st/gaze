#  Gaze

> A beautiful Terminal UI for monitoring and managing local development ports

Gaze is a real-time dashboard for your local development environment. It solves the everyday annoyance of hung ports, ghost processes, and "address already in use" errors by giving you instant visibility and control over your local network sockets—all without leaving your terminal.

##  Features

-  **Auto-Discovery**: Continuously scans your local ports to detect active connections
-  **Process Identification**: Maps each port to its process name and PID
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
git clone https://github.com/junjiang/gaze.git
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
go install github.com/junjiang/gaze/cmd/gaze@latest
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

```
🔍 GAZE - Local Port Monitor

┌──────────────────────────────────────────────────────────────┐
│ Port       PID        Process                 Status         │
├──────────────────────────────────────────────────────────────┤
│ 3000       12345      node                    LISTEN         │
│ 5432       23456      postgres                LISTEN         │
│ 6379       34567      redis-server            LISTEN         │
│ 8080       45678      docker-proxy            LISTEN         │
└──────────────────────────────────────────────────────────────┘

Monitoring 4 ports • Last scan: 1s ago

↑/↓: Navigate • k: Kill Process • r: Refresh • q: Quit
```


##  Roadmap

- [ ] Health checking with HTTP ping and latency measurement
- [ ] Port filtering and search functionality
- [ ] Configuration file for custom port lists
- [ ] Export port snapshots to JSON/CSV
- [ ] Docker container detection
- [ ] Process resource usage (CPU/Memory)

## License

MIT License - see [LICENSE](LICENSE) for details

##  Acknowledgments

Built with amazing Go libraries from [Charm](https://charm.sh/) and inspired by modern developer tools like `lazygit` and `k9s`.

---

**made by [jun jiang](https://github.com/junj-st)**
