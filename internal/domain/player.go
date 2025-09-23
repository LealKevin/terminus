package domain

import "fmt"

type PlayerStore interface {
	GetPlayer(id string) *Player
	SavePlayer(player *Player)
}

type Player struct {
	ID      string `json:"id"`
	WorldID string `json:"worldID"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Health  int    `json:"health"`
	Attack  int    `json:"attack"`
	Defense int    `json:"defense"`
	Range   int    `json:"range"`
}

func NewPlayer(id string, x, y int) *Player {
	return &Player{
		ID:      id,
		WorldID: "world1",
		X:       x,
		Y:       y,
		Health:  100,
		Attack:  50,
		Defense: 5,
		Range:   10,
	}
}

func (p *Player) canMove(x, y int, world *World) bool {
	if y < 0 || y >= len(world.Layout) {
		return false
	}
	if x < 0 || x >= len(world.Layout[y]) {
		return false
	}

	if world.Layout[y][x] == '#' {
		return false
	}
	return true
}

var deltas = map[string]struct{ dx, dy int }{
	"N":  {0, -1},
	"S":  {0, 1},
	"E":  {1, 0},
	"W":  {-1, 0},
	"NE": {1, -1},
	"NW": {-1, -1},
	"SE": {1, 1},
	"SW": {-1, 1},
}

func (p *Player) Move(direction string, world *World) error {
	d, ok := deltas[direction]
	if !ok {
		return fmt.Errorf("invalid direction %q", direction)
	}
	nx, ny := p.X+d.dx, p.Y+d.dy
	if !p.canMove(nx, ny, world) {
		return fmt.Errorf("cannot move %s", direction)
	}
	p.X, p.Y = nx, ny
	return nil
}

func (p *Player) TakeDamage(damage int) {
	damage -= p.Defense
	p.Health -= damage
	if p.Health < 0 {
		p.Health = 0
	}
}

func (p *Player) AttackMob(mob *Mob) {
	mob.TakeDamage(p.Attack)
}

func (p *Player) IsAlive() bool {
	return p.Health > 0
}

func (p *Player) SpawnPlayer(w *World, occupiedPositions map[string]bool) error {
	x, y, err := FindRandomSpawnPosition(w, occupiedPositions)
	if err != nil {
		return err
	}

	p.X = x
	p.Y = y
	return nil
}
