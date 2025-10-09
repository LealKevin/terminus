package client

import (
	"bufio"
	"encoding/json"
	"net"

	tea "github.com/charmbracelet/bubbletea"
)

type connectionWrapper struct {
	conn    net.Conn
	writer  *bufio.Writer
	encoder *json.Encoder
	decoder *json.Decoder
}

func ServerConnection() (*connectionWrapper, error) {
	port := "4200"
	addr := "localhost:" + port
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}

	writer := bufio.NewWriter(conn)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	decoder := json.NewDecoder(conn)

	cw := &connectionWrapper{
		conn:    conn,
		writer:  writer,
		encoder: encoder,
		decoder: decoder,
	}

	// Don't send initial message - let the server set up the player first

	return cw, nil
}

func (cw *connectionWrapper) listenForServerMessages() tea.Cmd {
	return func() tea.Msg {
		var msg ServerMsg
		err := cw.decoder.Decode(&msg)
		if err != nil {
			return errMsg{err}
		}
		return msg
	}
}

func (cw *connectionWrapper) getWorld(id string) tea.Cmd {
	return func() tea.Msg {
		msg := ClientMsg{Type: "getWorld", Message: id}
		data, err := json.Marshal(msg)
		if err != nil {
			return errMsg{err}
		}
		_, err = cw.conn.Write(append(data, '\n'))
		if err != nil {
			return errMsg{err}
		}

		return nil
	}
}

func (cw *connectionWrapper) getPlayer(id string) tea.Cmd {
	return func() tea.Msg {
		msg := ClientMsg{Type: "getPlayer", PlayerID: id}
		data, err := json.Marshal(msg)
		if err != nil {
			return errMsg{err}
		}
		_, err = cw.conn.Write(append(data, '\n'))
		if err != nil {
			return errMsg{err}
		}

		var player Player
		err = cw.decoder.Decode(&player)
		if err != nil {
			return errMsg{err}
		}

		return player
	}
}

func (cw *connectionWrapper) sendMove(playerID, dir string) tea.Cmd {
	return func() tea.Msg {
		msg := ClientMsg{PlayerID: playerID, Type: "move", Direction: dir}
		data, err := json.Marshal(msg)
		if err != nil {
			return errMsg{err}
		}
		_, err = cw.conn.Write(append(data, '\n'))
		if err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (cw *connectionWrapper) sendAttack(playerID string) tea.Cmd {
	return func() tea.Msg {
		msg := ClientMsg{PlayerID: playerID, Type: "attack"}
		data, err := json.Marshal(msg)
		if err != nil {
			return errMsg{err}
		}
		_, err = cw.conn.Write(append(data, '\n'))
		if err != nil {
			return errMsg{err}
		}
		return nil
	}
}
