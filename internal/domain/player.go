package domain

type Player struct {
	ID string
	Position
}

type Position struct {
	X float64
	Y float64
}

func NewPlayer(id string, x, y float64) *Player {
	return &Player{
		ID: id,
		Position: Position{
			X: x,
			Y: y,
		},
	}
}

func (p *Player) CanMove(x, y int, map )

