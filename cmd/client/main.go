package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type ClientMsg struct {
	PlayerID  string `json:"playerID"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Direction string `json:"direction"`
}

type ServerMsg struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Layout string `json:"layout,omitempty"`
	Msg    string `json:"msg,omitempty"`
	Mobs   []Mob  `json:"mobs"`
	World  world  `json:"world,omitempty"`
	Player Player `json:"player,omitempty"`
}

type Player struct {
	ID      string `json:"id"`
	WorldID string `json:"worldID"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
}

type errMsg struct{ error }

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

	return &connectionWrapper{
		conn:    conn,
		writer:  writer,
		encoder: encoder,
		decoder: decoder,
	}, nil
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
		fmt.Printf("Requesting world: %+v\n", msg)
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

func (cw *connectionWrapper) sendMove(dir string) tea.Cmd {
	return func() tea.Msg {
		msg := ClientMsg{PlayerID: "1", Type: "move", Direction: dir}
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

type world struct {
	ID     string
	Width  int
	Height int
	Layout [][]byte
}

type Entity struct {
	ID     string
	X      int
	Y      int
	Symbol rune
	Type   string
}

type Mob struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	WorldID     string `json:"worldID"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Type        string `json:"type"`
	Health      int    `json:"health"`
	Attack      int    `json:"attack"`
	Defense     int    `json:"defense"`
	AttackSpeed int    `json:"attackSpeed"`
	Symbol      rune   `json:"symbol"`
}

type GameState struct {
	world  world
	player Player
	mobs   []Mob
	items  []Entity
}

func (gs *GameState) copyWorldLayout() [][]rune {
	if len(gs.world.Layout) == 0 {
		return nil
	}

	display := make([][]rune, len(gs.world.Layout))
	for i, row := range gs.world.Layout {
		display[i] = make([]rune, len(row))
		for j, cell := range row {
			display[i][j] = rune(cell)
		}
	}
	return display
}

func (gs *GameState) Render() string {
	if len(gs.world.Layout) == 0 {
		return "Loading world...\n"
	}

	display := gs.copyWorldLayout()

	for _, item := range gs.items {
		if item.Y >= 0 && item.Y < len(display) &&
			item.X >= 0 && item.X < len(display[item.Y]) {
			display[item.Y][item.X] = item.Symbol
		}
	}

	for _, mob := range gs.mobs {
		if mob.Y >= 0 && mob.Y < len(display) &&
			mob.X >= 0 && mob.X < len(display[mob.Y]) {
			display[mob.Y][mob.X] = mob.Symbol
		}
	}

	if gs.player.Y >= 0 && gs.player.Y < len(display) &&
		gs.player.X >= 0 && gs.player.X < len(display[gs.player.Y]) {
		display[gs.player.Y][gs.player.X] = '@'
	} else {
		worldWidth := 0
		if len(display) > 0 {
			worldWidth = len(display[0])
		}
		fmt.Printf("DEBUG: Player not drawn - pos (%d, %d), world size (%d, %d)\n",
			gs.player.X, gs.player.Y, worldWidth, len(display))
	}

	var result string
	for _, row := range display {
		result += string(row) + "\n"
	}

	return result
}

type Model struct {
	gameState GameState
	conn      connectionWrapper
	err       error
	msgForNow string
}

func NewModel(conn connectionWrapper) Model {
	return Model{conn: conn}
}

func (m Model) Init() tea.Cmd {
	fmt.Print("Connecting to server...\n")
	var cmds []tea.Cmd
	cmds = append(cmds, m.conn.getWorld("world1"))
	cmds = append(cmds, m.conn.listenForServerMessages())

	return tea.Batch(cmds...)
}

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
		}

		return m, m.conn.listenForServerMessages()
	}
	return m, nil
}

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

func main() {
	conn, err := ServerConnection()
	if err != nil {
		panic(err)
	}
	p := tea.NewProgram(NewModel(*conn))
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err.Error())
		os.Exit(1)
	}
}
