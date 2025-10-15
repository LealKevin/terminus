package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/LealKevin/terminus/internal/domain"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalMessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "terminus_messages_received_total",
		Help: "The total number of messages received from clients",
	}, []string{"type"})

	totalMessagesErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "terminus_messages_errors_total",
		Help: "The total number of errors processing messages from clients",
	}, []string{"type"})

	messageProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "terminus_message_processing_duration_seconds",
		Help: "Duration of message processing",
	}, []string{"type"})

	playersOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "terminus_players_online",
		Help: "Total of players online",
	})

	playerMoveDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "terminus_player_move_duration_seconds",
		Help: "Latency of HandlePlayerMove",
	})

	playerMoveInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "terminus_player_move_in_flight",
		Help: "In-flight player move operations",
	})

	playerMoves = promauto.NewCounter(prometheus.CounterOpts{
		Name: "terminus_player_moves_total",
		Help: "Total player moves",
	})

	playerAttacks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "terminus_player_attacks_total",
		Help: "Total player attacks",
	})

	playerAttackDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "terminus_player_attack_duration_seconds",
		Help: "Latency of HandlePlayerAttack",
	})

	playerAttackInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "terminus_player_attack_in_flight",
		Help: "In-flight player attack operations",
	})

	mobMoves = promauto.NewCounter(prometheus.CounterOpts{
		Name: "terminus_mob_moves_total",
		Help: "Total mob moves",
	})

	mobMoveDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "terminus_mob_move_duration_seconds",
		Help: "Latency of HandleMobMove",
	})

	mobMoveInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "terminus_mob_move_in_flight",
		Help: "In-flight mob move operations",
	})
)

type clientConn struct {
	conn   net.Conn
	w      *bufio.Writer
	enc    *json.Encoder
	mu     sync.Mutex
	player *domain.Player
	dirty  bool
}

