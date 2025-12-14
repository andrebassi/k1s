# k1s

Kubernetes TUI Debugger - **One screen to see why your pod is broken.**

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)
![Platform](https://img.shields.io/badge/platform-macOS%20|%20Linux%20|%20Windows%20|%20Android-lightgrey.svg)

## Overview

k1s is a terminal-based user interface (TUI) for debugging Kubernetes workloads. It provides real-time logs, events, metrics, and resource inspection in a single dashboard view.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              k1s - Namespace: prod                           │
├─────────────────────────────────────────────────────────────────────────────┤
│ PODS                                                                         │
│ ▶ api-server-7d8f9c6b5-abc12    Running    1/1    0    2d                   │
│   worker-5f4e3d2c1-def34        Running    1/1    5    1d                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ HPA                                                                          │
│   api-server    Deployment/api-server    cpu: 45%/80%    2    10    3       │
├─────────────────────────────────────────────────────────────────────────────┤
│ CONFIGMAPS                           │ SECRETS                              │
│   app-config         3 keys          │   db-credentials    Opaque    2 keys │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Features

### Pod Debugging Dashboard
4-panel layout showing everything you need:
```
┌─────────────────────┬─────────────────────┐
│        Logs         │       Events        │
│  (container logs)   │   (pod events)      │
├─────────────────────┼─────────────────────┤
│     Pod Details     │   Resource Usage    │
│   (info + related)  │  (CPU/Mem + Node)   │
└─────────────────────┴─────────────────────┘
```

### Resource Management
- **Pods**: Status, Ready count, Restarts, Age
- **HPAs**: Reference, Targets (CPU/Memory/External/KEDA), Min/Max/Current Replicas
- **ConfigMaps**: Key count, Age, full data viewing
- **Secrets**: Type, Key count, base64-decoded viewing
- **Docker Registry**: Registry credentials viewing

### Workload Operations
- Support for: Deployments, StatefulSets, DaemonSets, Jobs, CronJobs, Argo Rollouts
- Scale up/down workloads
- Rolling restart with confirmation
- Delete pods

### Namespace Management
- List all namespaces with status (Active/Terminating)
- Color-coded status indicators
- Force delete stuck Terminating namespaces
- Split view with Nodes panel

### Cross-Namespace Copy
- Copy ConfigMaps to single namespace or all namespaces
- Copy Secrets to single namespace or all namespaces
- Copy Docker Registry secrets
- Progress indicator during batch operations

### Additional Features
- Real-time container logs with filtering and error highlighting
- Pod events with Warning/Normal type filtering
- Resource metrics (CPU/Memory from metrics-server)
- Istio VirtualServices and Gateways detection
- Related resources discovery (Services, Ingresses)
- Clipboard support for copying values
- Vim-style keyboard navigation

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/andrebassi/k1s/main/scripts/install.sh | bash
```

> **Note:** The install script automatically detects your OS/architecture and installs kubectl if not found.

### Via Homebrew (macOS/Linux)

```bash
brew tap andrebassi/k1s
brew install k1s
```

### Via Chocolatey (Windows)

```powershell
choco install k1s
```

### Via Scoop (Windows)

```powershell
scoop bucket add k1s https://github.com/andrebassi/scoop-k1s
scoop install k1s
```

### Via AUR (Arch Linux)

```bash
# Using yay
yay -S k1s-bin

# Using paru
paru -S k1s-bin
```

> **Note:** AUR submission pending. See [aur/k1s-bin](aur/k1s-bin/PKGBUILD) for the PKGBUILD.

### Via Termux (Android)

```bash
# Install via curl (recommended)
curl -sSL https://raw.githubusercontent.com/andrebassi/k1s/main/scripts/install.sh | bash

# Or download directly (PIE binary for Android)
curl -L -o $PREFIX/bin/k1s https://github.com/andrebassi/k1s/releases/latest/download/k1s-android-arm64
chmod +x $PREFIX/bin/k1s
```

**Prerequisites for Termux:**
```bash
# Install required packages
pkg install curl kubectl

# Configure kubectl with your cluster
# Copy your kubeconfig to ~/.kube/config
```

> **Note:** The `k1s-android-arm64` binary is built with PIE (Position Independent Executable) support required by Android 5.0+.

### Via apt-get (Debian/Ubuntu)

```bash
# Download the .deb package (x86_64)
curl -LO https://github.com/andrebassi/k1s/releases/latest/download/k1s_VERSION_amd64.deb

# Or for ARM64
curl -LO https://github.com/andrebassi/k1s/releases/latest/download/k1s_VERSION_arm64.deb

# Install
sudo apt install ./k1s_*.deb
```

### Via yum/dnf (RHEL/Fedora/CentOS)

```bash
# Download the .rpm package (x86_64)
curl -LO https://github.com/andrebassi/k1s/releases/latest/download/k1s-VERSION-1.amd64.rpm

# Or for ARM64
curl -LO https://github.com/andrebassi/k1s/releases/latest/download/k1s-VERSION-1.arm64.rpm

# Install with yum
sudo yum localinstall ./k1s-*.rpm

# Or with dnf
sudo dnf install ./k1s-*.rpm
```

### Via MacPorts (macOS)

```bash
sudo port install k1s
```

> **Note:** MacPorts submission pending. See [ports/sysutils/k1s](ports/sysutils/k1s/Portfile) for the Portfile.

### Download Binary

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Apple Silicon (arm64) | [k1s-darwin-arm64](https://github.com/andrebassi/k1s/releases/latest/download/k1s-darwin-arm64) |
| macOS | Intel (amd64) | [k1s-darwin-amd64](https://github.com/andrebassi/k1s/releases/latest/download/k1s-darwin-amd64) |
| Linux | x86_64 (amd64) | [k1s-linux-amd64](https://github.com/andrebassi/k1s/releases/latest/download/k1s-linux-amd64) |
| Linux | ARM64 | [k1s-linux-arm64](https://github.com/andrebassi/k1s/releases/latest/download/k1s-linux-arm64) |
| Linux | ARMv7 (Raspberry Pi) | [k1s-linux-armv7](https://github.com/andrebassi/k1s/releases/latest/download/k1s-linux-armv7) |
| Android | Termux (arm64) | [k1s-android-arm64](https://github.com/andrebassi/k1s/releases/latest/download/k1s-android-arm64) |
| Windows | x86_64 (amd64) | [k1s-windows-amd64.exe](https://github.com/andrebassi/k1s/releases/latest/download/k1s-windows-amd64.exe) |

```bash
# Example: macOS Apple Silicon
curl -L -o k1s https://github.com/andrebassi/k1s/releases/latest/download/k1s-darwin-arm64
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

- kubectl configured with cluster access (auto-installed by install script if missing)
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

## Keyboard Shortcuts

### Global
| Key | Action |
|-----|--------|
| `?` | Help |
| `q`, `Ctrl+C` | Quit |
| `r` | Refresh |
| `Esc` | Back/Close |
| `Enter` | Select/Expand |
| `Tab`/`Shift+Tab` | Next/Previous section |
| `↑`/`↓` or `j`/`k` | Navigate |
| `/` | Search/Filter |
| `c` | Clear filter |

### Namespace View
| Key | Action |
|-----|--------|
| `d` | Delete Terminating namespace |
| `Enter` | Select namespace (or delete if Terminating) |
| `←`/`→` | Switch between Namespace/Nodes panels |

### Resources View
| Key | Action |
|-----|--------|
| `Tab` | Cycle sections (Pods → HPA → ConfigMaps → Secrets → Docker Registry) |
| `Enter` | Open viewer/dashboard for selected item |
| `a` | Actions menu |

### Viewers (ConfigMap, Secret, HPA)
| Key | Action |
|-----|--------|
| `↑`/`↓` | Scroll/Navigate |
| `Enter` | Copy selected value to clipboard |
| `a` | Actions menu (copy to namespace) |
| `g`/`G` | Go to top/bottom |
| `Esc`/`q` | Close |

### Logs Panel
| Key | Action |
|-----|--------|
| `f` | Toggle follow mode |
| `/` | Search/filter logs |
| `e` | Jump to next error |
| `[`/`]` | Switch container |
| `T` | Cycle time filter (All, 5m, 15m, 1h, 6h) |
| `P` | Toggle previous container logs |

## Configuration

Config file: `~/.config/k1s/configs.json`

```json
{
  "lastNamespace": "default",
  "lastResourceType": "deployments",
  "refreshInterval": 5
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBECONFIG` | Path to kubeconfig file | `~/.kube/config` |
| `K1S_NAMESPACE` | Initial namespace | `default` |

## Development

```bash
# Build
task build

# Run
task run

# Install to /usr/local/bin
task install

# Clean build artifacts
task clean

# Format code
task fmt

# Run tests
task test

# Create a release
task release -- v0.2.0
```

### Testing Installation

```bash
# Test Homebrew installation
task test:brew

# Test MacPorts installation
task test:macports

# Test all Docker installations (Debian, Ubuntu, Fedora, Alpine)
task test:docker:all

# Test specific platform
task test:docker:debian
task test:docker:ubuntu
task test:docker:fedora
task test:docker:alpine

# Test local macOS installation
task test:local
```

## Technology Stack

- **Language**: Go 1.21+
- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Kubernetes Client**: [client-go](https://github.com/kubernetes/client-go)
- **Metrics**: [metrics-server client](https://github.com/kubernetes-sigs/metrics-server)

## Project Structure

```
k1s/
├── cmd/k1s/              # Entry point, CLI parsing
├── configs/              # Configuration management
├── internal/
│   ├── adapters/
│   │   ├── repository/   # Kubernetes API interactions
│   │   └── tui/          # Terminal UI components
│   │       ├── component/  # Reusable UI components
│   │       ├── view/       # View layouts
│   │       ├── keys/       # Keyboard shortcuts
│   │       └── style/      # Styling
│   ├── domain/           # Domain entities and interfaces
│   └── usecase/          # Business logic
├── scripts/              # Install and test scripts
├── chocolatey/           # Chocolatey package (Windows)
├── aur/                  # AUR package (Arch Linux)
├── termux/               # Termux package (Android)
├── ports/                # MacPorts package (macOS)
├── .github/workflows/    # CI/CD release workflow
├── Taskfile.yaml         # Task runner configuration
└── go.mod
```

## HPA Metrics Support

k1s supports multiple HPA metric types:
- **Resource**: CPU, Memory utilization
- **External**: KEDA metrics, custom metrics
- **Pods**: Pod-level metrics
- **Object**: Object-based metrics

## Inspired by

- [k9s](https://k9scli.io/) - Kubernetes CLI to manage clusters
- [k9sight](https://github.com/doganarif/k9sight) - TUI for Kubernetes debugging

## License

MIT
