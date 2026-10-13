package store

import (
	"github.com/LealKevin/terminus/internal/domain"
)

type Bundle struct {
	world  domain.WorldStore
	player domain.PlayerStore
	mob    domain.MobStore
}

func NewBundle(world domain.WorldStore, player domain.PlayerStore, mob domain.MobStore) *Bundle {
	return &Bundle{world: world, player: player, mob: mob}
}

func NewMemoryBundle() *Bundle {
	return NewBundle(NewWorldMemoryStore(), NewPlayerMemoryStore(), NewMobMemoryStore())
}

func (b *Bundle) Worlds() domain.WorldStore {
	return b.world
}

func (b *Bundle) Players() domain.PlayerStore {
	return b.player
}

func (b *Bundle) Mobs() domain.MobStore {
	return b.mob
}
