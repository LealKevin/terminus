package domain

import "context"

type MobStore interface {
	GetMob(ctx context.Context, id string) (*Mob, error)
	SaveMob(ctx context.Context, mob *Mob) (*Mob, error)
	CreateMob(ctx context.Context, mob *Mob) (*Mob, error)
	DeleteMob(ctx context.Context, id string) error
	GetMobsByWorld(ctx context.Context, worldID string) ([]*Mob, error)
}

type Mob struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	WorldID     string `json:"worldID"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Type        string `json:"type"`
	Health      int    `json:"health"`
	Attack      int    `json:"attack"`
	Defense     int    `json:"defense"`
	AttackSpeed int    `json:"attackSpeed"`
	Symbol      rune   `json:"symbol"`
}

func (m *Mob) canMove(x, y int, world *World) bool {
	if y < 0 || y >= len(world.Layout) {
		return false
	}
	if x < 0 || x >= len(world.Layout[y]) {
		return false
	}

	if world.Layout[y][x] == '#' {
		return false
	}

	if world.Layout[y][x] == '@' {
		return false
	}

	return true
}

var mobDeltas = map[string]struct{ dx, dy int }{
	"N":  {0, -1},
	"S":  {0, 1},
	"E":  {1, 0},
	"W":  {-1, 0},
	"NE": {1, -1},
	"NW": {-1, -1},
	"SE": {1, 1},
	"SW": {-1, 1},
}

func (m *Mob) Move(direction string, world *World) error {
	d, ok := mobDeltas[direction]
	if !ok {
		return nil
	}
	nx, ny := m.X+d.dx, m.Y+d.dy
	if !m.canMove(nx, ny, world) {
		return nil
	}
	m.X, m.Y = nx, ny
	return nil
}

func (m *Mob) IsAlive() bool {
	return m.Health > 0
}

func (m *Mob) TakeDamage(damage int) {
	m.Health -= netDamage(damage, m.Defense)
	if m.Health < 0 {
		m.Health = 0
	}
}

func netDamage(damage, defense int) int {
	if d := damage - defense; d > 0 {
		return d
	}
	return 0
}

func (m *Mob) AttackTarget(target *Player) {
	target.TakeDamage(m.Attack)
}
