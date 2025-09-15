package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/LealKevin/terminus/internal/domain"
)

type ClientMsg struct {
	PlayerID  string `json:"playerID"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Direction string `json:"direction"`
}

type Handler struct {
	Worlds domain.WorldStore
	Player domain.PlayerStore
}

func NewHandler(worldStore domain.WorldStore, playerStore domain.PlayerStore) *Handler {
	return &Handler{
		Worlds: worldStore,
		Player: playerStore,
	}
}

func (h *Handler) HandleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	fmt.Printf("New connection from: %v \n", conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)

	for {
		select {
		case <-ctx.Done():
			h.sendError(conn, fmt.Errorf("server is shutting down"))
			return
		default:
			line, err := reader.ReadBytes('\n')
			if err != nil {
				log.Printf("error reading from connection: %v", err)
				return
			}

			var msg ClientMsg
			err = json.Unmarshal(line, &msg)
			if err != nil {
				log.Printf("error unmarshaling json: %v", err)
				return
			}

			h.HandleMessage(ctx, msg, conn)
		}
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg ClientMsg, conn net.Conn) {
	switch msg.Type {
	case "move":
		err := h.HandlePlayerMove(ctx, msg.PlayerID, msg.Direction)
		if err != nil {
			h.sendError(conn, err)
		}
	}
}

type serverMsg struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
}

func (h *Handler) sendError(conn net.Conn, err error) error {
	response := serverMsg{
		Status: "error",
		Msg:    err.Error(),
	}
	e := json.NewEncoder(conn).Encode(response)
	if e != nil {
		return e
	}
	return nil
}

func (h *Handler) sendSuccess(conn net.Conn, msg string) error {
	response := serverMsg{
		Status: "sucess",
		Msg:    msg,
	}
	err := json.NewEncoder(conn).Encode(response)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) HandlePlayerMove(ctx context.Context, playerID string, dir string) error {
	player := h.Player.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found") // TODO: custom error
	}
	world := h.Worlds.GetWorld(player.WorldID)
	if world == nil {
		return fmt.Errorf("world not found") // TODO: custom error
	}
	err := player.Move(dir, world)
	if err != nil {
		return err
	}
	h.Player.SavePlayer(player)
	return nil
}
