package store

import (
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
			"1": {ID: "1", WorldID: "world1", X: 2, Y: 2},
		},
	}
}

func NewWorldMemoryStore() *WorldMemoryStore {
	return &WorldMemoryStore{
		worlds: map[string]*domain.World{
			"world1": {ID: "1", Width: 53, Height: 25, Layout: domain.ConvertLayout(domain.Raw)},
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

func (ms *WorldMemoryStore) GetWorld(id string) *domain.World {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.worlds[id]
}

func (ms *PlayerMemoryStore) GetPlayer(id string) *domain.Player {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.players[id]
}

func (ms *PlayerMemoryStore) SavePlayer(player *domain.Player) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.players[player.ID] = player
}

func (ms *MobMemoryStore) GetMob(id string) *domain.Mob {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.mobs[id]
}

func (ms *MobMemoryStore) SaveMob(mob *domain.Mob) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.mobs[mob.ID] = mob
}

func (ms *MobMemoryStore) CreateMob(mob *domain.Mob) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.mobs[mob.ID] = mob
}

func (ms *MobMemoryStore) DeleteMob(id string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.mobs, id)
}

func (ms *MobMemoryStore) CountMobsInWorld(worldID string) int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	count := 0
	for _, mob := range ms.mobs {
		if mob.WorldID == worldID {
			count++
		}
	}
	return count
}

func (ms *MobMemoryStore) GetMobsByWorld(worldID string) []*domain.Mob {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	var mobs []*domain.Mob
	for _, mob := range ms.mobs {
		if mob.WorldID == worldID {
			mobs = append(mobs, mob)
		}
	}
	return mobs
}
