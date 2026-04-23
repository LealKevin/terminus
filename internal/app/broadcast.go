package app

import (
	"context"

	"github.com/LealKevin/terminus/internal/domain"
)

type MobUpdate struct {
	Type string        `json:"type"`
	Mobs []*domain.Mob `json:"mobs"`
}

func (h *Handler) BroadcastMobsUpdate(ctx context.Context, worldID string) error {
	response := MobUpdate{
		Type: "mobsUpdate",
		Mobs: h.Mobs.Snapshot(worldID),
	}

	conns := h.conns.snapshot()
	var failed []*clientConn
	for _, conn := range conns {
		if err := conn.sendJson(response); err != nil {
			failed = append(failed, conn)
		}
	}
	for _, conn := range failed {
		h.conns.remove(conn)
	}
	return nil
}
