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
	"time"

	"github.com/LealKevin/terminus/internal/domain"
	"github.com/google/uuid"
)

type clientConn struct {
	conn net.Conn
	w    *bufio.Writer
	enc  *json.Encoder

	mu sync.Mutex

	stateMu sync.Mutex
	player  *domain.Player
	dirty   bool
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
	Worlds  domain.WorldStore
	Players domain.PlayerStore
	Mobs    domain.MobStore

	connections map[*clientConn]struct{}
	connMutex   sync.RWMutex

	cachedWorld *domain.World
	worldMutex  sync.RWMutex

	mobsMu     sync.RWMutex
	mobs       map[string]*domain.Mob
	mobDirty   map[string]struct{}
	mobDeleted map[string]struct{}
}

func NewHandler(stores StoreProvider) *Handler {
	return &Handler{
		Worlds:      stores.Worlds(),
		Players:     stores.Players(),
		Mobs:        stores.Mobs(),
		connections: make(map[*clientConn]struct{}),
		mobs:        make(map[string]*domain.Mob),
		mobDirty:    make(map[string]struct{}),
		mobDeleted:  make(map[string]struct{}),
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

	h.connections[conn] = struct{}{}
}

func (h *Handler) snapshotConnections() []*clientConn {
	h.connMutex.RLock()
	defer h.connMutex.RUnlock()
	conns := make([]*clientConn, 0, len(h.connections))
	for c := range h.connections {
		conns = append(conns, c)
	}
	return conns
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
	fmt.Printf("New connection from: %v \n", conn.RemoteAddr().String())

	world, err := h.getCachedWorld(ctx, domain.ResolveWorldID())
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

	h.addConnection(cc)
	defer h.removeConn(cc)
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

	h.mobsMu.Lock()
	mob := h.nearestMobLocked(p, p.WorldID, p.Range)
	if mob == nil {
		h.mobsMu.Unlock()
		return cc.sendJson(serverMsg{
			Type: "error",
			Msg:  "no mob in range to attack",
		})
	}

	p.AttackMob(mob)
	if mob.IsAlive() {
		h.mobDirty[mob.ID] = struct{}{}
	} else {
		delete(h.mobs, mob.ID)
		delete(h.mobDirty, mob.ID)
		h.mobDeleted[mob.ID] = struct{}{}
	}
	h.mobsMu.Unlock()

	cc.stateMu.Lock()
	cc.dirty = true
	cc.stateMu.Unlock()
	return nil
}

func (h *Handler) mobsInWorldLocked(worldID string) []*domain.Mob {
	out := make([]*domain.Mob, 0, len(h.mobs))
	for _, m := range h.mobs {
		if m.WorldID == worldID {
			out = append(out, m)
		}
	}
	return out
}

func (h *Handler) nearestMobLocked(p *domain.Player, worldID string, attackRange int) *domain.Mob {
	var nearest *domain.Mob
	minDist := attackRange + 1
	for _, mob := range h.mobsInWorldLocked(worldID) {
		dist := abs(p.X-mob.X) + abs(p.Y-mob.Y)
		if dist < minDist {
			minDist = dist
			nearest = mob
		}
	}
	return nearest
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (h *Handler) LoadMobs(ctx context.Context, worldID string) error {
	mobs, err := h.Mobs.GetMobsByWorld(ctx, worldID)
	if err != nil {
		return err
	}
	h.mobsMu.Lock()
	defer h.mobsMu.Unlock()
	for _, m := range mobs {
		h.mobs[m.ID] = m
	}
	return nil
}

func (h *Handler) MobIDsInWorld(worldID string) []string {
	h.mobsMu.RLock()
	defer h.mobsMu.RUnlock()
	mobs := h.mobsInWorldLocked(worldID)
	ids := make([]string, 0, len(mobs))
	for _, m := range mobs {
		ids = append(ids, m.ID)
	}
	return ids
}

func (h *Handler) HandleMobSpawn(ctx context.Context, worldID string, mobType string, name string) (*domain.Mob, error) {
	world, err := h.getCachedWorld(ctx, worldID)
	if err != nil {
		return nil, err
	}
	if world == nil {
		return nil, fmt.Errorf("world not found")
	}

	h.mobsMu.Lock()
	occupied := h.occupiedPositionsLocked(worldID)
	mobID := uuid.New().String()
	mob, err := world.SpawnMob(mobType, name, mobID, occupied)
	if err != nil {
		h.mobsMu.Unlock()
		return nil, err
	}
	h.mobs[mob.ID] = mob
	h.mobsMu.Unlock()

	created, err := h.Mobs.CreateMob(ctx, mob)
	if err != nil {
		h.mobsMu.Lock()
		delete(h.mobs, mob.ID)
		h.mobsMu.Unlock()
		return nil, err
	}

	h.mobsMu.Lock()
	if existing, ok := h.mobs[mob.ID]; ok {
		*existing = *created
	}
	h.mobsMu.Unlock()

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

	world, err := h.getCachedWorld(ctx, domain.ResolveWorldID())
	if err != nil {
		return err
	}
	if world == nil {
		return fmt.Errorf("world not found")
	}

	direction := mobDirections[rand.Intn(len(mobDirections))]

	h.mobsMu.Lock()
	defer h.mobsMu.Unlock()
	mob, ok := h.mobs[mobID]
	if !ok {
		return fmt.Errorf("mob not found")
	}
	if err := mob.Move(direction, world); err != nil {
		return err
	}
	h.mobDirty[mobID] = struct{}{}
	return nil
}

var mobDirections = []string{"N", "S", "E", "W", "NE", "NW", "SE", "SW"}

func (h *Handler) HandleMobDespawn(ctx context.Context, mobID string) error {
	h.mobsMu.Lock()
	mob, ok := h.mobs[mobID]
	if !ok {
		h.mobsMu.Unlock()
		return fmt.Errorf("mob not found")
	}
	delete(h.mobs, mobID)
	delete(h.mobDirty, mobID)
	h.mobDeleted[mobID] = struct{}{}
	worldID := mob.WorldID
	h.mobsMu.Unlock()

	log.Printf("Mob %s despawned from world %s", mobID, worldID)
	return nil
}

func (h *Handler) occupiedPositionsLocked(worldID string) map[string]bool {
	mobs := h.mobsInWorldLocked(worldID)
	occupied := make(map[string]bool, len(mobs))
	for _, mob := range mobs {
		occupied[fmt.Sprintf("%d,%d", mob.X, mob.Y)] = true
	}
	return occupied
}

func (h *Handler) getOccupiedPositions(ctx context.Context, worldID string) (map[string]bool, error) {
	h.mobsMu.RLock()
	defer h.mobsMu.RUnlock()
	return h.occupiedPositionsLocked(worldID), nil
}

type MobUpdate struct {
	Type string        `json:"type"`
	Mobs []*domain.Mob `json:"mobs"`
}

func (h *Handler) BroadcastMobsUpdate(ctx context.Context, worldID string) error {
	h.mobsMu.RLock()
	live := h.mobsInWorldLocked(worldID)
	mobs := make([]*domain.Mob, 0, len(live))
	for _, m := range live {
		snap := *m
		mobs = append(mobs, &snap)
	}
	h.mobsMu.RUnlock()

	response := MobUpdate{
		Type: "mobsUpdate",
		Mobs: mobs,
	}

	conns := h.snapshotConnections()
	var failed []*clientConn
	for _, conn := range conns {
		if err := conn.sendJson(response); err != nil {
			failed = append(failed, conn)
		}
	}
	for _, conn := range failed {
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
			conns := h.snapshotConnections()
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

func (h *Handler) FlushMobs(ctx context.Context) error {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			h.flushMobsOnce(ctx)
		case <-ctx.Done():
			h.flushMobsOnce(context.Background())
			return nil
		}
	}
}

func (h *Handler) flushMobsOnce(ctx context.Context) {
	h.mobsMu.Lock()
	if len(h.mobDirty) == 0 && len(h.mobDeleted) == 0 {
		h.mobsMu.Unlock()
		return
	}
	dirtyIDs := make([]string, 0, len(h.mobDirty))
	for id := range h.mobDirty {
		dirtyIDs = append(dirtyIDs, id)
	}
	deletedIDs := make([]string, 0, len(h.mobDeleted))
	for id := range h.mobDeleted {
		deletedIDs = append(deletedIDs, id)
	}

	snaps := make([]domain.Mob, 0, len(dirtyIDs))
	for _, id := range dirtyIDs {
		m, ok := h.mobs[id]
		if !ok {
			delete(h.mobDirty, id)
			continue
		}
		snaps = append(snaps, *m)
		delete(h.mobDirty, id)
	}
	for _, id := range deletedIDs {
		delete(h.mobDeleted, id)
	}
	h.mobsMu.Unlock()

	for i := range snaps {
		s := snaps[i]
		if _, err := h.Mobs.SaveMob(ctx, &s); err != nil {
			log.Printf("failed to save mob %s: %v", s.ID, err)
			h.mobsMu.Lock()
			if _, ok := h.mobs[s.ID]; ok {
				h.mobDirty[s.ID] = struct{}{}
			}
			h.mobsMu.Unlock()
		}
	}
	for _, id := range deletedIDs {
		if err := h.Mobs.DeleteMob(ctx, id); err != nil {
			log.Printf("failed to delete mob %s: %v", id, err)
			h.mobsMu.Lock()
			h.mobDeleted[id] = struct{}{}
			h.mobsMu.Unlock()
		}
	}
}
