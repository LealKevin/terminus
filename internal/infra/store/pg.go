package store

import (
	"context"
	"errors"
	"log"

	"github.com/LealKevin/terminus/internal/domain"
	"github.com/LealKevin/terminus/internal/infra/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	return &PlayerPgStore{db: q}
}

func NewWorldPgStore(q *db.Queries) *WorldPgStore {
	return &WorldPgStore{db: q}
}

func NewMobPgStore(q *db.Queries) *MobPgStore {
	return &MobPgStore{db: q}
}

// NewPgBundle wraps the Postgres-backed stores into a single bundle.
func NewPgBundle(q *db.Queries) *Bundle {
	return NewBundle(
		NewWorldPgStore(q),
		NewPlayerPgStore(q),
		NewMobPgStore(q),
	)
}

func (ws *WorldPgStore) CreateWorld(ctx context.Context, id string, width, height int, layout string) (*domain.World, error) {
	pgID, err := ensureUUID(&id)
	if err != nil {
		return nil, err
	}

	rec, err := ws.db.CreateWorld(ctx, db.CreateWorldParams{
		ID:     pgID,
		Width:  int32(width),
		Height: int32(height),
		Layout: layout,
	})
	if err != nil {
		return nil, err
	}

	return dbWorldToDomain(rec), nil
}

func (ws *WorldPgStore) GetWorld(ctx context.Context, id string) (*domain.World, error) {
	pgID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	rec, err := ws.db.GetWorldByID(ctx, pgID)
	if err != nil {
		return nil, err
	}

	return dbWorldToDomain(rec), nil
}

