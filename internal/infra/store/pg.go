package store

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/LealKevin/terminus/internal/domain"
	"github.com/LealKevin/terminus/internal/infra/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type PlayerPgStore struct {
	db *db.Queries
}

type WorldPgStore struct {
	db *db.Queries
}

type MobPgStore struct {
	db *db.Queries
}

func NewPlayerPgStore(q *db.Queries) *PlayerPgStore {
	return &PlayerPgStore{
		db: q,
	}
}

func NewWorldPgStore(q *db.Queries) *WorldPgStore {
	return &WorldPgStore{
		db: q,
	}
}

func NewMobPgStore(q *db.Queries) *MobPgStore {
	return &MobPgStore{
		db: q,
	}
}

func (ms *WorldPgStore) CreateWorld(ctx context.Context, id string, width, height int, layout string) (*domain.World, error) {
	world, err := ms.db.CreateWorld(ctx, db.CreateWorldParams{
		Width:  int32(width),
		Height: int32(height),
		Layout: layout,
	})
	if err != nil {
		return nil, err
	}

	return &domain.World{
		ID:     world.ID.String(),
		Width:  int(world.Width),
		Height: int(world.Height),
		Layout: domain.ConvertLayout(world.Layout),
	}, nil
}

func (ms *WorldPgStore) GetWorld(ctx context.Context, id string) (*domain.World, error) {
	uid, err := uuid.Parse(id)
	world, err := ms.db.GetWorldByID(ctx, pgtype.UUID{
		Bytes: uid,
		Valid: true,
	})
	if err != nil {
		slog.Error("error getting world", "err", err)
		return nil, err
	}
	return &domain.World{
		ID:     world.ID.String(),
		Width:  int(world.Width),
		Height: int(world.Height),
		Layout: domain.ConvertLayout(world.Layout),
	}, nil
}

func (ms *PlayerPgStore) GetPlayer(ctx context.Context, id string) (*domain.Player, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	player, err := ms.db.GetPlayerByID(ctx, pgtype.UUID{
		Bytes: uid,
		Valid: true,
	})
	if err != nil {
		return nil, err
	}
	return &domain.Player{
		ID:      player.ID.String(),
		WorldID: player.WorldID.String(),
		X:       int(player.X),
		Y:       int(player.Y),
	}, nil
}

func (ms *PlayerPgStore) SavePlayer(ctx context.Context, player *domain.Player) (*domain.Player, error) {
	uid, err := uuid.Parse(player.ID)
	if err != nil {
		return nil, err
	}
	worldUID, err := uuid.Parse(player.WorldID)
	if err != nil {
		return nil, err
	}

	data, err := ms.db.UpdatePlayer(ctx, db.UpdatePlayerParams{
		WorldID: pgtype.UUID{
			Bytes: worldUID,
			Valid: true,
		},
		X: int32(player.X),
		Y: int32(player.Y),
		ID: pgtype.UUID{
			Bytes: uid,
			Valid: true,
		},
		Attack:  int32(player.Attack),
		Health:  int32(player.Health),
		Defense: int32(player.Defense),
		Range:   int32(player.Range),
	})
	if err != nil {
		return nil, err
	}
	return &domain.Player{
		ID:      data.ID.String(),
		WorldID: data.WorldID.String(),
		X:       int(data.X),
		Y:       int(data.Y),
		Attack:  int(data.Attack),
		Health:  int(data.Health),
		Defense: int(data.Defense),
		Range:   int(data.Range),
	}, nil
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
	fmt.Printf("Mobs in world %s: %+v\n", worldID, mobs)
	return mobs
}