type ClientMsg struct {
	PlayerID  string `json:"playerID"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Direction string `json:"direction"`
}

type StoreProvider interface {
	Worlds() domain.WorldStore
	Players() domain.PlayerStore
	Mobs() domain.MobStore
}

type Handler struct {
	Worlds domain.WorldStore
	Player domain.PlayerStore
	Mobs   domain.MobStore

	connections map[*clientConn]bool
	connMutex   sync.RWMutex

	cachedWorld *domain.World
	worldMutex  sync.RWMutex
}

func NewHandler(stores StoreProvider) *Handler {
	return &Handler{
		Worlds:      stores.Worlds(),
		Player:      stores.Players(),
		Mobs:        stores.Mobs(),
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

func (h *Handler) getCachedWorld(ctx context.Context, worldID string) (*domain.World, error) {
	h.worldMutex.RLock()
	if h.cachedWorld != nil && h.cachedWorld.ID == worldID {
		world := h.cachedWorld
		h.worldMutex.RUnlock()
		return world, nil
	}
	h.worldMutex.RUnlock()

	h.worldMutex.Lock()
	defer h.worldMutex.Unlock()

	if h.cachedWorld != nil && h.cachedWorld.ID == worldID {
		return h.cachedWorld, nil
	}

	world, err := h.Worlds.GetWorld(ctx, worldID)
	if err != nil {
		return nil, err
	}
	h.cachedWorld = world
	return world, nil
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
	defer playersOnline.Dec()
	fmt.Printf("New connection from: %v \n", conn.RemoteAddr().String())

	h.addConnection(cc)
	defer h.removeConn(cc)

	worldID := os.Getenv("WORLD_ID")
	if worldID == "" {
		worldID = "123e4567-e89b-12d3-a456-426614174000" // fallback
	}
	
	world, err := h.getCachedWorld(ctx, worldID)
	if err != nil {
		h.sendError(conn, fmt.Errorf("unable to load world: %v", err))
		return
	}
	if world == nil {
		h.sendError(conn, fmt.Errorf("world not found"))
		return
	}

	occupied, err := h.getOccupiedPositions(ctx, world.ID)
	if err != nil {
		h.sendError(conn, fmt.Errorf("unable to inspect world occupancy: %v", err))
		return
	}

	y, x, err := domain.FindRandomSpawnPosition(world, occupied)
	if err != nil {
		h.sendError(conn, fmt.Errorf("unable to find spawn position: %v", err))
		return
	}

	playerID := uuid.New().String()
	player, err := h.Player.GetPlayer(ctx, playerID)
	if err != nil {
		h.sendError(conn, fmt.Errorf("unable to load player: %v", err))
	}
	if player == nil {
		player = domain.NewPlayer(playerID, x, y)
	}

	saved, err := h.Player.SavePlayer(ctx, player)
	if err != nil {
		h.sendError(conn, fmt.Errorf("unable to persist player: %v", err))
		return
	}
	cc.player = saved

	err = cc.sendJson(serverMsg{
		Type:   "playerUpdate",
		Msg:    "Player created successfully",
		Player: saved,
	})
	if err != nil {
		log.Printf("error sending initial player data: %v", err)
	}

	reader := bufio.NewReader(cc.conn)

	playersOnline.Inc()

	for {
		select {
		case <-ctx.Done():
			h.sendError(conn, fmt.Errorf("server is shutting down"))
			return

		default:
			cc.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("error reading from connection: %v", err)
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
	start := time.Now()
	totalMessagesReceived.WithLabelValues(msg.Type).Inc()
	defer func() {
		messageProcessingDuration.WithLabelValues(msg.Type).Observe(time.Since(start).Seconds())
	}()

	if msg.Type == "getPlayer" {
		if cc.player != nil && cc.player.ID != "" {
			cc.sendJson(serverMsg{
				Type:   "playerUpdate",
				Msg:    "Player retrieved successfully",
				Player: cc.player,
			})
		} else {
			cc.sendJson(serverMsg{Type: "error", Msg: "connection not properly initialized - player created during connection setup"})
		}
		return
	}

	if cc.player == nil {
		cc.sendJson(serverMsg{Type: "error", Msg: "player not initialized"})
		return
	}

	player := cc.player

	switch msg.Type {
	case "move":
		err := h.HandlePlayerMove(ctx, cc, player, msg.Direction)
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
		err := h.HandlerPlayerAttack(ctx, player, cc)
		if err != nil {
			log.Printf("error handling player attack: %v", err)
			totalMessagesErrors.WithLabelValues(msg.Type).Inc()
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

func (h *Handler) HandleSendWorld(ctx context.Context, cc *clientConn, worldID string) error {
	fmt.Printf("Sending world %s\n", worldID)
	world, err := h.getCachedWorld(ctx, worldID)
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

func (h *Handler) HandlePlayerMove(ctx context.Context, cc *clientConn, player *domain.Player, dir string) error {
	start := time.Now()
	playerMoveInFlight.Inc()
	playerMoves.Inc()

	defer func() {
		playerMoveDuration.Observe((time.Since(start).Seconds()))
		playerMoveInFlight.Dec()
	}()

	if player == nil {
		return fmt.Errorf("player is nil")
	}
	world, err := h.getCachedWorld(ctx, player.WorldID)
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

	err = player.Move(dir, world)
	if err != nil {
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  err.Error(),
		})
	}

	cc.dirty = true
	return cc.sendJson(serverMsg{
		Type:   "playerUpdate",
		Msg:    "Player moved successfully",
		Player: player,
	})
}

func (h *Handler) HandlerPlayerAttack(ctx context.Context, p *domain.Player, cc *clientConn) error {
	start := time.Now()
	playerAttackInFlight.Inc()
	playerAttacks.Inc()

	defer func() {
		playerAttackDuration.Observe((time.Since(start).Seconds()))
		playerAttackInFlight.Dec()
	}()
	if p == nil {
		return fmt.Errorf("player not found")
	}
	mob, err := h.nearestMob(ctx, p, p.WorldID, p.Range)
	if err != nil {
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  "no mob in range to attack",
		})
	}

	p.AttackMob(mob)
	if mob.IsAlive() {
		if _, err := h.Mobs.SaveMob(ctx, mob); err != nil {
			return cc.sendJson(serverMsg{
				Type: "error",
				Msg:  err.Error(),
			})
		}
	} else {
		if err := h.Mobs.DeleteMob(ctx, mob.ID); err != nil {
			return cc.sendJson(serverMsg{
				Type: "error",
				Msg:  err.Error(),
			})
		}
	}

	cc.dirty = true
	return nil
}

func (h *Handler) nearestMob(ctx context.Context, p *domain.Player, worldID string, attackRange int) (*domain.Mob, error) {
	mobs, err := h.Mobs.GetMobsByWorld(ctx, worldID)
	if err != nil {
		return nil, err
	}

	var nearest *domain.Mob
	minDist := attackRange + 1

	for _, mob := range mobs {
		// Manhattan distance
		dist := abs(p.X-mob.X) + abs(p.Y-mob.Y)
		if dist < minDist {
			minDist = dist
			nearest = mob
		}
	}
	if nearest == nil {
		return nil, fmt.Errorf("no mob found")
	}
	return nearest, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (h *Handler) HandleMobSpawn(ctx context.Context, worldID string, mobType string, name string) (*domain.Mob, error) {
	world, err := h.getCachedWorld(ctx, worldID)
	if err != nil {
		return nil, err
	}
	if world == nil {
		return nil, fmt.Errorf("world not found")
	}

	mobID := uuid.New().String()
	occupiedPositions, err := h.getOccupiedPositions(ctx, worldID)
	if err != nil {
		return nil, err
	}

	mob, err := world.SpawnMob(mobType, name, mobID, occupiedPositions)
	if err != nil {
		return nil, err
	}

	created, err := h.Mobs.CreateMob(ctx, mob)
	if err != nil {
		return nil, err
	}
	log.Printf("Spawned mob %s of type %s at (%d, %d) in world %s", created.ID, created.Type, created.X, created.Y, world.ID)

	return created, nil
}

func (h *Handler) HandleMobMove(ctx context.Context, mobID string) error {
	start := time.Now()
	mobMoveInFlight.Inc()
	mobMoves.Inc()

	defer func() {
		mobMoveDuration.Observe((time.Since(start).Seconds()))
		mobMoveInFlight.Dec()
	}()
	mob, err := h.Mobs.GetMob(ctx, mobID)
	if err != nil {
		return err
	}
	if mob == nil {
		return fmt.Errorf("mob not found")
	}
	world, err := h.getCachedWorld(ctx, mob.WorldID)
	if err != nil {
		return err
	}
	if world == nil {
		return fmt.Errorf("world not found")
	}

	directions := []string{"N", "S", "E", "W", "NE", "NW", "SE", "SW"}
	direction := directions[rand.Intn(len(directions))]

	err = mob.Move(direction, world)
	if err != nil {
		return err
	}

	if _, err := h.Mobs.SaveMob(ctx, mob); err != nil {
		return err
	}
	return nil
}

func (h *Handler) HandleMobDespawn(ctx context.Context, mobID string) error {
	mob, err := h.Mobs.GetMob(ctx, mobID)
	if err != nil {
		return err
	}
	if mob == nil {
		return fmt.Errorf("mob not found")
	}
	if err := h.Mobs.DeleteMob(ctx, mobID); err != nil {
		return err
	}
	log.Printf("Mob %s despawned from world %s", mob.ID, mob.WorldID)
	return nil
}

func (h *Handler) getOccupiedPositions(ctx context.Context, worldID string) (map[string]bool, error) {
	occupied := make(map[string]bool)

	mobs, err := h.Mobs.GetMobsByWorld(ctx, worldID)
	if err != nil {
		return nil, err
	}

	for _, mob := range mobs {
		key := fmt.Sprintf("%d,%d", mob.X, mob.Y)
		occupied[key] = true
	}

	return occupied, nil
}

type MobUpdate struct {
	Type string        `json:"type"`
	Mobs []*domain.Mob `json:"mobs"`
}

func (h *Handler) BroadcastMobsUpdate(ctx context.Context, worldID string) error {
	h.connMutex.RLock()
	defer h.connMutex.RUnlock()

	mobs, err := h.Mobs.GetMobsByWorld(ctx, worldID)
	if err != nil {
		return err
	}

	response := MobUpdate{
		Type: "mobsUpdate",
		Mobs: mobs,
	}

	var failedConns []*clientConn

	for conn := range h.connections {
		if err := conn.sendJson(response); err != nil {
			failedConns = append(failedConns, conn)
		}
	}

	for _, conn := range failedConns {
		h.removeConn(conn)
	}
	return nil
}

func (h *Handler) SaveDirtyPlayers(ctx context.Context) error {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			h.connMutex.RLock()
			for conn := range h.connections {
				if conn.dirty && conn.player != nil && conn.player.ID != "" {
					conn.dirty = false
					_, err := h.Player.SavePlayer(ctx, conn.player)
					if err != nil {
						log.Printf("failed to save player %s: %v", conn.player.ID, err)
						continue
					}
				}
			}
			h.connMutex.RUnlock()
		case <-ctx.Done():
			return nil
		}
	}
}
