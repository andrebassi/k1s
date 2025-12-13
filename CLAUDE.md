# k1s - Project Documentation

## Project Overview

k1s is a Terminal User Interface (TUI) application for debugging Kubernetes workloads. It provides a fast, keyboard-driven interface to navigate, inspect, and manage Kubernetes resources without leaving the terminal.

Inspired by [k9sight](https://github.com/doganarif/k9sight).

## Technology Stack

- **Language**: Go 1.21+
- **TUI Framework**: Charmbracelet (bubbletea, bubbles, lipgloss)
- **Kubernetes SDK**: client-go v0.29.0
- **Architecture**: Model-View-Update (MVU) pattern via bubbletea

## Features

### Dashboard View (4-Panel Layout)
The main dashboard displays pod information in a 2x2 grid:

```
┌─────────────────────┬─────────────────────┐
│        Logs         │       Events        │
│  (container logs)   │   (pod events)      │
├─────────────────────┼─────────────────────┤
│   Resource Usage    │     Pod Details     │
│ (CPU/Mem + Node)    │  (info + manifest)  │
└─────────────────────┴─────────────────────┘
```

### Logs Panel
- Real-time container logs with auto-follow
- Multi-container support with `[`/`]` to switch containers
- Time filter (`T` to cycle: All, 5m, 15m, 1h, 6h)
- Search/filter with `/` (live search as you type)
- Error highlighting in red
- Jump to next error with `e`
- **Fullscreen mode**: Press `Enter` to expand
- **Copy to clipboard**: Press `Enter` in fullscreen mode

### Events Panel
- Pod events with Warning/Normal highlighting
- Toggle warnings only with `w`
- Fullscreen toggle with `Enter`
- **Copy to clipboard**: Press `Enter` in fullscreen mode

### Resource Usage Panel
- Two-column layout:
  - **Left box**: Container Resources (CPU/Memory requests/limits, usage)
  - **Right box**: Node Info (name, status, version, IP, pod count)
- Navigate between boxes with `←`/`→` arrows
- Scroll within boxes with `↑`/`↓` or `j`/`k`
- **Detailed Resource View**: Press `Enter` to open full resource details

### Resource Details View (Enter on Resource Usage)
Shows comprehensive pod information:
- Pod Info (QoS, Service Account, Restart Policy, DNS Policy, etc.)
- Network (Pod IP, Host IP, Node, Started time)
- Services (related services with Type, ClusterIP, Ports, Endpoints)
- **Istio VirtualServices** (hosts, gateways, routes with destinations)
- **Istio Gateways** (servers, ports, protocols, TLS mode)
- Tolerations
- Container details (image, state, probes, resources, ports, mounts)
- Volumes, ConfigMaps, Secrets used
- **Copy to clipboard**: Press `Enter` to copy as clean markdown

### Pod Details Panel
- Summary, Details, and Resources view modes (cycle with `d`)
- Owner reference
- Labels and conditions
- Debug hints for common issues

### Resources View (Navigator)
Browse cluster resources by type:
- **Pods**: List all pods with status indicators
- **ConfigMaps**: View and copy ConfigMap data
- **Secrets**: View decoded secrets (base64 decoded)
- **Docker Registry**: View image pull secrets

Navigate with:
- `↑`/`↓` or `j`/`k`: Move within section
- `Tab`/`Shift+Tab`: Switch sections
- `Enter`: View details / Copy value to clipboard

### Namespace & Node Selection
- Side-by-side panels for namespace and node selection
- `←`/`→` arrows to switch between panels
- Filter namespaces/nodes as you type

## Key Bindings

### Global Navigation
| Key | Action |
|-----|--------|
| `Tab` | Next panel / section |
| `Shift+Tab` | Previous panel / section |
| `←`/`→`/`↑`/`↓` | Arrow key navigation |
| `j`/`k` | Vim-style up/down |
| `g`/`G` | Go to top/bottom |
| `Enter` | Select / Expand / Copy |
| `Esc` | Go back / Close |
| `q` | Quit |

### Panel-Specific
| Key | Panel | Action |
|-----|-------|--------|
| `f` | Logs | Toggle follow mode |
| `/` | Logs | Search/filter |
| `c` | Logs | Clear filter |
| `e` | Logs | Jump to next error |
| `[`/`]` | Logs | Switch container |
| `T` | Logs | Cycle time filter |
| `P` | Logs | Toggle previous logs |
| `w` | Events | Toggle warnings only |
| `d` | Pod Details | Cycle view mode |
| `1`-`4` | Dashboard | Focus panel directly |
| `F` | Dashboard | Toggle fullscreen |

### Action Keys
| Key | Action |
|-----|--------|
| `a` | Pod actions menu (delete, exec, port-forward, describe) |
| `y` | Copy kubectl command menu |
| `?` | Show help |
| `r` | Refresh |

## Project Structure

```
k1s/
├── cmd/k1s/
│   └── main.go              # Application entry point
├── internal/
│   ├── app/
│   │   └── app.go           # Main application model, update, view
│   ├── config/
│   │   └── config.go        # Configuration loading
│   ├── k8s/
│   │   ├── client.go        # Kubernetes client (clientset + dynamic client)
│   │   ├── resources.go     # Resource fetching and types
│   │   └── types.go         # Data types and helpers
│   └── ui/
│       ├── components/
│       │   ├── logs.go          # Logs panel with copy support
│       │   ├── events.go        # Events panel with copy support
│       │   ├── metrics.go       # Resource usage (two-box layout)
│       │   ├── manifest.go      # Pod details panel
│       │   ├── navigator.go     # Resource browser
│       │   ├── result_viewer.go # Modal result display with copy
│       │   ├── configmap_viewer.go
│       │   ├── secret_viewer.go
│       │   ├── clipboard.go     # Cross-platform clipboard
│       │   ├── action_menu.go   # Pod actions and kubectl commands
│       │   └── ...
│       ├── views/
│       │   └── dashboard.go     # Main dashboard (4-panel + fullscreen)
│       └── styles/
│           └── styles.go        # Lipgloss styles and colors
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Key Components

### internal/k8s/client.go
- Kubernetes clientset wrapper
- **Dynamic client** for fetching Istio CRDs (VirtualServices, Gateways)
- Metrics client for CPU/Memory usage

### internal/k8s/resources.go
- `GetRelatedResources()`: Fetches Services, Ingresses, VirtualServices, Gateways, ConfigMaps, Secrets related to a pod
- `getIstioResources()`: Dynamic client fetch for Istio VirtualServices and Gateways
- Workload management (Deployments, StatefulSets, DaemonSets, Jobs, CronJobs)

### internal/ui/views/dashboard.go
- 4-panel dashboard layout
- Fullscreen mode for Logs/Events
- Arrow key navigation between panels
- Handles Enter key for different actions per panel
- Renders detailed resource view with Istio info

### internal/ui/components/result_viewer.go
- Modal overlay for displaying content (describe output, resource details)
- **Copy to clipboard**: Enter key copies content as clean markdown
- Strips ANSI codes for clean clipboard content
- Scroll support with j/k and g/G

### internal/ui/components/logs.go
- Container log display with filtering
- Time-based filtering (5m, 15m, 1h, 6h)
- **Copy to clipboard**: Enter key in fullscreen copies logs as plain text
- Multi-container support

### internal/ui/components/metrics.go
- Two-box layout (Container Resources | Node Info)
- Independent scroll for each box
- Focus switching with left/right arrows

## Istio Integration

The app detects and displays Istio resources when available:

### VirtualServices
- Name and associated hosts
- Gateways referenced
- HTTP routes with match conditions and destinations
- Traffic weights for canary deployments

### Gateways
- Name and namespace
- Server configurations (port, protocol)
- TLS mode (SIMPLE, MUTUAL, PASSTHROUGH)
- Host patterns

## Clipboard Copy Feature

Multiple ways to copy data:

1. **Resource Details** (Enter on Resource Usage → Enter again)
   - Copies full resource info as markdown
   - Strips ANSI color codes
   - Includes Pod Info, Network, Services, VirtualServices, Gateways, etc.

2. **Logs** (Enter to fullscreen → Enter again)
   - Copies filtered logs as plain text
   - Respects current container and time filter

3. **Events** (Enter to fullscreen → Enter again)
   - Copies events as plain text
   - Respects warnings-only filter

4. **ConfigMap/Secret Values** (Enter on selected key)
   - Copies the value directly

5. **kubectl Commands** (`y` key)
   - Menu of common kubectl commands to copy

## Configuration

Command-line flags:
- `-n, --namespace NS`: Go directly to resources view for namespace NS
- `-h, --help`: Show help message
- `-v, --version`: Show version information

Environment variables:
- `KUBECONFIG`: Path to kubeconfig file (default: `~/.kube/config`)
- `K1S_NAMESPACE`: Initial namespace (default: `default`)

## Build & Run

```bash
# Build binary
go build -o bin/k1s ./cmd/k1s

# Run
./bin/k1s

# Run with namespace (goes directly to resources view)
./bin/k1s -n api
```

## Dependencies

- github.com/charmbracelet/bubbles v0.18.0
- github.com/charmbracelet/bubbletea v0.26.4
- github.com/charmbracelet/lipgloss v0.11.0
- k8s.io/api v0.29.0
- k8s.io/apimachinery v0.29.0
- k8s.io/client-go v0.29.0
- k8s.io/metrics v0.29.0

## Development History

1. **Initial Commit** - Basic TUI structure with bubbletea
2. **Fullscreen toggle** - Enter key expands Logs/Events panels
3. **Resources view** - ConfigMaps and Secrets browser with viewer modals
4. **Clipboard copy** - Copy ConfigMap/Secret values with Enter
5. **Node info display** - Two-column Resource Usage with Node Info
6. **Arrow navigation** - Full arrow key support across all panels
7. **Istio integration** - VirtualServices and Gateways display with dynamic client
8. **Copy features** - Resource Details and Logs copy to clipboard
9. **Namespace flag** - `-n namespace` flag to go directly to resources view

## Last Updated

2025-12-13
