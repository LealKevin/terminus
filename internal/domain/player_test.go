package domain

import "testing"

func TestTakeDamage(t *testing.T) {
	tests := []struct {
		name       string
		health     int
		defense    int
		damage     int
		wantHealth int
	}{
		{"normal hit", 100, 5, 30, 75},
		{"damage below defense does not heal", 100, 5, 3, 100},
		{"equal damage and defense", 100, 5, 5, 100},
		{"lethal damage clamps to zero", 10, 0, 50, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Player{Health: tt.health, Defense: tt.defense}
			p.TakeDamage(tt.damage)
			if p.Health != tt.wantHealth {
				t.Errorf("Health = %d, want %d", p.Health, tt.wantHealth)
			}
		})
	}
}

func TestPlayerMove(t *testing.T) {
	world := NewWorld("w", 3, 3, ConvertLayout("###\n# #\n###"))

	p := &Player{X: 1, Y: 1}
	if err := p.Move("N", world); err == nil {
		t.Error("expected wall to block move N, got nil error")
	}
	if p.X != 1 || p.Y != 1 {
		t.Errorf("position changed on blocked move: (%d,%d)", p.X, p.Y)
	}

	if err := p.Move("X", world); err == nil {
		t.Error("expected error for invalid direction")
	}
}
