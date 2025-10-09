package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/LealKevin/terminus/internal/domain"
)

type PlayerMemoryStore struct {
	players map[string]*domain.Player
	mu      sync.RWMutex
}

type WorldMemoryStore struct {
	worlds map[string]*domain.World
	mu     sync.RWMutex
}

type MobMemoryStore struct {
	mobs map[string]*domain.Mob
	mu   sync.RWMutex
}

func NewPlayerMemoryStore() *PlayerMemoryStore {
	return &PlayerMemoryStore{
		players: map[string]*domain.Player{
			"1": {ID: "1", WorldID: "world1", X: 2, Y: 2, Health: 100, Attack: 50, Defense: 5, Range: 1},
		},
	}
}

func NewWorldMemoryStore() *WorldMemoryStore {
	return &WorldMemoryStore{
		worlds: map[string]*domain.World{
			"world1": {ID: "world1", Width: 53, Height: 25, Layout: domain.ConvertLayout(domain.Raw)},
		},
	}
}

func NewMobMemoryStore() *MobMemoryStore {
	return &MobMemoryStore{
		mobs: make(map[string]*domain.Mob),
	}
}

func (ms *WorldMemoryStore) NewWorld(id string, width, height int, layout domain.Layout) *domain.World {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	world := domain.NewWorld(id, width, height, layout)
	ms.worlds[id] = world
	return world
}

func (ms *WorldMemoryStore) GetWorld(ctx context.Context, id string) (*domain.World, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	world, ok := ms.worlds[id]
	if !ok {
		return nil, fmt.Errorf("world %s not found", id)
	}
	return world, nil
}

func (ms *PlayerMemoryStore) GetPlayer(ctx context.Context, id string) (*domain.Player, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	player, ok := ms.players[id]
	if !ok {
		return nil, nil
	}
	return player, nil
}

func (ms *PlayerMemoryStore) SavePlayer(ctx context.Context, player *domain.Player) (*domain.Player, error) {
	if player == nil {
		return nil, fmt.Errorf("player is nil")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.players[player.ID] = player
	return player, nil
}

func (ms *MobMemoryStore) GetMob(ctx context.Context, id string) (*domain.Mob, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	mob, ok := ms.mobs[id]
	if !ok {
		return nil, nil
	}
	return mob, nil
}

func (ms *MobMemoryStore) SaveMob(ctx context.Context, mob *domain.Mob) (*domain.Mob, error) {
	if mob == nil {
		return nil, fmt.Errorf("mob is nil")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.mobs[mob.ID] = mob
	return mob, nil
}

func (ms *MobMemoryStore) CreateMob(ctx context.Context, mob *domain.Mob) (*domain.Mob, error) {
	return ms.SaveMob(ctx, mob)
}

func (ms *MobMemoryStore) DeleteMob(ctx context.Context, id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.mobs, id)
	return nil
}

func (ms *MobMemoryStore) CountMobsInWorld(ctx context.Context, worldID string) (int, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	count := 0
	for _, mob := range ms.mobs {
		if mob.WorldID == worldID {
			count++
		}
	}
	return count, nil
}

func (ms *MobMemoryStore) GetMobsByWorld(ctx context.Context, worldID string) ([]*domain.Mob, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	var mobs []*domain.Mob
	for _, mob := range ms.mobs {
		if mob.WorldID == worldID {
			mobs = append(mobs, mob)
		}
	}
	fmt.Printf("Mobs in world %s: %+v\n", worldID, mobs)
	return mobs, nil
}
