package store

import "github.com/LealKevin/terminus/internal/domain"

type MemoryStore struct {
	players map[string]*domain.Player
	worlds  map[string]*domain.World
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		players: make(map[string]*domain.Player),
		worlds:  make(map[string]*domain.World),
	}
}

func (ms *MemoryStore) NewWorld(id string, width, height int, layout domain.Layout) *domain.World {
	world := domain.NewWorld(id, width, height, layout)
	ms.worlds[id] = world
	return world
}

func (ms *MemoryStore) GetWorld(id string) *domain.World {
	return ms.worlds[id]
}
