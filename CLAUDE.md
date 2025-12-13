# k8sdebug - Project Documentation

## Project Overview

k8sdebug is a Terminal User Interface (TUI) application for debugging Kubernetes workloads. It provides a fast, keyboard-driven interface to navigate, inspect, and manage Kubernetes resources without leaving the terminal.

Inspired by [k9sight](https://github.com/doganarif/k9sight).

## Technology Stack

- **Language**: Go 1.21+
- **TUI Framework**: Charmbracelet (bubbletea, bubbles, lipgloss)
- **Kubernetes SDK**: client-go v0.29.0
- **Architecture**: Model-View-Update (MVU) pattern via bubbletea

## Project Structure

```
k8sdebug/
├── cmd/k8sdebug/
│   └── main.go           # Application entry point
├── internal/
│   ├── app/
│   │   ├── model.go      # Main application state and commands
│   │   ├── update.go     # Key handling and state updates
│   │   └── view.go       # UI rendering
│   ├── config/
│   │   └── config.go     # Configuration loading (kubeconfig, namespace)
│   ├── k8s/
│   │   ├── client.go     # Kubernetes client operations
│   │   └── types.go      # Data types and status helpers
│   └── ui/
│       ├── styles.go     # Lipgloss styles and colors
│       ├── list.go       # Generic list component
│       ├── logs.go       # Log viewer component
│       └── help.go       # Help bar component
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── CLAUDE.md
```

## Key Components

### app/model.go
- Main bubbletea Model struct holding application state
- View modes: Workloads, Pods, Logs, Events, Namespaces
- Commands for loading data from Kubernetes API
- Auto-refresh every 5 seconds

### app/update.go
- Keyboard event handling (Vim-style navigation)
- Message processing for async operations
- View switching logic

### app/view.go
- Header with context/namespace info
- Body content based on current view
- Help bar with key bindings

### k8s/client.go
- Kubernetes clientset wrapper
- Operations: GetNamespaces, GetPods, GetDeployments, GetStatefulSets, GetDaemonSets, GetEvents, GetPodLogs, DeletePod, ScaleDeployment

### ui/
- Reusable TUI components
- Consistent styling via lipgloss
- List with selection and scrolling
- Log viewer with search highlighting

## Configuration

Environment variables:
- `KUBECONFIG`: Path to kubeconfig file (default: `~/.kube/config`)
- `K8SDEBUG_NAMESPACE`: Initial namespace (default: `default`)

## Key Bindings

| Key | Action |
|-----|--------|
| j/k | Navigate up/down |
| enter | Select/drill down |
| esc | Go back |
| n | Switch namespace |
| w | Workloads view |
| p | Pods view |
| l | Logs view |
| e | Events view |
| r | Refresh |
| g/G | Top/bottom |
| q | Quit |

## Dependencies

Direct dependencies (go.mod):
- github.com/charmbracelet/bubbles v0.18.0
- github.com/charmbracelet/bubbletea v0.26.4
- github.com/charmbracelet/lipgloss v0.11.0
- k8s.io/api v0.29.0
- k8s.io/apimachinery v0.29.0
- k8s.io/client-go v0.29.0
- k8s.io/metrics v0.29.0

## Build & Run

```bash
# Download dependencies
make deps

# Build binary
make build

# Run directly
make run

# Install to GOPATH
make install
```

## Notes

- Uses bubbletea's Model-View-Update pattern
- Async operations via tea.Cmd
- Auto-refresh ticker for real-time updates
- Context timeout of 10s for all K8s operations
- Log viewer supports scroll, search highlighting, error/warn coloring

## Future Improvements

- [ ] Resource describe view
- [ ] Port forwarding
- [ ] Shell exec into pods
- [ ] Scale dialog
- [ ] Restart workloads
- [ ] Search/filter functionality
- [ ] Multiple container log selection
- [ ] Metrics display (CPU/Memory)
- [ ] Context switching

## Last Updated

2025-12-13
