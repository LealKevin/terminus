package app

import (
	"context"
	"sync"

	"github.com/LealKevin/terminus/internal/domain"
)

type worldCache struct {
	store domain.WorldStore
	mu    sync.RWMutex
	world *domain.World
}

func newWorldCache(store domain.WorldStore) *worldCache {
	return &worldCache{store: store}
}

func (c *worldCache) Get(ctx context.Context, worldID string) (*domain.World, error) {
	c.mu.RLock()
	if c.world != nil && c.world.ID == worldID {
		w := c.world
		c.mu.RUnlock()
		return w, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.world != nil && c.world.ID == worldID {
		return c.world, nil
	}

	w, err := c.store.GetWorld(ctx, worldID)
	if err != nil {
		return nil, err
	}
	c.world = w
	return w, nil
}