func (ps *PlayerPgStore) GetPlayer(ctx context.Context, id string) (*domain.Player, error) {
	log.Printf("DEBUG: GetPlayer called with id='%s' (length=%d)", id, len(id))
	pgID, err := parseUUID(id)
	if err != nil {
		log.Printf("DEBUG: parseUUID failed for id='%s': %v", id, err)
		return nil, err
	}

	rec, err := ps.db.GetPlayerByID(ctx, pgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return dbPlayerToDomain(rec), nil
}

func (ps *PlayerPgStore) SavePlayer(ctx context.Context, player *domain.Player) (*domain.Player, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}

	pgPlayerID, err := ensureUUID(&player.ID)
	if err != nil {
		return nil, err
	}

	pgWorldID, err := parseUUID(player.WorldID)
	if err != nil {
		return nil, err
	}

	rec, err := ps.db.UpdatePlayer(ctx, db.UpdatePlayerParams{
		ID:      pgPlayerID,
		WorldID: pgWorldID,
		X:       int32(player.X),
		Y:       int32(player.Y),
		Health:  int32(player.Health),
		Attack:  int32(player.Attack),
		Defense: int32(player.Defense),
		Range:   int32(player.Range),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		rec, err = ps.db.CreatePlayer(ctx, db.CreatePlayerParams{
			ID:      pgPlayerID,
			WorldID: pgWorldID,
			X:       int32(player.X),
			Y:       int32(player.Y),
			Health:  int32(player.Health),
			Attack:  int32(player.Attack),
			Defense: int32(player.Defense),
			Range:   int32(player.Range),
		})
	}
	if err != nil {
		return nil, err
	}

	return dbPlayerToDomain(rec), nil
}

func (ms *MobPgStore) GetMob(ctx context.Context, id string) (*domain.Mob, error) {
	pgID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	rec, err := ms.db.GetMobByID(ctx, pgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return dbMobToDomain(rec), nil
}

func (ms *MobPgStore) SaveMob(ctx context.Context, mob *domain.Mob) (*domain.Mob, error) {
	if mob == nil {
		return nil, errors.New("mob is nil")
	}

	pgMobID, err := ensureUUID(&mob.ID)
	if err != nil {
		return nil, err
	}

	pgWorldID, err := parseUUID(mob.WorldID)
	if err != nil {
		return nil, err
	}

	rec, err := ms.db.UpdateMob(ctx, db.UpdateMobParams{
		ID:          pgMobID,
		Name:        mob.Name,
		WorldID:     pgWorldID,
		X:           int32(mob.X),
		Y:           int32(mob.Y),
		Type:        mob.Type,
		Health:      int32(mob.Health),
		Attack:      int32(mob.Attack),
		Defense:     int32(mob.Defense),
		AttackSpeed: int32(mob.AttackSpeed),
		Symbol:      runeToString(mob.Symbol),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		created, createErr := ms.db.CreateMob(ctx, db.CreateMobParams{
			ID:          pgMobID,
			Name:        mob.Name,
			WorldID:     pgWorldID,
			X:           int32(mob.X),
			Y:           int32(mob.Y),
			Type:        mob.Type,
			Health:      int32(mob.Health),
			Attack:      int32(mob.Attack),
			Defense:     int32(mob.Defense),
			AttackSpeed: int32(mob.AttackSpeed),
			Symbol:      runeToString(mob.Symbol),
		})
		if createErr != nil {
			return nil, createErr
		}
		return dbMobCreateRowToDomain(created), nil
	}
	if err != nil {
		return nil, err
	}

	return dbMobUpdateRowToDomain(rec), nil
}

func (ms *MobPgStore) CreateMob(ctx context.Context, mob *domain.Mob) (*domain.Mob, error) {
	if mob == nil {
		return nil, errors.New("mob is nil")
	}

	pgWorldID, err := parseUUID(mob.WorldID)
	if err != nil {
		return nil, err
	}

	pgMobID, err := ensureUUID(&mob.ID)
	if err != nil {
		return nil, err
	}

	rec, err := ms.db.CreateMob(ctx, db.CreateMobParams{
		ID:          pgMobID,
		Name:        mob.Name,
		WorldID:     pgWorldID,
		X:           int32(mob.X),
		Y:           int32(mob.Y),
		Type:        mob.Type,
		Health:      int32(mob.Health),
		Attack:      int32(mob.Attack),
		Defense:     int32(mob.Defense),
		AttackSpeed: int32(mob.AttackSpeed),
		Symbol:      runeToString(mob.Symbol),
	})
	if err != nil {
		return nil, err
	}

	return dbMobCreateRowToDomain(rec), nil
}

func (ms *MobPgStore) DeleteMob(ctx context.Context, id string) error {
	pgID, err := parseUUID(id)
	if err != nil {
		return err
	}

	return ms.db.DeleteMob(ctx, pgID)
}

func (ms *MobPgStore) CountMobsInWorld(ctx context.Context, worldID string) (int, error) {
	pgWorldID, err := parseUUID(worldID)
	if err != nil {
		return 0, err
	}

	count, err := ms.db.CountMobsInWorld(ctx, pgWorldID)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

func (ms *MobPgStore) GetMobsByWorld(ctx context.Context, worldID string) ([]*domain.Mob, error) {
	pgWorldID, err := parseUUID(worldID)
	if err != nil {
		return nil, err
	}

	rows, err := ms.db.ListMobsInWorld(ctx, pgWorldID)
	if err != nil {
		return nil, err
	}

	mobs := make([]*domain.Mob, 0, len(rows))
	for _, row := range rows {
		mobs = append(mobs, dbMobRowToDomain(row))
	}

	return mobs, nil
}

func dbWorldToDomain(rec db.World) *domain.World {
	return &domain.World{
		ID:     rec.ID.String(),
		Width:  int(rec.Width),
		Height: int(rec.Height),
		Layout: domain.ConvertLayout(rec.Layout),
	}
}

func dbPlayerToDomain(rec db.Player) *domain.Player {
	return &domain.Player{
		ID:      rec.ID.String(),
		WorldID: rec.WorldID.String(),
		X:       int(rec.X),
		Y:       int(rec.Y),
		Health:  int(rec.Health),
		Attack:  int(rec.Attack),
		Defense: int(rec.Defense),
		Range:   int(rec.Range),
	}
}

func dbMobToDomain(rec db.Mob) *domain.Mob {
	return &domain.Mob{
		ID:          rec.ID.String(),
		Name:        rec.Name,
		WorldID:     rec.WorldID.String(),
		X:           int(rec.X),
		Y:           int(rec.Y),
		Type:        rec.Type,
		Health:      int(rec.Health),
		Attack:      int(rec.Attack),
		Defense:     int(rec.Defense),
		AttackSpeed: int(rec.AttackSpeed),
		Symbol:      firstRune(rec.Symbol),
	}
}

func dbMobRowToDomain(rec db.Mob) *domain.Mob {
	return &domain.Mob{
		ID:          rec.ID.String(),
		Name:        rec.Name,
		WorldID:     rec.WorldID.String(),
		X:           int(rec.X),
		Y:           int(rec.Y),
		Type:        rec.Type,
		Health:      int(rec.Health),
		Attack:      int(rec.Attack),
		Defense:     int(rec.Defense),
		AttackSpeed: int(rec.AttackSpeed),
		Symbol:      firstRune(rec.Symbol),
	}
}

func dbMobCreateRowToDomain(rec db.Mob) *domain.Mob {
	return &domain.Mob{
		ID:          rec.ID.String(),
		Name:        rec.Name,
		WorldID:     rec.WorldID.String(),
		X:           int(rec.X),
		Y:           int(rec.Y),
		Type:        rec.Type,
		Health:      int(rec.Health),
		Attack:      int(rec.Attack),
		Defense:     int(rec.Defense),
		AttackSpeed: int(rec.AttackSpeed),
		Symbol:      firstRune(rec.Symbol),
	}
}

func dbMobUpdateRowToDomain(rec db.Mob) *domain.Mob {
	return &domain.Mob{
		ID:          rec.ID.String(),
		Name:        rec.Name,
		WorldID:     rec.WorldID.String(),
		X:           int(rec.X),
		Y:           int(rec.Y),
		Type:        rec.Type,
		Health:      int(rec.Health),
		Attack:      int(rec.Attack),
		Defense:     int(rec.Defense),
		AttackSpeed: int(rec.AttackSpeed),
		Symbol:      firstRune(rec.Symbol),
	}
}

func parseUUID(id string) (pgtype.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: uid, Valid: true}, nil
}

func ensureUUID(id *string) (pgtype.UUID, error) {
	if id == nil {
		return pgtype.UUID{}, errors.New("nil id pointer")
	}

	if *id == "" {
		newID := uuid.New()
		*id = newID.String()
		return pgtype.UUID{Bytes: newID, Valid: true}, nil
	}

	return parseUUID(*id)
}

func runeToString(r rune) string {
	if r == 0 {
		return ""
	}
	return string(r)
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}
