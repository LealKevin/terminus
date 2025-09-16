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
			log.Printf("Received message: %+v", msg)

			h.HandleMessage(ctx, msg, conn)
		}
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg ClientMsg, conn net.Conn) {
	switch msg.Type {
	case "getPlayer":
		player := h.Player.GetPlayer(msg.PlayerID)
		if player == nil {
			err := fmt.Errorf("player not found")
			log.Printf("error retrieving player: %v", err)
			h.sendError(conn, err)
			return
		}
		response, err := json.Marshal(player)
		if err != nil {
			log.Printf("error marshaling player: %v", err)
			h.sendError(conn, err)
			return
		}
		_, err = conn.Write(append(response, '\n'))
		if err != nil {
			log.Printf("error sending player data: %v", err)
			h.sendError(conn, err)
			return
		}

	case "move":
		err := h.HandlePlayerMove(ctx, conn, msg.PlayerID, msg.Direction)
		if err != nil {
			log.Printf("error handling player move: %v", err)
			h.sendError(conn, err)
		}

	case "getWorld":
		err := h.HandleSendWorld(ctx, conn, msg.Message)
		if err != nil {
			log.Printf("error sending world: %v", err)
			h.sendError(conn, err)
		} else {
			log.Printf("World %s sent successfully", msg.Message)
		}
	}
}

type serverMsg struct {
	Type string `json:"type"`
	Msg  string `json:"msg"`
}

func (h *Handler) sendError(conn net.Conn, err error) error {
	response := serverMsg{
		Type: "error",
		Msg:  err.Error(),
	}
	e := json.NewEncoder(conn).Encode(response)
	if e != nil {
		return e
	}
	return nil
}

func (h *Handler) sendSuccess(conn net.Conn, msg string) error {
	response := serverMsg{
		Type: "success",
		Msg:  msg,
	}
	err := json.NewEncoder(conn).Encode(response)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) HandleSendWorld(ctx context.Context, conn net.Conn, worldID string) error {
	fmt.Printf("Sending world %s\n", worldID)
	world := h.Worlds.GetWorld(worldID)
	data, err := json.Marshal(world)
	if err != nil {
		return err
	}
	fmt.Printf("World data: %s\n", string(data))
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) HandlePlayerMove(ctx context.Context, conn net.Conn, playerID string, dir string) error {
	player := h.Player.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found") // TODO: custom error
	}
	world := h.Worlds.GetWorld(player.WorldID)
	if world == nil {
		return fmt.Errorf("world not found") // TODO: custom error
	}
	err := player.Move(dir, world)
	log.Printf("Player %s moved to (%d, %d) in world %s", player.ID, player.X, player.Y, world.ID)
	if err != nil {
		return err
	}
	h.Player.SavePlayer(player)
	response, err := json.Marshal(player)
	if err != nil {
		return err
	}
	fmt.Printf("Sending player data: %s\n", string(response))
	_, err = conn.Write(append(response, '\n'))
	if err != nil {
		return err
	}

	return nil
}
