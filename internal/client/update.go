package client

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg.error
		return m, m.conn.listenForServerMessages()

	case Player:
		m.msgForNow = fmt.Sprintf("Player moved to (%d, %d)", msg.X, msg.Y)
		m.gameState.player = msg
		return m, m.conn.listenForServerMessages()

	case ServerMsg:
		if msg.Type == "world" {
			m.gameState.world = msg.World
			return m, tea.Batch(
				m.conn.getPlayer("1"),
				m.conn.listenForServerMessages(),
			)
		}

		if msg.Type == "error" {
			m.msgForNow = "Cannot move: " + msg.Msg
			m.err = nil
			return m, m.conn.listenForServerMessages()
		}

		if msg.Type == "playerUpdate" {
			m.msgForNow = msg.Msg
			m.gameState.player = msg.Player
			return m, m.conn.listenForServerMessages()
		}

		if msg.Type == "mobsUpdate" {
			m.gameState.mobs = msg.Mobs
			return m, m.conn.listenForServerMessages()
		}

		if msg.Type == "success" {
			m.msgForNow = msg.Msg
			m.err = nil
			return m, m.conn.listenForServerMessages()
		}

		return m, m.conn.listenForServerMessages()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.msgForNow = "Quitting..."
			return m, tea.Quit
		case "up", "k":
			m.msgForNow = "Moving up"
			return m, tea.Batch(m.conn.sendMove("N"), m.conn.listenForServerMessages())
		case "down", "j":
			m.msgForNow = "Moving down"
			return m, tea.Batch(m.conn.sendMove("S"), m.conn.listenForServerMessages())
		case "left", "h":
			m.msgForNow = "Moving left"
			return m, tea.Batch(m.conn.sendMove("W"), m.conn.listenForServerMessages())
		case "right", "l":
			m.msgForNow = "Moving right"
			return m, tea.Batch(m.conn.sendMove("E"), m.conn.listenForServerMessages())
		case "y":
			m.msgForNow = "Moving up-left"
			return m, tea.Batch(m.conn.sendMove("NW"), m.conn.listenForServerMessages())
		case "u":
			m.msgForNow = "Moving up-right"
			return m, tea.Batch(m.conn.sendMove("NE"), m.conn.listenForServerMessages())
		case "b":
			m.msgForNow = "Moving down-left"
			return m, tea.Batch(m.conn.sendMove("SW"), m.conn.listenForServerMessages())
		case "n":
			m.msgForNow = "Moving down-right"
			return m, tea.Batch(m.conn.sendMove("SE"), m.conn.listenForServerMessages())
		case "a":
			m.msgForNow = "Attacking!"
			return m, tea.Batch(m.conn.sendAttack(), m.conn.listenForServerMessages())
		}

		return m, m.conn.listenForServerMessages()
	}
	return m, nil
}
