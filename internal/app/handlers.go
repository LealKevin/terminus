package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/LealKevin/terminus/internal/domain"
	"github.com/google/uuid"
)

type StoreProvider interface {
	Worlds() domain.WorldStore
	Players() domain.PlayerStore
	Mobs() domain.MobStore
}

type Handler struct {
	Worlds  domain.WorldStore
	Players domain.PlayerStore
	Mobs    *MobManager

	conns      *connRegistry
	worldCache *worldCache
}

func NewHandler(stores StoreProvider) *Handler {
	cache := newWorldCache(stores.Worlds())
	return &Handler{
		Worlds:     stores.Worlds(),
		Players:    stores.Players(),
		Mobs:       NewMobManager(stores.Mobs(), cache),
		conns:      newConnRegistry(),
		worldCache: cache,
	}
}

func (h *Handler) HandleConnection(ctx context.Context, conn net.Conn) {
	cc := wrapConnection(conn)

	defer conn.Close()
	fmt.Printf("New connection from: %v \n", conn.RemoteAddr().String())

	world, err := h.worldCache.Get(ctx, domain.ResolveWorldID())
	if err != nil {
		h.sendError(conn, fmt.Errorf("unable to load world: %v", err))
		return
	}
	if world == nil {
		h.sendError(conn, fmt.Errorf("world not found"))
		return
	}

	y, x, err := domain.FindRandomSpawnPosition(world, h.Mobs.OccupiedPositions(world.ID))
	if err != nil {
		h.sendError(conn, fmt.Errorf("unable to find spawn position: %v", err))
		return
	}

	player := domain.NewPlayer(uuid.New().String(), x, y)
	saved, err := h.Players.SavePlayer(ctx, player)
	if err != nil {
		h.sendError(conn, fmt.Errorf("unable to persist player: %v", err))
		return
	}

	cc.stateMu.Lock()
	cc.player = saved
	cc.stateMu.Unlock()

	if err := cc.sendJson(serverMsg{
		Type:   "playerUpdate",
		Msg:    "Player created successfully",
		Player: saved,
	}); err != nil {
		log.Printf("error sending initial player data: %v", err)
		return
	}

	h.conns.add(cc)
	defer h.conns.remove(cc)
	playersOnline.Inc()
	defer playersOnline.Dec()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	reader := bufio.NewReader(cc.conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("error reading from connection: %v", err)
			}
			return
		}

		var msg ClientMsg
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Printf("error unmarshaling json: %v", err)
			return
		}

		h.HandleMessage(ctx, msg, cc)
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg ClientMsg, cc *clientConn) {
	start := time.Now()
	totalMessagesReceived.WithLabelValues(msg.Type).Inc()
	defer func() {
		messageProcessingDuration.WithLabelValues(msg.Type).Observe(time.Since(start).Seconds())
	}()

	cc.stateMu.Lock()
	player := cc.player
	cc.stateMu.Unlock()

	if msg.Type == "getPlayer" {
		if player != nil && player.ID != "" {
			cc.sendJson(serverMsg{
				Type:   "playerUpdate",
				Msg:    "Player retrieved successfully",
				Player: player,
			})
		} else {
			cc.sendJson(serverMsg{Type: "error", Msg: "connection not properly initialized - player created during connection setup"})
		}
		return
	}

	if player == nil {
		cc.sendJson(serverMsg{Type: "error", Msg: "player not initialized"})
		return
	}

	switch msg.Type {
	case "move":
		err := h.HandlePlayerMove(ctx, cc, msg.Direction)
		if err != nil {
			log.Printf("error handling player move: %v", err)
			totalMessagesErrors.WithLabelValues(msg.Type).Inc()
		}

	case "getWorld":
		err := h.HandleSendWorld(ctx, cc, msg.Message)
		if err != nil {
			log.Printf("error sending world: %v", err)
			totalMessagesErrors.WithLabelValues(msg.Type).Inc()
		}

	case "attack":
		err := h.HandlePlayerAttack(ctx, cc)
		if err != nil {
			log.Printf("error handling player attack: %v", err)
			totalMessagesErrors.WithLabelValues(msg.Type).Inc()
		}
	}
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

func (h *Handler) HandleSendWorld(ctx context.Context, cc *clientConn, worldID string) error {
	fmt.Printf("Sending world %s\n", worldID)
	world, err := h.worldCache.Get(ctx, worldID)
	if err != nil {
		return err
	}
	if world == nil {
		return fmt.Errorf("world %s not found", worldID)
	}

	response := serverMsg{
		Type:  "world",
		World: world,
	}

	return cc.sendJson(response)
}

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

