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

type ClientMsg struct {
	PlayerID  string `json:"playerID"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Direction string `json:"direction"`
}

type Handler struct {
	Worlds      domain.WorldStore
	Player      domain.PlayerStore
	Mobs        domain.MobStore
	connections map[net.Conn]bool
	connMutex   sync.RWMutex
}

func NewHandler(worldStore domain.WorldStore, playerStore domain.PlayerStore, mobStore domain.MobStore) *Handler {
	return &Handler{
		Worlds:      worldStore,
		Player:      playerStore,
		Mobs:        mobStore,
		connections: make(map[net.Conn]bool),
	}
}

func (h *Handler) HandleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	fmt.Printf("New connection from: %v \n", conn.RemoteAddr().String())

	h.addConnection(conn)
	defer h.removeConn(conn)

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

func (h *Handler) HandleSendWorld(ctx context.Context, conn net.Conn, worldID string) error {
	fmt.Printf("Sending world %s\n", worldID)
	world := h.Worlds.GetWorld(worldID)

	response := serverMsg{
		Type:  "world",
		World: world,
	}

	data, err := json.Marshal(response)
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
	
	response := serverMsg{
		Type:   "playerUpdate",
		Msg:    fmt.Sprintf("Player moved to (%d, %d)", player.X, player.Y),
		Player: player,
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	fmt.Printf("Sending player data: %s\n", string(data))
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) HandleMobSpawn(ctx context.Context, worldID string, mobType string, name string) (*domain.Mob, error) {
	world := h.Worlds.GetWorld(worldID)
	if world == nil {
		return nil, fmt.Errorf("world not found")
	}

	x, y, err := h.getRandomSpawnPosition(worldID)
	if err != nil {
		return nil, err
	}

	mobID := fmt.Sprintf("mob-%d", h.Mobs.CountMobsInWorld(worldID)+1)
	mob := &domain.Mob{
		ID:      mobID,
		Name:    name,
		Type:    mobType,
		X:       x,
		Y:       y,
		Health:  100,
		Attack:  10,
		Defense: 5,
		Symbol:  'M',
	}

	mob.WorldID = worldID
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

func (h *Handler) getRandomSpawnPosition(worldID string) (int, int, error) {
	world := h.Worlds.GetWorld(worldID)
	if world == nil {
		return 0, 0, fmt.Errorf("world not found")
	}

	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		x := rand.Intn(world.Width)
		y := rand.Intn(world.Height)

		if world.Layout[y][x] != '#' && world.Layout[y][x] != '@' {
			if !h.isMobAtPosition(worldID, x, y) {
				return x, y, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("could not find valid spawn position after %d attempts", maxAttempts)
}

func (h *Handler) isMobAtPosition(worldID string, x, y int) bool {
	mobs := h.Mobs.GetMobsByWorld(worldID)
	for _, mob := range mobs {
		if mob.X == x && mob.Y == y {
			return true
		}
	}
	return false
}

func (h *Handler) addConnection(conn net.Conn) {
	h.connMutex.Lock()
	defer h.connMutex.Unlock()

	h.connections[conn] = true
}

func (h *Handler) removeConn(conn net.Conn) {
	h.connMutex.Lock()
	defer h.connMutex.Unlock()

	delete(h.connections, conn)
}

type MobUpdate struct {
	Type string        `json:"type"`
	Mobs []*domain.Mob `json:"mobs"`
}

func (h *Handler) BroadcastMobUpdate(worldID string) error {
	h.connMutex.RLock()
	defer h.connMutex.RUnlock()

	mobs := h.Mobs.GetMobsByWorld(worldID)

	response := MobUpdate{
		Type: "mobsUpdate",
		Mobs: mobs,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	var failedConns []net.Conn
	for conn := range h.connections {
		_, err := conn.Write(append(data, '\n'))
		if err != nil {
			failedConns = append(failedConns, conn)
		}
	}

	for _, conn := range failedConns {
		h.removeConn(conn)
	}
	return nil
}
