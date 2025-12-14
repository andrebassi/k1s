# k1s

Kubernetes TUI Debugger - One screen to see why your pod is broken.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)

## Features

- Real-time container logs with filtering and error highlighting
- Pod events with Warning/Normal type filtering
- Resource metrics (CPU/Memory from metrics-server)
- HPA monitoring with KEDA support
- ConfigMaps/Secrets viewing and cross-namespace copy
- Navigate deployments, statefulsets, daemonsets, jobs, cronjobs
- Scale and restart workloads
- Vim-style keyboard navigation

## Installation

### Via Homebrew (macOS/Linux)

```bash
brew tap andrebassi/k1s
brew install k1s
```

### Via MacPorts (macOS)

```bash
sudo port install k1s
```

> **Note:** MacPorts submission pending. See [ports/sysutils/k1s](ports/sysutils/k1s/Portfile) for the Portfile.

### Download Binary

Download the latest release for your platform:

```bash
# macOS Apple Silicon
curl -L -o k1s https://github.com/andrebassi/k1s/releases/latest/download/k1s-darwin-arm64
chmod +x k1s
sudo mv k1s /usr/local/bin/

# macOS Intel
curl -L -o k1s https://github.com/andrebassi/k1s/releases/latest/download/k1s-darwin-amd64
chmod +x k1s
sudo mv k1s /usr/local/bin/

# Linux amd64
curl -L -o k1s https://github.com/andrebassi/k1s/releases/latest/download/k1s-linux-amd64
chmod +x k1s
sudo mv k1s /usr/local/bin/

# Linux arm64
curl -L -o k1s https://github.com/andrebassi/k1s/releases/latest/download/k1s-linux-arm64
chmod +x k1s
sudo mv k1s /usr/local/bin/

# Linux armv7 (Raspberry Pi)
curl -L -o k1s https://github.com/andrebassi/k1s/releases/latest/download/k1s-linux-armv7
chmod +x k1s
sudo mv k1s /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/andrebassi/k1s
cd k1s
task build
./bin/k1s
```

### Using Go

```bash
go install github.com/andrebassi/k1s/cmd/k1s@latest
```

## Requirements

- kubectl configured with cluster access
- metrics-server (optional, for CPU/Memory metrics)

## Usage

```bash
# Start k1s
k1s

# Start with specific namespace
k1s -n my-namespace

# Show version
k1s --version

# Show help
k1s --help
```

## Key Bindings

### Global Navigation

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Next/Previous section |
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` / `G` | Go to top/bottom |
| `Enter` | Select / Expand |
| `Esc` | Go back / Close |
| `r` | Refresh data |
| `?` | Show help |
| `q` | Quit |

### Resource Views

| Key | Action |
|-----|--------|
| `/` | Search / Filter |
| `c` | Clear filter |
| `a` | Actions menu |
| `d` | Delete (on Terminating namespaces) |

### Logs Panel

| Key | Action |
|-----|--------|
| `f` | Toggle follow mode |
| `e` | Jump to next error |
| `[` / `]` | Switch container |
| `T` | Cycle time filter |
| `P` | Previous container logs |

## Screenshots

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              k1s - Namespace: prod                           │
├─────────────────────────────────────────────────────────────────────────────┤
│ PODS                                                                         │
│ ▶ api-server-7d8f9c6b5-abc12    Running    1/1    0    2d                   │
│   worker-5f4e3d2c1-def34        Running    1/1    5    1d                   │
│   scheduler-9a8b7c6d-ghi56      Running    1/1    0    3d                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ HPA                                                                          │
│   api-server    Deployment/api-server    cpu: 45%/80%    2    10    3       │
├─────────────────────────────────────────────────────────────────────────────┤
│ CONFIGMAPS                                                                   │
│   app-config                    3 keys                                       │
│   nginx-config                  1 key                                        │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Development

```bash
# Build
task build

# Run
task run

# Clean
task clean

# Format code
task fmt

# Run tests
task test
```

## Inspired by

- [k9s](https://k9scli.io/) - Kubernetes CLI to manage clusters
- [k9sight](https://github.com/doganarif/k9sight) - Original TUI for Kubernetes debugging

## License

MIT
