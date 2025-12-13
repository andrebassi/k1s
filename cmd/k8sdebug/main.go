package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k8sdebug/internal/app"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("k8sdebug version %s\n", version)
			os.Exit(0)
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		}
	}

	model, err := app.New()
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

func printHelp() {
	help := `k8sdebug - Kubernetes Manifest Debugger TUI

One screen to see why your pod is broken.

USAGE:
    k8sdebug [OPTIONS]

OPTIONS:
    -h, --help       Show this help message
    -v, --version    Show version information

KEYBOARD SHORTCUTS:
    Navigation:
        ↑/k          Move up
        ↓/j          Move down
        Enter        Select/drill down
        Esc          Go back
        Tab          Next panel
        Shift+Tab    Previous panel

    Actions:
        n            Change namespace
        t            Change resource type
        r            Refresh data
        /            Search
        *            Toggle favorite

    Dashboard:
        L            Focus logs panel
        E            Focus events panel
        M            Focus manifest panel
        m            Focus metrics panel
        F            Toggle log following
        e            Jump to next error
        w            Toggle all events

    General:
        ?            Show help
        q            Quit

CONFIGURATION:
    Config file: ~/.config/k8sdebug/config.json

For more information, visit: https://github.com/andrebassi/k8sdebug
`
	fmt.Println(help)
}
