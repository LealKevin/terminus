package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"

	"github.com/LealKevin/terminus/internal/domain"
)

type clientConn struct {
	conn net.Conn
	w    *bufio.Writer
	enc  *json.Encoder
	mu   sync.Mutex
}

type ClientMsg struct {
	PlayerID  string `json:"playerID"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Direction string `json:"direction"`
}

type Handler struct {
	Worlds domain.WorldStore
	Player domain.PlayerStore
	Mobs   domain.MobStore

	connections map[*clientConn]bool
	connMutex   sync.RWMutex
}

func NewHandler(worldStore domain.WorldStore, playerStore domain.PlayerStore, mobStore domain.MobStore) *Handler {
	return &Handler{
		Worlds:      worldStore,
		Player:      playerStore,
		Mobs:        mobStore,
		connections: make(map[*clientConn]bool),
	}
}

func (h *Handler) wrapConnection(c net.Conn) *clientConn {
	w := bufio.NewWriter(c)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &clientConn{conn: c, w: w, enc: enc}
}

func (h *Handler) addConnection(conn *clientConn) {
	h.connMutex.Lock()
	defer h.connMutex.Unlock()

	h.connections[conn] = true
}

func (h *Handler) removeConn(conn *clientConn) {
	h.connMutex.Lock()
	defer h.connMutex.Unlock()

	delete(h.connections, conn)
}

func (cc *clientConn) sendJson(v any) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if err := cc.enc.Encode(v); err != nil {
		return err
	}
	return cc.w.Flush()
}

func (h *Handler) HandleConnection(ctx context.Context, conn net.Conn) {
	cc := h.wrapConnection(conn)

	defer conn.Close()

	fmt.Printf("New connection from: %v \n", conn.RemoteAddr().String())

	h.addConnection(cc)
	defer h.removeConn(cc)

	reader := bufio.NewReader(cc.conn)

	for {
		select {
		case <-ctx.Done():
			h.sendError(conn, fmt.Errorf("server is shutting down"))
			return

		default:
			line, err := reader.ReadBytes('\n')
			if err != nil {
				log.Printf("error reading from connection: %v, addr: %v", err, cc.conn.RemoteAddr().String())
				return
			}

			var msg ClientMsg
			err = json.Unmarshal(line, &msg)
			if err != nil {
				log.Printf("error unmarshaling json: %v", err)
				return
			}

			h.HandleMessage(ctx, msg, cc)
		}
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg ClientMsg, cc *clientConn) {
	switch msg.Type {
	case "getPlayer":
		player := h.Player.GetPlayer(msg.PlayerID)

		if player == nil {
			err := fmt.Errorf("player not found")
			log.Printf("error retrieving player: %v", err)
			cc.sendJson(serverMsg{
				Type: "error",
				Msg:  err.Error(),
			})
			return
		}
		err := cc.sendJson(serverMsg{
			Type:   "playerUpdate",
			Msg:    "Player retrieved successfully",
			Player: player,
		})
		if err != nil {
			log.Printf("error sending player data: %v", err)
		} else {
			log.Printf("Player %s sent successfully", player.ID)
		}

	case "move":
		err := h.HandlePlayerMove(ctx, cc, msg.PlayerID, msg.Direction)
		if err != nil {
			log.Printf("error handling player move: %v", err)
		}

	case "getWorld":
		err := h.HandleSendWorld(ctx, cc, msg.Message)
		if err != nil {
			log.Printf("error sending world: %v", err)
		}
	}
}

type serverMsg struct {
	Type   string         `json:"type"`
	Msg    string         `json:"msg"`
	World  *domain.World  `json:"world,omitempty"`
	Player *domain.Player `json:"player,omitempty"`
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

func (h *Handler) HandleSendWorld(ctx context.Context, cc *clientConn, worldID string) error {
	fmt.Printf("Sending world %s\n", worldID)
	world := h.Worlds.GetWorld(worldID)

	response := serverMsg{
		Type:  "world",
		World: world,
	}

	return cc.sendJson(response)
}

func (h *Handler) HandlePlayerMove(ctx context.Context, cc *clientConn, playerID string, dir string) error {
	player := h.Player.GetPlayer(playerID)
	if player == nil {
		err := fmt.Errorf("player not found")
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  err.Error(),
		})
	}
	world := h.Worlds.GetWorld(player.WorldID)
	if world == nil {
		err := fmt.Errorf("world not found")
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  err.Error(),
		})
	}
	err := player.Move(dir, world)
	if err != nil {
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  err.Error(),
		})
	}
	h.Player.SavePlayer(player)

	return cc.sendJson(serverMsg{
		Type:   "playerUpdate",
		Msg:    "Player moved successfully",
		Player: player,
	})
}

func (h *Handler) HandleMobSpawn(ctx context.Context, worldID string, mobType string, name string) (*domain.Mob, error) {
	world := h.Worlds.GetWorld(worldID)
	if world == nil {
		return nil, fmt.Errorf("world not found")
	}

	mobID := fmt.Sprintf("mob-%d-%d", h.Mobs.CountMobsInWorld(worldID)+1, rand.Intn(10000))
	occupiedPositions := h.getOccupiedPositions(worldID)

	mob, err := world.SpawnMob(mobType, name, mobID, occupiedPositions)
	if err != nil {
		return nil, err
	}

	h.Mobs.CreateMob(mob)
	log.Printf("Spawned mob %s of type %s at (%d, %d) in world %s", mob.ID, mob.Type, mob.X, mob.Y, world.ID)

	return mob, nil
}

func (h *Handler) HandleMobMove(ctx context.Context, mobID string) error {
	mob := h.Mobs.GetMob(mobID)
	if mob == nil {
		return fmt.Errorf("mob not found")
	}
	world := h.Worlds.GetWorld(mob.WorldID)
	if world == nil {
		return fmt.Errorf("world not found")
	}

	directions := []string{"N", "S", "E", "W", "NE", "NW", "SE", "SW"}
	direction := directions[rand.Intn(len(directions))]

	err := mob.Move(direction, world)
	if err != nil {
		return err
	}
	h.Mobs.SaveMob(mob)
	log.Printf("Mob %s moved to (%d, %d) in world %s", mob.ID, mob.X, mob.Y, world.ID)
	return nil
}

func (h *Handler) HandleMobDespawn(ctx context.Context, mobID string) error {
	mob := h.Mobs.GetMob(mobID)
	if mob == nil {
		return fmt.Errorf("mob not found")
	}
	h.Mobs.DeleteMob(mobID)
	log.Printf("Mob %s despawned from world %s", mob.ID, mob.WorldID)
	return nil
}

func (h *Handler) getOccupiedPositions(worldID string) map[string]bool {
	occupied := make(map[string]bool)

	mobs := h.Mobs.GetMobsByWorld(worldID)
	for _, mob := range mobs {
		key := fmt.Sprintf("%d,%d", mob.X, mob.Y)
		occupied[key] = true
	}

	return occupied
}

type MobUpdate struct {
	Type string        `json:"type"`
	Mobs []*domain.Mob `json:"mobs"`
}

func (h *Handler) BroadcastMobsUpdate(worldID string) error {
	h.connMutex.RLock()
	defer h.connMutex.RUnlock()

	mobs := h.Mobs.GetMobsByWorld(worldID)
	fmt.Printf("Broadcasting mobs update: %+v\n", mobs)

	response := MobUpdate{
		Type: "mobsUpdate",
		Mobs: mobs,
	}

	var failedConns []*clientConn

	for conn := range h.connections {
		err := conn.sendJson(response)
		if err != nil {
			failedConns = append(failedConns, conn)
		}
	}

	for _, conn := range failedConns {
		h.removeConn(conn)
	}
	return nil
}
