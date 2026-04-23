package app

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/LealKevin/terminus/internal/domain"
	"github.com/google/uuid"
)

var mobDirections = []string{"N", "S", "E", "W", "NE", "NW", "SE", "SW"}

type MobManager struct {
	store  domain.MobStore
	worlds *worldCache

	mu      sync.RWMutex
	mobs    map[string]*domain.Mob
	dirty   map[string]struct{}
	deleted map[string]struct{}
}

func NewMobManager(store domain.MobStore, worlds *worldCache) *MobManager {
	return &MobManager{
		store:   store,
		worlds:  worlds,
		mobs:    make(map[string]*domain.Mob),
		dirty:   make(map[string]struct{}),
		deleted: make(map[string]struct{}),
	}
}

func (m *MobManager) Load(ctx context.Context, worldID string) error {
	mobs, err := m.store.GetMobsByWorld(ctx, worldID)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mob := range mobs {
		m.mobs[mob.ID] = mob
	}
	return nil
}

func (m *MobManager) inWorldLocked(worldID string) []*domain.Mob {
	out := make([]*domain.Mob, 0, len(m.mobs))
	for _, mob := range m.mobs {
		if mob.WorldID == worldID {
			out = append(out, mob)
		}
	}
	return out
}

func (m *MobManager) IDsInWorld(worldID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mobs := m.inWorldLocked(worldID)
	ids := make([]string, 0, len(mobs))
	for _, mob := range mobs {
		ids = append(ids, mob.ID)
	}
	return ids
}

func (m *MobManager) Snapshot(worldID string) []*domain.Mob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	live := m.inWorldLocked(worldID)
	out := make([]*domain.Mob, 0, len(live))
	for _, mob := range live {
		snap := *mob
		out = append(out, &snap)
	}
	return out
}

func (m *MobManager) OccupiedPositions(worldID string) map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mobs := m.inWorldLocked(worldID)
	occupied := make(map[string]bool, len(mobs))
	for _, mob := range mobs {
		occupied[fmt.Sprintf("%d,%d", mob.X, mob.Y)] = true
	}
	return occupied
}

func (m *MobManager) Spawn(ctx context.Context, worldID, mobType, name string) (*domain.Mob, error) {
	world, err := m.worlds.Get(ctx, worldID)
	if err != nil {
		return nil, err
	}
	if world == nil {
		return nil, fmt.Errorf("world not found")
	}

	m.mu.Lock()
	occupied := make(map[string]bool, len(m.mobs))
	for _, mob := range m.mobs {
		if mob.WorldID != worldID {
			continue
		}
		occupied[fmt.Sprintf("%d,%d", mob.X, mob.Y)] = true
	}
	mobID := uuid.New().String()
	mob, err := world.SpawnMob(mobType, name, mobID, occupied)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	m.mobs[mob.ID] = mob
	m.mu.Unlock()

	created, err := m.store.CreateMob(ctx, mob)
	if err != nil {
		m.mu.Lock()
		delete(m.mobs, mob.ID)
		m.mu.Unlock()
		return nil, err
	}

	m.mu.Lock()
	if existing, ok := m.mobs[mob.ID]; ok {
		*existing = *created
	}
	m.mu.Unlock()

	log.Printf("Spawned mob %s of type %s at (%d, %d) in world %s", created.ID, created.Type, created.X, created.Y, world.ID)
	return created, nil
}

func (m *MobManager) Move(ctx context.Context, mobID string) error {
	start := time.Now()
	mobMoveInFlight.Inc()
	mobMoves.Inc()

	defer func() {
		mobMoveDuration.Observe(time.Since(start).Seconds())
		mobMoveInFlight.Dec()
	}()

	world, err := m.worlds.Get(ctx, domain.ResolveWorldID())
	if err != nil {
		return err
	}
	if world == nil {
		return fmt.Errorf("world not found")
	}

	direction := mobDirections[rand.Intn(len(mobDirections))]

	m.mu.Lock()
	defer m.mu.Unlock()
	mob, ok := m.mobs[mobID]
	if !ok {
		return fmt.Errorf("mob not found")
	}
	if err := mob.Move(direction, world); err != nil {
		return err
	}
	m.dirty[mobID] = struct{}{}
	return nil
}

func (m *MobManager) Despawn(mobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	mob, ok := m.mobs[mobID]
	if !ok {
		return fmt.Errorf("mob not found")
	}
	delete(m.mobs, mobID)
	delete(m.dirty, mobID)
	m.deleted[mobID] = struct{}{}
	log.Printf("Mob %s despawned from world %s", mobID, mob.WorldID)
	return nil
}

func (m *MobManager) DamageNearest(p *domain.Player) *domain.Mob {
	m.mu.Lock()
	defer m.mu.Unlock()

	var nearest *domain.Mob
	minDist := p.Range + 1
	for _, mob := range m.mobs {
		if mob.WorldID != p.WorldID {
			continue
		}
		dist := abs(p.X-mob.X) + abs(p.Y-mob.Y)
		if dist < minDist {
			minDist = dist
			nearest = mob
		}
	}
	if nearest == nil {
		return nil
	}

	p.AttackMob(nearest)
	if nearest.IsAlive() {
		m.dirty[nearest.ID] = struct{}{}
	} else {
		delete(m.mobs, nearest.ID)
		delete(m.dirty, nearest.ID)
		m.deleted[nearest.ID] = struct{}{}
	}
	return nearest
}

func (m *MobManager) Flush(ctx context.Context) error {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			m.flushOnce(ctx)
		case <-ctx.Done():
			m.flushOnce(context.Background())
			return nil
		}
	}
}

func (m *MobManager) flushOnce(ctx context.Context) {
	m.mu.Lock()
	if len(m.dirty) == 0 && len(m.deleted) == 0 {
		m.mu.Unlock()
		return
	}
	dirtyIDs := make([]string, 0, len(m.dirty))
	for id := range m.dirty {
		dirtyIDs = append(dirtyIDs, id)
	}
	deletedIDs := make([]string, 0, len(m.deleted))
	for id := range m.deleted {
		deletedIDs = append(deletedIDs, id)
	}

	snaps := make([]domain.Mob, 0, len(dirtyIDs))
	for _, id := range dirtyIDs {
		mob, ok := m.mobs[id]
		if !ok {
			delete(m.dirty, id)
			continue
		}
		snaps = append(snaps, *mob)
		delete(m.dirty, id)
	}
	for _, id := range deletedIDs {
		delete(m.deleted, id)
	}
	m.mu.Unlock()

	for i := range snaps {
		s := snaps[i]
		if _, err := m.store.SaveMob(ctx, &s); err != nil {
			log.Printf("failed to save mob %s: %v", s.ID, err)
			m.mu.Lock()
			if _, ok := m.mobs[s.ID]; ok {
				m.dirty[s.ID] = struct{}{}
			}
			m.mu.Unlock()
		}
	}
	for _, id := range deletedIDs {
		if err := m.store.DeleteMob(ctx, id); err != nil {
			log.Printf("failed to delete mob %s: %v", id, err)
			m.mu.Lock()
			m.deleted[id] = struct{}{}
			m.mu.Unlock()
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
