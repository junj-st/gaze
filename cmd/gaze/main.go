package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/junjiang/gaze/internal/ui"
)

func main() {
	// Create the Bubble Tea program
	p := tea.NewProgram(ui.InitialModel(), tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running gaze: %v\n", err)
		os.Exit(1)
	}
}
