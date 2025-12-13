# k1s

A fast, keyboard-driven TUI for debugging Kubernetes workloads.

## Features

- Navigate deployments, statefulsets, daemonsets
- View pod logs with search and filtering
- Monitor events and resource metrics
- Scale and restart workloads
- Vim-style navigation

## Requirements

- Go 1.21+
- kubectl configured with cluster access

## Installation

### From Source

```bash
git clone https://github.com/andrebassi/k1s
cd k1s
make build
```

### Using Go

```bash
go install github.com/andrebassi/k1s/cmd/k1s@latest
```

## Usage

```bash
k1s
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBECONFIG` | Path to kubeconfig file | `~/.kube/config` |
| `K8SDEBUG_NAMESPACE` | Default namespace | `default` |

## Key Bindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `enter` | Select item |
| `esc` | Go back |
| `g` | Go to top |
| `G` | Go to bottom |

### Views

| Key | Action |
|-----|--------|
| `w` | Workloads view |
| `p` | Pods view |
| `l` | Logs view |
| `e` | Events view |
| `n` | Namespaces |

### Actions

| Key | Action |
|-----|--------|
| `r` | Refresh |
| `q` | Quit |
| `ctrl+u` | Page up (logs) |
| `ctrl+d` | Page down (logs) |

## Project Structure

```
k1s/
├── cmd/k1s/       # Main application entry point
├── internal/
│   ├── app/            # Application logic (bubbletea model)
│   ├── config/         # Configuration handling
│   ├── k8s/            # Kubernetes client operations
│   └── ui/             # TUI components (list, logs, styles)
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Development

### Build

```bash
make build
```

### Run

```bash
make run
```

### Clean

```bash
make clean
```

## Inspired by

- [k9sight](https://github.com/doganarif/k9sight) - Original TUI for Kubernetes debugging
- [k9s](https://k9scli.io/) - Kubernetes CLI to manage clusters

## License

MIT
