package client

import (
	"fmt"

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
	World  world  `json:"world"`
	Player Player `json:"player"`
}

type Player struct {
	ID      string `json:"id"`
	WorldID string `json:"worldID"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
}

type errMsg struct{ error }

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
