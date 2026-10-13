package client

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg.error
		return m, m.conn.listenForServerMessages()

	case ServerMsg:
		if msg.Type == "world" {
			m.gameState.world = msg.World
			return m, m.conn.listenForServerMessages()
		}

		if msg.Type == "error" {
			m.msgForNow = "Cannot move: " + msg.Msg
			m.err = nil
			return m, m.conn.listenForServerMessages()
		}

		if msg.Type == "playerUpdate" {
			m.msgForNow = msg.Msg
			m.gameState.player = msg.Player
			m.playerID = msg.Player.ID // Store the player ID from server
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
			return m, m.conn.sendMove(m.playerID, "N")
		case "down", "j":
			m.msgForNow = "Moving down"
			return m, m.conn.sendMove(m.playerID, "S")
		case "left", "h":
			m.msgForNow = "Moving left"
			return m, m.conn.sendMove(m.playerID, "W")
		case "right", "l":
			m.msgForNow = "Moving right"
			return m, m.conn.sendMove(m.playerID, "E")
		case "y":
			m.msgForNow = "Moving up-left"
			return m, m.conn.sendMove(m.playerID, "NW")
		case "u":
			m.msgForNow = "Moving up-right"
			return m, m.conn.sendMove(m.playerID, "NE")
		case "b":
			m.msgForNow = "Moving down-left"
			return m, m.conn.sendMove(m.playerID, "SW")
		case "n":
			m.msgForNow = "Moving down-right"
			return m, m.conn.sendMove(m.playerID, "SE")
		case "a":
			m.msgForNow = "Attacking!"
			return m, m.conn.sendAttack(m.playerID)
		}

		return m, nil
	}
	return m, nil
}
