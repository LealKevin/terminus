package app

import (
	"context"
	"fmt"

	"github.com/LealKevin/terminus/internal/domain"
)

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
