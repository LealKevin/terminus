package main

import (
	"fmt"
	"os"

	"github.com/LealKevin/terminus/internal/client"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	conn, err := client.ServerConnection()
	if err != nil {
		panic(err)
	}
	p := tea.NewProgram(client.NewModel(*conn))
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err.Error())
		os.Exit(1)
	}
}
