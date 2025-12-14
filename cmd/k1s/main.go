// Package main is the entry point for k1s, a Kubernetes Pod Debugger TUI.
//
// k1s provides a terminal-based user interface for debugging Kubernetes
// workloads, offering real-time logs, events, metrics, and resource
// inspection in a single dashboard view.
//
// Usage:
//
//	k1s [options]
//
// Options:
//
//	-h, --help         Show help message
//	-v, --version      Show version information
//	-n, --namespace    Go directly to resources view for specified namespace
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k1s/internal/app"
)

// version defines the current version of k1s.
const version = "0.1.0"

// main initializes and runs the k1s TUI application.
// It parses command-line arguments for namespace selection and help/version flags,
// then starts the bubbletea program with alternate screen and mouse support.
func main() {
	var namespace string

	// Parse command-line arguments manually to avoid external dependencies.
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--version", "-v":
			fmt.Printf("k1s version %s\n", version)
			os.Exit(0)
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		case "-n", "--namespace":
			if i+1 < len(os.Args) {
				namespace = os.Args[i+1]
				i++ // Skip the next argument
			} else {
				fmt.Fprintf(os.Stderr, "Error: -n/--namespace requires an argument\n")
				os.Exit(1)
			}
		default:
			// Check for -n=value format
			if len(os.Args[i]) > 3 && os.Args[i][:3] == "-n=" {
				namespace = os.Args[i][3:]
			} else if len(os.Args[i]) > 12 && os.Args[i][:12] == "--namespace=" {
				namespace = os.Args[i][12:]
			} else {
				fmt.Fprintf(os.Stderr, "Unknown option: %s\n", os.Args[i])
				fmt.Fprintf(os.Stderr, "Use -h for help\n")
				os.Exit(1)
			}
		}
	}

	model, err := app.NewWithOptions(app.Options{
		Namespace: namespace,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

// printHelp displays the comprehensive help message including usage,
// keyboard shortcuts, features, and configuration options.
func printHelp() {
	help := `k1s - Kubernetes Pod Debugger TUI

One screen to see why your pod is broken.

USAGE:
    k1s [OPTIONS]

OPTIONS:
    -h, --help            Show this help message
    -v, --version         Show version information
    -n, --namespace NS    Go directly to resources view for namespace NS

DASHBOARD LAYOUT:
    ┌─────────────────────┬─────────────────────┐
    │        Logs         │       Events        │
    │  (container logs)   │   (pod events)      │
    ├─────────────────────┼─────────────────────┤
    │     Pod Details     │   Resource Usage    │
    │   (info + related)  │  (CPU/Mem + Node)   │
    └─────────────────────┴─────────────────────┘

KEYBOARD SHORTCUTS:

  Global Navigation:
    Tab/Shift+Tab    Next/Previous panel
    ←/→/↑/↓          Arrow key navigation
    j/k              Vim-style up/down
    g/G              Go to top/bottom
    Enter            Select / Expand / Copy
    Esc              Go back / Close
    1-4              Focus panel directly
    F                Toggle fullscreen
    r                Refresh data
    ?                Show help
    q                Quit

  Logs Panel:
    f                Toggle follow mode
    /                Search/filter logs
    c                Clear filter
    e                Jump to next error
    [/]              Switch container (multi-container pods)
    T                Cycle time filter (All, 5m, 15m, 1h, 6h)
    P                Toggle previous container logs
    Enter            Fullscreen → Enter again to copy

  Events Panel:
    w                Toggle warnings only
    Enter            Fullscreen → Enter again to copy

  Pod Details Panel:
    Enter            Show Resource Details view
                     (Pod Info, Network, Services, Istio, etc.)

  Resource Usage Panel:
    ←/→              Switch between Container/Node columns
    ↑/↓/j/k          Scroll within column
    Enter            Show kubectl describe output

  Action Menus:
    a                Pod actions (delete, exec, port-forward, describe)
    y                Copy kubectl command to clipboard

FEATURES:
    • Real-time container logs with filtering and error highlighting
    • Pod events with Warning/Normal type filtering
    • Resource metrics (CPU/Memory usage from metrics-server)
    • Node information display
    • Istio VirtualServices and Gateways detection
    • Related resources (Services, Ingresses, ConfigMaps, Secrets)
    • Workload owner chain (Pod → ReplicaSet → Deployment)
    • Clipboard copy support (logs, events, resource details)
    • Multi-container pod support

CONFIGURATION:
    Config file: ~/.config/k1s/config.json
    Environment:
      KUBECONFIG        Path to kubeconfig (default: ~/.kube/config)
      K1S_NAMESPACE     Initial namespace (default: default)

For more information, visit: https://github.com/andrebassi/k1s
`
	fmt.Println(help)
}
