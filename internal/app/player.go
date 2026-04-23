package app

import (
	"context"
	"fmt"
	"log"
	"time"
)

func (h *Handler) HandlePlayerMove(ctx context.Context, cc *clientConn, dir string) error {
	start := time.Now()
	playerMoveInFlight.Inc()
	playerMoves.Inc()

	defer func() {
		playerMoveDuration.Observe((time.Since(start).Seconds()))
		playerMoveInFlight.Dec()
	}()

	cc.stateMu.Lock()
	player := cc.player
	cc.stateMu.Unlock()

	if player == nil {
		return fmt.Errorf("player is nil")
	}
	world, err := h.worldCache.Get(ctx, player.WorldID)
	if err != nil {
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  err.Error(),
		})
	}
	if world == nil {
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  "world not found",
		})
	}

	cc.stateMu.Lock()
	if err := player.Move(dir, world); err != nil {
		cc.stateMu.Unlock()
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  err.Error(),
		})
	}
	cc.dirty = true
	snapshot := *player
	cc.stateMu.Unlock()

	return cc.sendJson(serverMsg{
		Type:   "playerUpdate",
		Msg:    "Player moved successfully",
		Player: &snapshot,
	})
}

func (h *Handler) HandlePlayerAttack(ctx context.Context, cc *clientConn) error {
	start := time.Now()
	playerAttackInFlight.Inc()
	playerAttacks.Inc()

	defer func() {
		playerAttackDuration.Observe((time.Since(start).Seconds()))
		playerAttackInFlight.Dec()
	}()

	cc.stateMu.Lock()
	p := cc.player
	cc.stateMu.Unlock()
	if p == nil {
		return fmt.Errorf("player not found")
	}

	if !h.Mobs.DamageNearest(p) {
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  "no mob in range to attack",
		})
	}

	cc.stateMu.Lock()
	cc.dirty = true
	cc.stateMu.Unlock()
	return nil
}

func (h *Handler) SaveDirtyPlayers(ctx context.Context) error {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			conns := h.conns.snapshot()
			for _, conn := range conns {
				conn.stateMu.Lock()
				if !conn.dirty || conn.player == nil || conn.player.ID == "" {
					conn.stateMu.Unlock()
					continue
				}
				snapshot := *conn.player
				conn.dirty = false
				conn.stateMu.Unlock()

				if _, err := h.Players.SavePlayer(ctx, &snapshot); err != nil {
					log.Printf("failed to save player %s: %v", snapshot.ID, err)
					conn.stateMu.Lock()
					conn.dirty = true
					conn.stateMu.Unlock()
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}
