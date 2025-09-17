package client

import (
	"fmt"
)

func (m Model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	s := m.msgForNow + "\n"
	s += fmt.Sprintf("World ID: %s (Width: %d, Height: %d)\n", m.gameState.world.ID, m.gameState.world.Width, m.gameState.world.Height)
	s += fmt.Sprintf("Player: (%d, %d)\n", m.gameState.player.X, m.gameState.player.Y)
	s += fmt.Sprintf("Entities: %d items, %d mobs\n", len(m.gameState.items), len(m.gameState.mobs))
	s += "\n"

	s += m.gameState.Render()

	return s
}
