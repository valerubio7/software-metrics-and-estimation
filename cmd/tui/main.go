package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func main() {
	config := tui.LoadConfig(os.Getenv)
	model := tui.NewModel(config.APIURL)
	if err := tui.ProbeAPI(config.APIURL); err != nil {
		model.APIErr = err
	}
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
